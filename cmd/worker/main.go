package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cashflow/payment-gateway/internal/adapter/secondary/database"
	"github.com/cashflow/payment-gateway/internal/adapter/secondary/messaging"
	"github.com/cashflow/payment-gateway/internal/constant/model/db"
	"github.com/cashflow/payment-gateway/internal/core/service"
)

func main() {
	// Get configuration from environment variables
	dbConnStr := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/payments?sslmode=disable")
	amqpURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	// Initialize secondary adapter: Database
	dbConn, err := db.NewDB(dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Initialize secondary adapter: Repository (implements output port)
	paymentRepo := database.NewGormPaymentRepository(dbConn.DB)

	// Initialize core service: Payment processor
	paymentProcessor := service.NewPaymentProcessor(paymentRepo)

	// Initialize secondary adapter: Messaging (concrete type for worker)
	msgClient, err := messaging.NewRabbitMQClientConcrete(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer msgClient.Close()

	// Start consuming messages
	err = msgClient.ConsumePaymentMessages(func(msg messaging.PaymentMessage) error {
		log.Printf("Processing payment: %s", msg.PaymentID)
		return paymentProcessor.ProcessPayment(msg.PaymentID)
	})
	if err != nil {
		log.Fatalf("Failed to start consuming messages: %v", err)
	}

	log.Println("Payment worker started. Press CTRL+C to exit.")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
