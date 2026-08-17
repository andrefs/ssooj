package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"ssooj/receipt-worker/receipt"
	"ssooj/receipt-worker/store"
)

type env struct {
	RawBucket      string
	ProcessedBucket string
	DynamoTable    string
	HashTable      string
}

func loadEnv() env {
	return env{
		RawBucket:       os.Getenv("RAW_BUCKET"),
		ProcessedBucket: os.Getenv("PROCESSED_BUCKET"),
		DynamoTable:     os.Getenv("DYNAMO_TABLE"),
		HashTable:       os.Getenv("HASH_TABLE"),
	}
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	e := loadEnv()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	ddbClient := dynamodb.NewFromConfig(cfg)
	dw := &store.DynamoWriter{Client: ddbClient, Table: e.DynamoTable}

	for _, msg := range sqsEvent.Records {
		if err := processMessage(ctx, msg, e, s3Client, ddbClient, dw); err != nil {
			log.Printf("ERROR processing message %s: %v", msg.MessageId, err)
			return err
		}
	}

	return nil
}

func processMessage(ctx context.Context, msg events.SQSMessage, e env, s3Client *s3.Client, ddbClient *dynamodb.Client, dw *store.DynamoWriter) error {
	var s3Event events.S3Event
	if err := json.Unmarshal([]byte(msg.Body), &s3Event); err != nil {
		return fmt.Errorf("unmarshal S3 event: %w", err)
	}

	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key

		log.Printf("processing s3://%s/%s", bucket, key)

		text, contentHash, err := extractText(ctx, s3Client, bucket, key)
		if err != nil {
			return fmt.Errorf("extract text from %s: %w", key, err)
		}

		claimed, err := claimHash(ctx, ddbClient, e.HashTable, contentHash)
		if err != nil {
			return fmt.Errorf("claim hash for %s: %w", key, err)
		}
		if !claimed {
			log.Printf("skipping %s: duplicate receipt (hash %s)", key, contentHash)
			continue
		}

		if err := process(ctx, s3Client, e, dw, key, text); err != nil {
			if relErr := releaseHash(ctx, ddbClient, e.HashTable, contentHash); relErr != nil {
				log.Printf("ERROR releasing claim for %s: %v", key, relErr)
			}
			return fmt.Errorf("process %s: %w", key, err)
		}
	}

	return nil
}

func process(ctx context.Context, s3Client *s3.Client, e env, dw *store.DynamoWriter, key, text string) error {
	r, err := receipt.Parse(text)
	if err != nil {
		return fmt.Errorf("parse receipt from %s: %w", key, err)
	}

	log.Printf("parsed: store=%s date=%s items=%d total=%.2f valid=%v",
		r.Store, r.Date, len(r.Items), r.Total, r.Valid())

	if err := storeCSV(ctx, s3Client, e.ProcessedBucket, key, r); err != nil {
		return fmt.Errorf("store csv: %w", err)
	}

	if err := dw.Write(ctx, r); err != nil {
		return fmt.Errorf("store dynamo: %w", err)
	}

	return nil
}

func extractText(ctx context.Context, s3Client *s3.Client, bucket, key string) (string, string, error) {
	tmpDir := "/tmp"
	pdfPath := filepath.Join(tmpDir, "receipt.pdf")
	txtPath := pdfPath + ".txt"

	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return "", "", fmt.Errorf("s3 get: %w", err)
	}
	defer result.Body.Close()

	out, err := os.Create(pdfPath)
	if err != nil {
		return "", "", fmt.Errorf("create tmp file: %w", err)
	}
	defer out.Close()

	h := sha256.New()
	if _, err := out.ReadFrom(io.TeeReader(result.Body, h)); err != nil {
		return "", "", fmt.Errorf("write tmp file: %w", err)
	}
	out.Close()

	contentHash := hex.EncodeToString(h.Sum(nil))

	pdftotextBin := "/opt/bin/pdftotext"
	if _, err := os.Stat(pdftotextBin); os.IsNotExist(err) {
		pdftotextBin = "pdftotext"
	}
	cmd := exec.Command(pdftotextBin, "-layout", pdfPath, txtPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("pdftotext: %s: %w", string(output), err)
	}

	data, err := os.ReadFile(txtPath)
	if err != nil {
		return "", "", fmt.Errorf("read output: %w", err)
	}

	return string(data), contentHash, nil
}

func storeCSV(ctx context.Context, s3Client *s3.Client, bucket, key string, r *receipt.Receipt) error {
	csvData, err := store.ToCSV(r)
	if err != nil {
		return fmt.Errorf("to csv: %w", err)
	}

	csvKey := "csv/" + key + ".csv"

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &csvKey,
		Body:   bytes.NewReader(csvData),
	})
	if err != nil {
		return fmt.Errorf("s3 put: %w", err)
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
