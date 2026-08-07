package receipt

import (
	"os"
	"testing"
)

func loadTestData(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestParseSample1(t *testing.T) {
	raw := loadTestData(t, "sample1.txt")
	r, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if r.Company != "Modelo Continente Test Co" {
		t.Errorf("company = %q, want %q", r.Company, "Modelo Continente Test Co")
	}
	if r.Store != "Test Store" {
		t.Errorf("store = %q, want %q", r.Store, "Test Store")
	}
	if r.Date != "01/01/2026" {
		t.Errorf("date = %q, want %q", r.Date, "01/01/2026")
	}
	if r.Hour != "10:00" {
		t.Errorf("hour = %q, want %q", r.Hour, "10:00")
	}
	if r.PaymentMethod != "Credito" {
		t.Errorf("payment = %q, want %q", r.PaymentMethod, "Credito")
	}
	if r.Total != 5.00 {
		t.Errorf("total = %.2f, want %.2f", r.Total, 5.00)
	}
	if r.ItemsTotal != 6.00 {
		t.Errorf("items total = %.2f, want %.2f", r.ItemsTotal, 6.00)
	}
	if r.CardDiscount != 1.00 {
		t.Errorf("card discount = %.2f, want %.2f", r.CardDiscount, 1.00)
	}
	if r.TotalSavingsAcc != 0.50 {
		t.Errorf("total savings = %.2f, want %.2f", r.TotalSavingsAcc, 0.50)
	}
	if r.VatNumber != "PT000000000" {
		t.Errorf("vat number = %q, want %q", r.VatNumber, "PT000000000")
	}
	if r.ClientCard != "XXXXXXXX0000X" {
		t.Errorf("client card = %q, want %q", r.ClientCard, "XXXXXXXX0000X")
	}
	if !r.SavingCardUsed {
		t.Error("saving card should be used")
	}
	if !r.Valid() {
		t.Error("receipt should be valid")
	}

	if len(r.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(r.Items))
	}

	item0 := r.Items[0]
	if item0.Category != "Category A" {
		t.Errorf("item[0] category = %q", item0.Category)
	}
	if item0.VatCategory != "C" {
		t.Errorf("item[0] vat = %q", item0.VatCategory)
	}
	if item0.Description != "ITEM ONE DESCRIPTION" {
		t.Errorf("item[0] desc = %q", item0.Description)
	}
	if item0.TotalValue != 1.00 {
		t.Errorf("item[0] value = %.2f", item0.TotalValue)
	}
	if item0.Savings != 0 {
		t.Errorf("item[0] savings = %.2f", item0.Savings)
	}

	item2 := r.Items[2]
	if item2.Savings != 0.50 {
		t.Errorf("item[2] savings = %.2f, want 0.50", item2.Savings)
	}

	if len(r.VatCategories) != 2 {
		t.Fatalf("got %d vat categories, want 2", len(r.VatCategories))
	}
	if r.VatCategories[0].Category != "A" || r.VatCategories[0].Value != 6 {
		t.Errorf("vat[0] = %+v", r.VatCategories[0])
	}
	if r.VatCategories[1].Category != "C" || r.VatCategories[1].Value != 23 {
		t.Errorf("vat[1] = %+v", r.VatCategories[1])
	}
}

func TestParseSample2(t *testing.T) {
	raw := loadTestData(t, "sample2.txt")
	r, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if r.Total != 1.80 {
		t.Errorf("total = %.2f, want %.2f", r.Total, 1.80)
	}
	if r.ItemsTotal != 1.80 {
		t.Errorf("items total = %.2f, want %.2f", r.ItemsTotal, 1.80)
	}
	if r.TotalDiscrepancy != 0 {
		t.Errorf("discrepancy = %.2f, want 0", r.TotalDiscrepancy)
	}
	if r.SavingCardUsed {
		t.Error("saving card should not be used")
	}
	if !r.Valid() {
		t.Error("receipt should be valid")
	}

	if len(r.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(r.Items))
	}

	if r.Items[0].Quantity != 0.5 {
		t.Errorf("item[0] qty = %.3f, want 0.5", r.Items[0].Quantity)
	}
	if r.Items[0].UnitValue != 2.00 {
		t.Errorf("item[0] unit = %.2f, want 2.00", r.Items[0].UnitValue)
	}
	if r.Items[0].TotalValue != 1.00 {
		t.Errorf("item[0] total = %.2f, want 1.00", r.Items[0].TotalValue)
	}

	if len(r.VatCategories) != 1 {
		t.Fatalf("got %d vat categories, want 1", len(r.VatCategories))
	}
	if r.VatCategories[0].Category != "A" {
		t.Errorf("vat category = %q", r.VatCategories[0].Category)
	}
}

func TestParseSample3(t *testing.T) {
	raw := loadTestData(t, "sample3.txt")
	r, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if r.Total != 5.29 {
		t.Errorf("total = %.2f, want %.2f", r.Total, 5.29)
	}
	if r.ItemsTotal != 5.29 {
		t.Errorf("items total = %.2f, want %.2f", r.ItemsTotal, 5.29)
	}
	if !r.Valid() {
		t.Error("receipt should be valid")
	}

	if len(r.Items) != 5 {
		t.Fatalf("got %d items, want 5", len(r.Items))
	}

	if r.Items[2].Savings != 0.20 {
		t.Errorf("item[2] savings = %.2f, want 0.20 (DESCONTO DIRETO)", r.Items[2].Savings)
	}

	if r.Items[3].VatCategory != "NS" {
		t.Errorf("item[3] vat = %q, want NS", r.Items[3].VatCategory)
	}
	if r.Items[3].Description != "DEPOSIT ITEM" {
		t.Errorf("item[3] desc = %q", r.Items[3].Description)
	}
	if r.Items[3].Quantity != 3 {
		t.Errorf("item[3] qty = %.0f, want 3", r.Items[3].Quantity)
	}

	if len(r.VatCategories) != 3 {
		t.Fatalf("got %d vat categories, want 3", len(r.VatCategories))
	}
	if r.VatCategories[0].Category != "NS" {
		t.Errorf("vat[0] = %+v, want NS", r.VatCategories[0])
	}
	if r.VatCategories[1].Category != "A" {
		t.Errorf("vat[1] = %+v, want A", r.VatCategories[1])
	}
	if r.VatCategories[2].Category != "C" {
		t.Errorf("vat[2] = %+v, want C", r.VatCategories[2])
	}
}
