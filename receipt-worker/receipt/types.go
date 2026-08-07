package receipt

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
	Company         string
	Store           string
	Date            string
	Hour            string
	PaymentMethod   string
	Total           float64
	SavingCardUsed  bool
	TotalSavingsAcc float64
	ClientCard      string
	Items           []Item
	VatCategories   []VAT
	VatNumber       string
	EmailSubj       string
	EmailDate       string
}
