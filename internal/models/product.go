package models

type Product struct {
	ID             int64
	Name           string
	Description    string
	Category       string
	PriceCents     int64
	TaxRatePercent int // ex: 20 pour une TVA à 20%
	Stock          int
}

// PriceCentsTTC renvoie le prix TTC (toutes taxes comprises), calculé à
// partir du prix HT stocké et du taux de TVA du produit.
func (p Product) PriceCentsTTC() int64 {
	return p.PriceCents * (100 + int64(p.TaxRatePercent)) / 100
}
