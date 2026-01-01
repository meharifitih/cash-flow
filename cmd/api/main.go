package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cashflow/payment-gateway/internal/adapter/primary/http"
	"github.com/cashflow/payment-gateway/internal/adapter/secondary/database"
	"github.com/cashflow/payment-gateway/internal/adapter/secondary/messaging"
	"github.com/cashflow/payment-gateway/internal/constant/model/db"
	"github.com/cashflow/payment-gateway/internal/core/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Get configuration from environment variables
	dbConnStr := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/payments?sslmode=disable")
	amqpURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	port := getEnv("PORT", "8080")

	// Initialize secondary adapter: Database
	dbConn, err := db.NewDB(dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Initialize secondary adapters: Repository and Messaging (implement output ports)
	paymentRepo := database.NewGormPaymentRepository(dbConn.DB)
	msgClient, err := messaging.NewRabbitMQClient(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer msgClient.Close()

	// Initialize core service (implements input port)
	paymentService := service.NewPaymentService(paymentRepo, msgClient)

	// Initialize primary adapter: HTTP handler (uses input port)
	paymentHandler := http.NewPaymentHandler(paymentService)

	// Initialize Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	api := e.Group("/api/v1")
	api.POST("/payments", paymentHandler.CreatePayment)
	api.GET("/payments/:id", paymentHandler.GetPayment)

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting API server on %s", addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
