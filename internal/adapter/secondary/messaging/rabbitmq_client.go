package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/cashflow/payment-gateway/internal/port/output"
)

const (
	ExchangeName   = "payments"
	QueueName      = "payment_processing"
	RoutingKey     = "payment.created"
	PrefetchCount  = 1 // Process one message at a time per worker
)

// PaymentMessage represents a payment processing message
type PaymentMessage struct {
	PaymentID uuid.UUID `json:"payment_id"`
	Timestamp time.Time `json:"timestamp"`
}

// RabbitMQClient is a secondary adapter that implements PaymentMessaging output port
type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQClient creates a new RabbitMQ client (returns interface for ports)
func NewRabbitMQClient(amqpURL string) (output.PaymentMessaging, error) {
	return NewRabbitMQClientConcrete(amqpURL)
}

// NewRabbitMQClientConcrete creates a new RabbitMQ client (returns concrete type for workers)
func NewRabbitMQClientConcrete(amqpURL string) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	err = channel.ExchangeDeclare(
		ExchangeName,
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	_, err = channel.QueueDeclare(
		QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		QueueName,
		RoutingKey,
		ExchangeName,
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &RabbitMQClient{
		conn:    conn,
		channel: channel,
	}, nil
}

// PublishPaymentMessage publishes a payment processing message
func (c *RabbitMQClient) PublishPaymentMessage(paymentID uuid.UUID) error {
	message := PaymentMessage{
		PaymentID: paymentID,
		Timestamp: time.Now(),
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = c.channel.Publish(
		ExchangeName,
		RoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // Make message persistent
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published payment message for payment ID: %s", paymentID)
	return nil
}

// ConsumePaymentMessages starts consuming payment messages
func (c *RabbitMQClient) ConsumePaymentMessages(handler func(PaymentMessage) error) error {
	// Set QoS to process one message at a time
	err := c.channel.Qos(
		PrefetchCount,
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := c.channel.Consume(
		QueueName,
		"",    // consumer tag
		false, // auto-ack (we'll manually ack after processing)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Println("Started consuming payment messages...")

	go func() {
		for msg := range msgs {
			var paymentMsg PaymentMessage
			if err := json.Unmarshal(msg.Body, &paymentMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				msg.Nack(false, true) // Requeue message
				continue
			}

			// Process the message
			if err := handler(paymentMsg); err != nil {
				log.Printf("Error processing payment %s: %v", paymentMsg.PaymentID, err)
				// Check if message should be requeued
				// If it's a terminal state error (already processed), don't requeue
				if isTerminalError(err) {
					msg.Ack(false) // Acknowledge to remove from queue
				} else {
					msg.Nack(false, true) // Requeue for retry
				}
				continue
			}

			// Successfully processed
			msg.Ack(false)
			log.Printf("Successfully processed payment: %s", paymentMsg.PaymentID)
		}
	}()

	return nil
}

// Close closes the RabbitMQ connection
func (c *RabbitMQClient) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// isTerminalError checks if an error indicates a terminal state
// (e.g., payment already processed)
func isTerminalError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "already processed") || strings.Contains(errStr, "payment not found")
}

