package receipt

import "testing"

func TestValid(t *testing.T) {
	tests := []struct {
		name string
		r    Receipt
		want bool
	}{
		{
			name: "empty items",
			r:    Receipt{Total: 10, Company: "C", Store: "S", VatNumber: "V"},
			want: false,
		},
		{
			name: "zero total",
			r:    Receipt{Total: 0, Items: []Item{{}}, Company: "C", Store: "S", VatNumber: "V"},
			want: false,
		},
		{
			name: "no company or store",
			r:    Receipt{Total: 10, Items: []Item{{}}},
			want: false,
		},
		{
			name: "no vat info",
			r:    Receipt{Total: 10, Items: []Item{{}}, Company: "C", Store: "S"},
			want: false,
		},
		{
			name: "mismatch total",
			r: Receipt{
				Total:      10,
				Items:      []Item{{TotalValue: 5}},
				ItemsTotal: 5,
				Company:    "C", Store: "S", VatNumber: "V",
			},
			want: false,
		},
		{
			name: "valid with card discount",
			r: Receipt{
				Total:        10,
				Items:        []Item{{TotalValue: 15}},
				ItemsTotal:   15,
				CardDiscount: 5,
				Company:      "C", Store: "S", VatNumber: "V",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Valid()
			if got != tt.want {
				t.Errorf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
