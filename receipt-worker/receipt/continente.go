package receipt

import (
	"regexp"
	"strings"
)

func init() {
	Register(continenteParser{})
}

type continenteParser struct{}

var (
	// -- building blocks --
	v       = `\(([A-Z])\)`         // VAT category letter e.g. (A), (C)
	val     = `(\d+[,]\d{2})`       // monetary value e.g. 1,59
	qty     = `(\d+[,]?\d*)`        // quantity (may be decimal) e.g. 2 or 0,715
	sp      = `\s+`                 // one or more spaces
	spLong  = `\s{2,}`              // two or more spaces (separates description from value)
	price   = val                   // alias: a value at end-of-line
	unitPrice = val                 // price per unit
	netVal  = val                   // VAT net value
	vatVal  = val                   // VAT percentage value (like 6,00%)
	ivaVal  = val                   // VAT amount
	total   = val                   // line total

	// -- composed patterns --
	reContinenteItem    = regexp.MustCompile(`^` + v + sp + `(.+?)` + spLong + price + `$`)
	reContinenteItemQty = regexp.MustCompile(`^` + sp + qty + sp + `X` + sp + unitPrice + spLong + total + `$`)
	reContinenteSavings = regexp.MustCompile(`^( )?POUPANCA` + sp + price + `$`)
	reContinenteDiscount = regexp.MustCompile(`^DESCONTO DIRETO` + sp + price + `$`)
	reContinenteDate    = regexp.MustCompile(`Nro:FS\s+\S+` + sp + `(\d{2}/\d{2}/\d{4})` + sp + `(\d{2}:\d{2})`)
	reContinenteTotal   = regexp.MustCompile(`^TOTAL A PAGAR` + sp + price + `$`)
	reContinenteSubtotal = regexp.MustCompile(`^SUBTOTAL` + sp + price + `$`)
	reContinentePay     = regexp.MustCompile(`^Cartao` + sp + `(.+?)` + spLong + price + `$`)
	reContinenteVatHdr  = regexp.MustCompile(`^\s*%IVA` + sp + `Total Liq`)
	reContinenteVatLine = regexp.MustCompile(`^` + v + sp + vatVal + `%` + sp + netVal + sp + ivaVal + sp + total + `$`)
	reContinenteCardDisc = regexp.MustCompile(`^Desconto Cartao Utilizado` + sp + price + `$`)
	reContinenteCard    = regexp.MustCompile(`Cartao cliente nº\s+(\S+)`)
	reContinenteNIF     = regexp.MustCompile(`NIF:\s*PT(\d{9})`)
	reContinenteCompra  = regexp.MustCompile(`COMPRA` + sp + price + `€`)
	reContinenteUsedCard = regexp.MustCompile(`UTILIZOU DO SEU CARTAO` + sp + price + `€`)
	reContinenteSavAcc  = regexp.MustCompile(`Total de descontos e poupancas` + sp + price + `$`)
)

func (continenteParser) Detect(lines []string) bool {
	for _, line := range lines {
		if strings.Contains(line, "Modelo Continente") {
			return true
		}
	}
	return false
}

func (continenteParser) Parse(lines []string, r *Receipt) (*Receipt, error) {
	var currentCategory string
	var pendingItem *Item
	inVATSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if r.Store == "" && strings.HasPrefix(line, "                       ") && !strings.Contains(trimmed, ":") && len(trimmed) < 40 {
			r.Store = trimmed
			continue
		}

		if strings.Contains(trimmed, "Modelo Continente") && r.Company == "" {
			r.Company = trimmed
			continue
		}

		if m := reContinenteNIF.FindStringSubmatch(trimmed); m != nil && r.VatNumber == "" {
			r.VatNumber = "PT" + m[1]
		}

		if m := reContinenteDate.FindStringSubmatch(trimmed); m != nil {
			r.Date = m[1]
			r.Hour = m[2]
			continue
		}

		if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, "IVA") && !strings.Contains(trimmed, "Nro:") {
			if pendingItem != nil {
				r.Items = append(r.Items, *pendingItem)
				pendingItem = nil
			}
			currentCategory = strings.TrimSuffix(trimmed, ":")
			continue
		}

		if strings.HasPrefix(trimmed, "(") && !reContinenteItem.MatchString(trimmed) && !reContinenteVatLine.MatchString(trimmed) && !strings.Contains(trimmed, "%") {
			if pendingItem != nil {
				r.Items = append(r.Items, *pendingItem)
			}
			pendingItem = &Item{
				VatCategory: string(trimmed[1]),
				Description: strings.TrimSpace(trimmed[4:]),
				Category:    currentCategory,
			}
			continue
		}

		if strings.HasPrefix(trimmed, "NS") && !inVATSection && !reContinenteItem.MatchString(trimmed) {
			if pendingItem != nil {
				r.Items = append(r.Items, *pendingItem)
			}
			pendingItem = &Item{
				VatCategory: "NS",
				Description: strings.TrimSpace(trimmed[2:]),
				Category:    currentCategory,
			}
			continue
		}

		if reContinenteVatHdr.MatchString(trimmed) {
			inVATSection = true
			continue
		}

		if inVATSection {
			if m := reContinenteVatLine.FindStringSubmatch(trimmed); m != nil {
				r.VatCategories = append(r.VatCategories, VAT{
					Category: m[1],
					Value:    parseFloat(m[2]),
					Net:      parseFloat(m[3]),
					Gross:    parseFloat(m[5]),
				})
				continue
			}
			if strings.HasPrefix(trimmed, "NS") {
				parts := strings.Fields(trimmed)
				if len(parts) == 4 {
					r.VatCategories = append(r.VatCategories, VAT{
						Category: "NS",
						Value:    0,
						Net:      parseFloat(parts[1]),
						Gross:    parseFloat(parts[3]),
					})
				}
				continue
			}
			if !reContinenteVatLine.MatchString(trimmed) && !strings.HasPrefix(trimmed, "(") && !strings.HasPrefix(trimmed, "NS") {
				inVATSection = false
			}
			continue
		}

		if m := reContinenteItem.FindStringSubmatch(trimmed); m != nil {
			if pendingItem != nil {
				r.Items = append(r.Items, *pendingItem)
			}
			pendingItem = &Item{
				VatCategory: m[1],
				Description: strings.TrimSpace(m[2]),
				TotalValue:  parseFloat(m[3]),
				Category:    currentCategory,
			}
			continue
		}

		if pendingItem != nil {

			if m := reContinenteItemQty.FindStringSubmatch(line); m != nil {
				pendingItem.Quantity = parseFloat(m[1])
				pendingItem.UnitValue = parseFloat(m[2])
				pendingItem.TotalValue = parseFloat(m[3])
				continue
			}

			if m := reContinenteSavings.FindStringSubmatch(trimmed); m != nil {
				pendingItem.Savings = parseFloat(m[2])
				continue
			}
			if m := reContinenteDiscount.FindStringSubmatch(trimmed); m != nil {
				pendingItem.Savings = parseFloat(m[1])
				continue
			}
		}

		if m := reContinenteSubtotal.FindStringSubmatch(trimmed); m != nil {
			if pendingItem != nil {
				r.Items = append(r.Items, *pendingItem)
				pendingItem = nil
			}
			continue
		}

		if m := reContinenteCardDisc.FindStringSubmatch(trimmed); m != nil {
			r.CardDiscount = parseFloat(m[1])
			continue
		}

		if m := reContinenteTotal.FindStringSubmatch(trimmed); m != nil {
			r.Total = parseFloat(m[1])
			if pendingItem != nil {
				r.Items = append(r.Items, *pendingItem)
				pendingItem = nil
			}
			continue
		}

		if m := reContinentePay.FindStringSubmatch(trimmed); m != nil {
			r.PaymentMethod = strings.TrimSpace(m[1])
			continue
		}

		if m := reContinenteSavAcc.FindStringSubmatch(trimmed); m != nil {
			r.TotalSavingsAcc = parseFloat(m[1])
			continue
		}

		if m := reContinenteCard.FindStringSubmatch(trimmed); m != nil {
			r.ClientCard = m[1]
			continue
		}

		if m := reContinenteUsedCard.FindStringSubmatch(trimmed); m != nil {
			r.SavingCardUsed = true
			continue
		}
	}

	if pendingItem != nil {
		r.Items = append(r.Items, *pendingItem)
	}

	if r.Total == 0 {
		if m := reContinenteCompra.FindStringSubmatch(strings.Join(lines, "\n")); m != nil {
			r.Total = parseFloat(m[1])
		}
	}

	return r, nil
}
