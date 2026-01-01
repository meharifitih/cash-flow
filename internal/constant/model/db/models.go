package db

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

// Payment represents a payment entity in the database
type Payment struct {
	ID        uuid.UUID     `gorm:"type:uuid;primary_key" json:"id"`
	Amount    float64       `gorm:"type:decimal(15,2);not null" json:"amount"`
	Currency  Currency      `gorm:"type:varchar(3);not null" json:"currency"`
	Reference string        `gorm:"type:varchar(255);not null;uniqueIndex" json:"reference"`
	Status    PaymentStatus `gorm:"type:varchar(20);not null" json:"status"`
	CreatedAt time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Payment) TableName() string {
	return "payments"
}

// BeforeCreate is a GORM hook that runs before creating a record
func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a record
func (p *Payment) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	return nil
}

// IsPending checks if payment is in pending status
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending
}

// IsTerminal checks if payment is in a terminal state
func (p *Payment) IsTerminal() bool {
	return p.Status == PaymentStatusSuccess || p.Status == PaymentStatusFailed
}
