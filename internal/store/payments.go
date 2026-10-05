package store

import "fmt"

type Payment struct {
	ID          int64
	CartID      int64
	OrderID     int64
	AmountCents int64
	CardLast4   string
}

func (s *Store) RecordPayment(cartID, orderID, amountCents int64, cardLast4 string) error {
	if _, err := s.db.Exec(
		`INSERT INTO payments (cart_id, order_id, amount_cents, card_last4) VALUES (?, ?, ?, ?)`,
		cartID, orderID, amountCents, cardLast4,
	); err != nil {
		return fmt.Errorf("store: échec de l'enregistrement du paiement: %w", err)
	}
	return nil
}
