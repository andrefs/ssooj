package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func corsHeaders(origin string) map[string]string {
	h := map[string]string{
		"Content-Type":                "application/json",
		"Access-Control-Allow-Origin": "*",
		"Access-Control-Allow-Methods": "POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type",
	}
	if origin != "" {
		h["Access-Control-Allow-Origin"] = origin
	}
	return h
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	origin := request.Headers["origin"]
	if origin == "" {
		origin = request.Headers["Origin"]
	}
	headers := corsHeaders(origin)

	if request.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{
			StatusCode: 204,
			Headers:    headers,
		}, nil
	}

	bucket := os.Getenv("BUCKET")
	filename := request.QueryStringParameters["name"]
	if filename == "" {
		filename = "receipt.pdf"
	}
	key := "raw/" + filename

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return serverError(err, headers)
	}

	client := s3.NewFromConfig(cfg)
	presigner := s3.NewPresignClient(client)

	presigned, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		ContentType: strPtr("application/pdf"),
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return serverError(err, headers)
	}

	body, _ := json.Marshal(map[string]string{
		"uploadUrl": presigned.URL,
		"key":       key,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       string(body),
	}, nil
}

func serverError(err error, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 500,
		Headers:   headers,
		Body:      err.Error(),
	}, nil
}

func strPtr(s string) *string { return &s }

func main() {
	lambda.Start(handler)
}
