package store

import (
	"strings"
	"testing"

	"ssooj/receipt-worker/receipt"
)

func TestToCSV(t *testing.T) {
	r := &receipt.Receipt{
		Company:   "Test Co",
		Store:     "Test Store",
		Date:      "01/01/2026",
		Hour:      "10:00",
		Total:     10.50,
		ClientCard: "XXXX1234",
		VatNumber: "PT000000000",
		Items: []receipt.Item{
			{
				Category:    "Cat A",
				VatCategory: "C",
				Description: "Item desc",
				Quantity:    2,
				UnitValue:   1.25,
				TotalValue:  2.50,
				Savings:     0.50,
			},
		},
	}

	csv, err := ToCSV(r)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(csv)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + 1 row), got %d", len(lines))
	}

	header := lines[0]
	if !strings.Contains(header, "receipt_id") || !strings.Contains(header, "description") {
		t.Errorf("unexpected header: %s", header)
	}

	row := lines[1]
	if !strings.Contains(row, "Item desc") {
		t.Errorf("row missing description: %s", row)
	}
	if !strings.Contains(row, "2.50") {
		t.Errorf("row missing total value: %s", row)
	}
}
