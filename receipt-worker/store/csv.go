package store

import (
	"encoding/csv"
	"fmt"
	"strings"

	"ssooj/receipt-worker/receipt"
)

func ToCSV(r *receipt.Receipt) ([]byte, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	header := []string{
		"receipt_id", "company", "store", "date", "hour",
		"payment_method", "total", "card_discount", "total_savings_acc",
		"client_card", "vat_number",
		"category", "vat_category", "description",
		"quantity", "unit_value", "total_value", "savings",
	}
	if err := w.Write(header); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}

	for _, item := range r.Items {
		row := []string{
			r.ClientCard, r.Company, r.Store, r.Date, r.Hour,
			r.PaymentMethod, fmt.Sprintf("%.2f", r.Total),
			fmt.Sprintf("%.2f", r.CardDiscount),
			fmt.Sprintf("%.2f", r.TotalSavingsAcc),
			r.ClientCard, r.VatNumber,
			item.Category, item.VatCategory, item.Description,
			fmt.Sprintf("%g", item.Quantity),
			fmt.Sprintf("%.2f", item.UnitValue),
			fmt.Sprintf("%.2f", item.TotalValue),
			fmt.Sprintf("%.2f", item.Savings),
		}
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("write row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}
