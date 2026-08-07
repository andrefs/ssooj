package receipt

import "math"

type Item struct {
	Category          string
	VatCategory       string
	Description       string
	UnitQuantityUnit  string
	UnitQuantityValue float64
	UnitValue         float64
	Quantity          float64
	TotalValue        float64
	Savings           float64
}

type VAT struct {
	Category string
	Value    float64
	Net      float64
	Gross    float64
}

type Receipt struct {
	Company          string
	Store            string
	Date             string
	Hour             string
	PaymentMethod    string
	Total            float64
	ItemsTotal       float64
	CardDiscount     float64
	TotalDiscrepancy float64
	SavingCardUsed   bool
	TotalSavingsAcc  float64
	ClientCard       string
	Items            []Item
	VatCategories    []VAT
	VatNumber        string
	EmailSubj        string
	EmailDate        string
}

func (r *Receipt) Valid() bool {
	switch {
	case len(r.Items) == 0:
		return false
	case r.Total <= 0:
		return false
	case r.Company == "" && r.Store == "":
		return false
	case len(r.VatCategories) == 0 && r.VatNumber == "":
		return false
	}

	expected := r.ItemsTotal - r.CardDiscount
	diff := math.Abs(r.Total - expected)
	if diff > 0.02 {
		return false
	}

	return true
}

 
