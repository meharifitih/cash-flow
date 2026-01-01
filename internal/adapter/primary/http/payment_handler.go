package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/cashflow/payment-gateway/internal/core"
	"github.com/cashflow/payment-gateway/internal/port/input"
)

// PaymentHandler is a primary adapter (HTTP handler)
type PaymentHandler struct {
	paymentService input.PaymentService
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentService input.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreatePaymentRequest represents the HTTP request to create a payment
type CreatePaymentRequest struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Reference string  `json:"reference"`
}

// PaymentResponse represents the HTTP response for a payment
type PaymentResponse struct {
	ID        string  `json:"id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Reference string  `json:"reference"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

// CreatePayment handles payment creation
func (h *PaymentHandler) CreatePayment(c echo.Context) error {
	var req CreatePaymentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Convert to service request
	serviceReq := input.CreatePaymentRequest{
		Amount:    req.Amount,
		Currency:  core.Currency(req.Currency),
		Reference: req.Reference,
	}

	// Call service (input port)
	response, err := h.paymentService.CreatePayment(serviceReq)
	if err != nil {
		// Handle different error types
		if strings.Contains(err.Error(), "must be greater than zero") ||
			strings.Contains(err.Error(), "must be ETB or USD") ||
			strings.Contains(err.Error(), "reference is required") {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		if strings.Contains(err.Error(), "already exists") {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create payment",
		})
	}

	// Convert to HTTP response
	httpResponse := PaymentResponse{
		ID:        response.ID.String(),
		Amount:    response.Amount,
		Currency:  string(response.Currency),
		Reference: response.Reference,
		Status:    string(response.Status),
		CreatedAt: response.CreatedAt.Format(time.RFC3339),
	}

	return c.JSON(http.StatusCreated, httpResponse)
}

// GetPayment handles payment retrieval by ID
func (h *PaymentHandler) GetPayment(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid payment ID",
		})
	}

	// Call service (input port)
	response, err := h.paymentService.GetPayment(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Payment not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve payment",
		})
	}

	// Convert to HTTP response
	httpResponse := PaymentResponse{
		ID:        response.ID.String(),
		Amount:    response.Amount,
		Currency:  string(response.Currency),
		Reference: response.Reference,
		Status:    string(response.Status),
		CreatedAt: response.CreatedAt.Format(time.RFC3339),
	}

	return c.JSON(http.StatusOK, httpResponse)
}

