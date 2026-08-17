package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var errDuplicate = errors.New("duplicate receipt, already processed")

type hashItem struct {
	ContentHash string `dynamodbav:"content_hash"`
	CreatedAt   string `dynamodbav:"created_at"`
}

// claimHash atomically reserves content_hash in the hashes table. It returns
// true when this invocation won the claim (no duplicate) and false when the
// content was already claimed by a previous run.
func claimHash(ctx context.Context, client *dynamodb.Client, table, hash string) (bool, error) {
	item, err := attributevalue.MarshalMap(hashItem{
		ContentHash: hash,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return false, fmt.Errorf("marshal: %w", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      item,
		ConditionExpression: aws.String("attribute_not_exists(content_hash)"),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return false, nil
		}
		return false, fmt.Errorf("put item: %w", err)
	}

	return true, nil
}

// releaseHash removes a claim so a failed run can be retried.
func releaseHash(ctx context.Context, client *dynamodb.Client, table, hash string) error {
	_, err := client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(table),
		Key: map[string]types.AttributeValue{
			"content_hash": &types.AttributeValueMemberS{Value: hash},
		},
	})
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	return nil
}
