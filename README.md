# Payment Gateway Service

A production-ready payment gateway service built with Go, featuring asynchronous payment processing, idempotent operations, and safe concurrency handling.

## Features

- **Payment Creation API**: Create payments with validation (amount, currency, unique reference)
- **Asynchronous Processing**: Background workers process payments via RabbitMQ
- **Idempotent Processing**: Payments can never be processed more than once, even with message redelivery
- **Concurrency Safe**: Uses PostgreSQL row-level locking to prevent race conditions
- **Status Tracking**: Real-time payment status (PENDING, SUCCESS, FAILED)
- **Reliable Messaging**: Handles RabbitMQ message redelivery and multiple concurrent workers

## Architecture

This project follows **Hexagonal Architecture** (Ports and Adapters) principles with clear separation of concerns:

### Architecture Layers (Hexagonal Architecture)

```
                    ┌─────────────────────────────┐
                    │      Primary Adapters       │
                    │   (Driving/Inbound Ports)   │
                    │  - HTTP Handlers            │
                    │  - CLI, gRPC, etc.          │
                    └──────────────┬──────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │      Input Ports            │
                    │   (Primary/Inbound)         │
                    │  - PaymentService           │
                    └──────────────┬──────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
┌───────▼────────┐      ┌──────────▼──────────┐    ┌─────────▼─────────┐
│   Core/Domain  │      │   Core Services    │    │   Core Entities   │
│  (Business     │      │  (Business Logic)  │    │  (Payment, etc.)   │
│   Logic)       │      │                     │    │                    │
└───────┬────────┘      └────────────────────┘    └────────────────────┘
        │
┌───────▼────────┐
│  Output Ports  │
│ (Secondary/    │
│  Outbound)     │
│ - PaymentRepo  │
│ - PaymentMsg   │
└───────┬────────┘
        │
┌───────▼─────────────────────────┐
│   Secondary Adapters            │
│   (Driven/Outbound)             │
│  - GORM Repository              │
│  - RabbitMQ Client              │
│  - External Services            │
└─────────────────────────────────┘
```

### Hexagonal Architecture Principles

- **Core is independent**: Business logic has no dependencies on adapters
- **Ports define contracts**: Interfaces (ports) define what the core needs
- **Adapters implement ports**: Primary adapters call core, secondary adapters are called by core
- **Dependency inversion**: Core depends on abstractions (ports), not implementations
- **Testability**: Easy to swap adapters for testing (mock repositories, in-memory DB, etc.)

### System Flow

```
┌─────────┐      ┌─────────┐      ┌──────────┐
│   API   │─────▶│ RabbitMQ│─────▶│  Worker  │
│  (Echo) │      │         │      │ (x2)     │
└─────────┘      └─────────┘      └──────────┘
     │                                  │
     └──────────┬───────────────────────┘
                │
          ┌─────────┐
          │PostgreSQL│
          └─────────┘
```

## Tech Stack

- **Backend API**: Go with Echo framework
- **Worker**: Go with RabbitMQ consumer
- **Database**: PostgreSQL with GORM ORM
- **Messaging**: RabbitMQ
- **Containerization**: Docker & Docker Compose

## Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository and navigate to the project directory:
```bash
cd Cashflow
```

2. Start all services:
```bash
docker-compose up -d
```

This will start:
- PostgreSQL on port 5432
- RabbitMQ on port 5672 (Management UI on port 15672)
- API server on port 8080
- 1 worker instance

To run multiple workers for concurrency testing:
```bash
docker-compose up -d --scale worker=2
```

3. Wait for services to be healthy (about 10-15 seconds)

4. Test the API:
```bash
# Create a payment
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 100.50,
    "currency": "USD",
    "reference": "REF-001"
  }'

# Get payment status
curl http://localhost:8080/api/v1/payments/{payment-id}
```

### Local Development

1. Start infrastructure services:
```bash
docker-compose up -d postgres rabbitmq
```

2. Run database migrations:
```bash
# Connect to PostgreSQL and run migrations
psql -h localhost -U postgres -d payments -f migrations/001_create_payments_table.sql
```

3. Download dependencies:
```bash
go mod download
```

4. Run the API server:
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/payments?sslmode=disable"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
go run cmd/api/main.go
```

5. Run the worker (in a separate terminal):
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/payments?sslmode=disable"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
go run cmd/worker/main.go
```

## API Endpoints

### Create Payment

**POST** `/api/v1/payments`

Request body:
```json
{
  "amount": 100.50,
  "currency": "USD",
  "reference": "REF-001"
}
```

Response (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "amount": 100.50,
  "currency": "USD",
  "reference": "REF-001",
  "status": "PENDING",
  "created_at": "2024-01-01T12:00:00Z"
}
```

### Get Payment

**GET** `/api/v1/payments/:id`

Response (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "amount": 100.50,
  "currency": "USD",
  "reference": "REF-001",
  "status": "SUCCESS",
  "created_at": "2024-01-01T12:00:00Z"
}
```

### Health Check

**GET** `/health`

Response (200 OK):
```json
{
  "status": "ok"
}
```

## Payment Processing Flow

1. **Client creates payment** → API validates and stores in PostgreSQL with status `PENDING`
2. **Message published** → Payment ID published to RabbitMQ queue
3. **Worker consumes** → Background worker picks up the message
4. **Idempotent processing** → Worker uses `SELECT FOR UPDATE` to lock the payment row
5. **Status check** → Only processes if status is `PENDING`
6. **Update status** → Randomly assigns `SUCCESS` or `FAILED` (simulated)
7. **Message acknowledgment** → Message is acked only after successful processing

## Idempotency Guarantees

The system ensures idempotent payment processing through:

1. **Database Transactions**: All payment updates happen within transactions
2. **Row-Level Locking**: `SELECT FOR UPDATE` prevents concurrent processing
3. **Status Validation**: Payments in terminal states (SUCCESS, FAILED) are never reprocessed
4. **Message Handling**: Messages for already-processed payments are acknowledged without requeue

## Concurrency Handling

- Multiple workers can run concurrently
- Each worker processes one message at a time (QoS prefetch=1)
- Database row-level locking prevents race conditions
- PostgreSQL is the source of truth for payment status

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/payments?sslmode=disable` |
| `RABBITMQ_URL` | RabbitMQ connection string | `amqp://guest:guest@localhost:5672/` |
| `PORT` | API server port | `8080` |

## Project Structure

```
.
├── cmd/
│   ├── api/                    # API server entry point
│   └── worker/                 # Worker service entry point
├── internal/
│   ├── core/                   # Core business logic (hexagon center)
│   │   ├── payment.go         # Domain entities
│   │   └── service/           # Business logic services
│   │       ├── payment_service.go
│   │       └── payment_processor.go
│   ├── port/                   # Ports (interfaces)
│   │   ├── input/             # Input ports (primary ports)
│   │   │   └── payment_service.go
│   │   └── output/            # Output ports (secondary ports)
│   │       ├── payment_repository.go
│   │       └── payment_messaging.go
│   ├── adapter/                # Adapters (implementations)
│   │   ├── primary/           # Primary adapters (driving/inbound)
│   │   │   └── http/          # HTTP handlers
│   │   │       └── payment_handler.go
│   │   └── secondary/        # Secondary adapters (driven/outbound)
│   │       ├── database/      # GORM repository implementation
│   │       │   └── gorm_repository.go
│   │       └── messaging/     # RabbitMQ client implementation
│   │           └── rabbitmq_client.go
│   └── constant/              # Constants and models
│       └── model/db/          # Database models (GORM)
│           ├── models.go
│           └── db.go
├── migrations/                 # Database migrations
├── docker-compose.yml
├── Dockerfile.api
├── Dockerfile.worker
└── README.md
```

### Clean Architecture Benefits

- **Testability**: Easy to mock interfaces for unit testing
- **Maintainability**: Clear separation of concerns
- **Flexibility**: Swap implementations (e.g., change DB or messaging system)
- **Independence**: Business logic doesn't depend on frameworks
- **Scalability**: Each layer can evolve independently

## Testing

### Manual Testing

1. Create multiple payments:
```bash
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/payments \
    -H "Content-Type: application/json" \
    -d "{\"amount\": $((RANDOM % 1000)), \"currency\": \"USD\", \"reference\": \"REF-$i\"}"
done
```

2. Check payment statuses:
```bash
# Replace {payment-id} with actual ID from creation response
curl http://localhost:8080/api/v1/payments/{payment-id}
```

3. Monitor RabbitMQ:
- Open http://localhost:15672 (guest/guest)
- Check queue `payment_processing` for messages

### Testing Idempotency

1. Create a payment and note the ID
2. Manually requeue the message in RabbitMQ management UI
3. Verify the payment is only processed once (check logs and database)

## Monitoring

- **RabbitMQ Management UI**: http://localhost:15672 (guest/guest)
- **API Logs**: `docker-compose logs -f api`
- **Worker Logs**: `docker-compose logs -f worker`
- **Database**: Connect to PostgreSQL on port 5432

## Stopping Services

```bash
docker-compose down
```

To remove volumes (clean database):
```bash
docker-compose down -v
```

## Production Considerations

For production deployment, consider:

1. **Security**: Use proper authentication/authorization
2. **TLS**: Enable TLS for database and RabbitMQ connections
3. **Monitoring**: Add metrics and distributed tracing
4. **Retry Logic**: Implement exponential backoff for failed messages
5. **Dead Letter Queue**: Handle permanently failed messages
6. **Database Pooling**: Tune connection pool settings
7. **Rate Limiting**: Add rate limiting to API endpoints
8. **Logging**: Structured logging with correlation IDs

## License

This project is part of a technical assessment.

