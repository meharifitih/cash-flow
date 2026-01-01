package output

import (
	"github.com/google/uuid"
)

// PaymentMessaging is an output port (secondary port) for payment messaging
// Secondary adapters (RabbitMQ implementations) will implement this
type PaymentMessaging interface {
	// PublishPaymentMessage publishes a payment processing message
	PublishPaymentMessage(paymentID uuid.UUID) error
	// Close closes the messaging connection
	Close() error
}

