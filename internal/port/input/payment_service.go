package input

import (
	"time"

	"github.com/google/uuid"
	"github.com/cashflow/payment-gateway/internal/core"
)

// PaymentService is an input port (primary port) for payment operations
// Primary adapters (HTTP handlers) will use this
type PaymentService interface {
	// CreatePayment creates a new payment
	CreatePayment(req CreatePaymentRequest) (*PaymentResponse, error)

	// GetPayment retrieves a payment by ID
	GetPayment(id uuid.UUID) (*PaymentResponse, error)
}

// CreatePaymentRequest represents the request to create a payment
type CreatePaymentRequest struct {
	Amount    float64
	Currency  core.Currency
	Reference string
}

// PaymentResponse represents the response for a payment
type PaymentResponse struct {
	ID        uuid.UUID
	Amount    float64
	Currency  core.Currency
	Reference string
	Status    core.PaymentStatus
	CreatedAt time.Time
}

