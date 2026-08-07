package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	bucket := os.Getenv("BUCKET")
	filename := request.QueryStringParameters["name"]
	if filename == "" {
		filename = "receipt.pdf"
	}
	key := "raw/" + filename

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return serverError(err)
	}

	client := s3.NewFromConfig(cfg)
	presigner := s3.NewPresignClient(client)

	presigned, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		ContentType: strPtr("application/pdf"),
	}, s3.WithPresignExpires(300))
	if err != nil {
		return serverError(err)
	}

	body, _ := json.Marshal(map[string]string{
		"uploadUrl": presigned.URL,
		"key":       key,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: 500, Body: err.Error()}, nil
}

func strPtr(s string) *string { return &s }

func main() {
	lambda.Start(handler)
}
