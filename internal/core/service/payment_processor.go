package service

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/cashflow/payment-gateway/internal/core"
	"github.com/cashflow/payment-gateway/internal/port/output"
)

// PaymentProcessor handles payment processing business logic
type PaymentProcessor struct {
	paymentRepo output.PaymentRepository
}

// NewPaymentProcessor creates a new payment processor
func NewPaymentProcessor(paymentRepo output.PaymentRepository) *PaymentProcessor {
	return &PaymentProcessor{
		paymentRepo: paymentRepo,
	}
}

// ProcessPayment processes a payment asynchronously
// This simulates payment processing and randomly assigns SUCCESS or FAILED status
// The processing is idempotent - it only processes payments in PENDING status
func (p *PaymentProcessor) ProcessPayment(paymentID uuid.UUID) error {
	// Randomly determine success or failure (50/50 chance)
	rand.Seed(time.Now().UnixNano())
	status := core.PaymentStatusFailed
	if rand.Float32() < 0.5 {
		status = core.PaymentStatusSuccess
	}

	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond)

	// Atomically update payment status
	// This uses SELECT FOR UPDATE to prevent concurrent processing
	err := p.paymentRepo.ProcessPayment(paymentID, status)
	if err != nil {
		return fmt.Errorf("failed to process payment: %w", err)
	}

	return nil
}

