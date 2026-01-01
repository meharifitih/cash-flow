package output

import (
	"github.com/google/uuid"
	"github.com/cashflow/payment-gateway/internal/core"
)

// PaymentRepository is an output port (secondary port) for payment data access
// Secondary adapters (database implementations) will implement this
type PaymentRepository interface {
	// Create creates a new payment
	Create(payment *core.Payment) error

	// GetByID retrieves a payment by its ID
	GetByID(id uuid.UUID) (*core.Payment, error)

	// ProcessPayment atomically processes a payment if it's in PENDING status
	// Uses SELECT FOR UPDATE to prevent concurrent processing
	ProcessPayment(id uuid.UUID, newStatus core.PaymentStatus) error

	// ReferenceExists checks if a reference already exists
	ReferenceExists(reference string) (bool, error)
}

