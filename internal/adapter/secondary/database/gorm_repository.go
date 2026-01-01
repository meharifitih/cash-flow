package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/cashflow/payment-gateway/internal/constant/model/db"
	"github.com/cashflow/payment-gateway/internal/core"
	"github.com/cashflow/payment-gateway/internal/port/output"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormPaymentRepository is a secondary adapter that implements PaymentRepository output port
type GormPaymentRepository struct {
	gormDB *gorm.DB
}

// NewGormPaymentRepository creates a new GORM payment repository
func NewGormPaymentRepository(gormDB *gorm.DB) output.PaymentRepository {
	return &GormPaymentRepository{gormDB: gormDB}
}

// toCore converts db.Payment to core.Payment
func toCore(p *db.Payment) *core.Payment {
	return &core.Payment{
		ID:        p.ID,
		Amount:    p.Amount,
		Currency:  core.Currency(p.Currency),
		Reference: p.Reference,
		Status:    core.PaymentStatus(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// fromCore converts core.Payment to db.Payment
func fromCore(p *core.Payment) *db.Payment {
	return &db.Payment{
		ID:        p.ID,
		Amount:    p.Amount,
		Currency:  db.Currency(p.Currency),
		Reference: p.Reference,
		Status:    db.PaymentStatus(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// Create creates a new payment
func (r *GormPaymentRepository) Create(payment *core.Payment) error {
	dbPayment := fromCore(payment)
	if err := r.gormDB.Create(dbPayment).Error; err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	// Update core entity with timestamps set by GORM hooks
	payment.CreatedAt = dbPayment.CreatedAt
	payment.UpdatedAt = dbPayment.UpdatedAt
	return nil
}

// GetByID retrieves a payment by its ID
func (r *GormPaymentRepository) GetByID(id uuid.UUID) (*core.Payment, error) {
	var dbPayment db.Payment
	if err := r.gormDB.Where("id = ?", id).First(&dbPayment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return toCore(&dbPayment), nil
}

// ProcessPayment atomically processes a payment if it's in PENDING status
// Uses SELECT FOR UPDATE to prevent concurrent processing
func (r *GormPaymentRepository) ProcessPayment(id uuid.UUID, newStatus core.PaymentStatus) error {
	return r.gormDB.Transaction(func(tx *gorm.DB) error {
		var dbPayment db.Payment

		// Lock the row and check status using SELECT FOR UPDATE
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", id).
			First(&dbPayment).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("payment not found")
			}
			return fmt.Errorf("failed to lock payment: %w", err)
		}

		// Only process if status is PENDING
		if dbPayment.Status != db.PaymentStatusPending {
			return fmt.Errorf("payment already processed: current status is %s", dbPayment.Status)
		}

		// Update the payment status
		dbPayment.Status = db.PaymentStatus(newStatus)
		dbPayment.UpdatedAt = time.Now()

		if err := tx.Save(&dbPayment).Error; err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		return nil
	})
}

// ReferenceExists checks if a reference already exists
func (r *GormPaymentRepository) ReferenceExists(reference string) (bool, error) {
	var count int64
	if err := r.gormDB.Model(&db.Payment{}).
		Where("reference = ?", reference).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check reference: %w", err)
	}
	return count > 0, nil
}

