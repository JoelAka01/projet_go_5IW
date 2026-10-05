package models

type Product struct {
	ID             int64
	Name           string
	Description    string
	Category       string
	PriceCents     int64
	TaxRatePercent int
	Stock          int
}

func (p Product) PriceCentsTTC() int64 {
	return p.PriceCents * (100 + int64(p.TaxRatePercent)) / 100
}
