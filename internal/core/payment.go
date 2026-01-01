package core

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusSuccess PaymentStatus = "SUCCESS"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

// Currency represents supported currencies
type Currency string

const (
	CurrencyETB Currency = "ETB"
	CurrencyUSD Currency = "USD"
)

// Payment represents a payment domain entity
type Payment struct {
	ID        uuid.UUID
	Amount    float64
	Currency  Currency
	Reference string
	Status    PaymentStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsPending checks if payment is in pending status
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending
}

// IsTerminal checks if payment is in a terminal state
func (p *Payment) IsTerminal() bool {
	return p.Status == PaymentStatusSuccess || p.Status == PaymentStatusFailed
}

