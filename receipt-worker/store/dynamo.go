package store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"

	"ssooj/receipt-worker/receipt"
)

type DynamoWriter struct {
	Client *dynamodb.Client
	Table  string
}

type receiptItem struct {
	PK               string             `dynamodbav:"receipt_id"`
	Company          string             `dynamodbav:"company"`
	Store            string             `dynamodbav:"store"`
	Date             string             `dynamodbav:"date"`
	Hour             string             `dynamodbav:"hour"`
	PaymentMethod    string             `dynamodbav:"payment_method"`
	Total            float64            `dynamodbav:"total"`
	CardDiscount     float64            `dynamodbav:"card_discount"`
	TotalSavingsAcc  float64            `dynamodbav:"total_savings_acc"`
	ClientCard       string             `dynamodbav:"client_card"`
	VatNumber        string             `dynamodbav:"vat_number"`
	ItemsTotal       float64            `dynamodbav:"items_total"`
	TotalDiscrepancy float64            `dynamodbav:"total_discrepancy"`
	Valid            bool               `dynamodbav:"valid"`
	Items            []receipt.Item     `dynamodbav:"items"`
	VatCategories    []receipt.VAT      `dynamodbav:"vat_categories"`
}

func (w *DynamoWriter) Write(ctx context.Context, r *receipt.Receipt) error {
	id := uuid.New().String()

	item := receiptItem{
		PK:               id,
		Company:          r.Company,
		Store:            r.Store,
		Date:             r.Date,
		Hour:             r.Hour,
		PaymentMethod:    r.PaymentMethod,
		Total:            r.Total,
		CardDiscount:     r.CardDiscount,
		TotalSavingsAcc:  r.TotalSavingsAcc,
		ClientCard:       r.ClientCard,
		VatNumber:        r.VatNumber,
		ItemsTotal:       r.ItemsTotal,
		TotalDiscrepancy: r.TotalDiscrepancy,
		Valid:            r.Valid(),
		Items:            r.Items,
		VatCategories:    r.VatCategories,
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	_, err = w.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(w.Table),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("put item: %w", err)
	}

	return nil
}
