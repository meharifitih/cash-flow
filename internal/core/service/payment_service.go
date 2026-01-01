package service

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/cashflow/payment-gateway/internal/core"
	"github.com/cashflow/payment-gateway/internal/port/input"
	"github.com/cashflow/payment-gateway/internal/port/output"
)

// PaymentServiceImpl implements the PaymentService input port
type PaymentServiceImpl struct {
	paymentRepo output.PaymentRepository
	paymentMsg  output.PaymentMessaging
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	paymentRepo output.PaymentRepository,
	paymentMsg output.PaymentMessaging,
) input.PaymentService {
	return &PaymentServiceImpl{
		paymentRepo: paymentRepo,
		paymentMsg:  paymentMsg,
	}
}

// CreatePayment creates a new payment
func (s *PaymentServiceImpl) CreatePayment(req input.CreatePaymentRequest) (*input.PaymentResponse, error) {
	// Validate amount
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	// Validate currency
	if req.Currency != core.CurrencyETB && req.Currency != core.CurrencyUSD {
		return nil, fmt.Errorf("currency must be ETB or USD")
	}

	// Validate reference
	req.Reference = strings.TrimSpace(req.Reference)
	if req.Reference == "" {
		return nil, fmt.Errorf("reference is required")
	}

	// Check if reference already exists
	exists, err := s.paymentRepo.ReferenceExists(req.Reference)
	if err != nil {
		return nil, fmt.Errorf("failed to validate reference: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("reference already exists")
	}

	// Create payment entity
	payment := &core.Payment{
		ID:        uuid.New(),
		Amount:    req.Amount,
		Currency:  req.Currency,
		Reference: req.Reference,
		Status:  core.PaymentStatusPending,
	}

	// Save payment
	if err := s.paymentRepo.Create(payment); err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Publish message to queue (non-blocking - log error but don't fail)
	if err := s.paymentMsg.PublishPaymentMessage(payment.ID); err != nil {
		// In production, you might want to implement a retry mechanism or dead letter queue
		// For now, we log the error but don't fail the request since payment is already created
		return nil, fmt.Errorf("payment created but failed to publish message: %w", err)
	}

	// Return response
	return &input.PaymentResponse{
		ID:        payment.ID,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		Reference: payment.Reference,
		Status:    payment.Status,
		CreatedAt: payment.CreatedAt,
	}, nil
}

// GetPayment retrieves a payment by ID
func (s *PaymentServiceImpl) GetPayment(id uuid.UUID) (*input.PaymentResponse, error) {
	payment, err := s.paymentRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return &input.PaymentResponse{
		ID:        payment.ID,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		Reference: payment.Reference,
		Status:    payment.Status,
		CreatedAt: payment.CreatedAt,
	}, nil
}

