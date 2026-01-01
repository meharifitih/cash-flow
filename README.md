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

### Prerequisites

- Docker and Docker Compose (for containerized setup)
- Go 1.21+ (for local development)
- PostgreSQL 15+ (if running database locally)
- RabbitMQ 3+ (if running messaging locally)

### Using Docker Compose (Recommended)

This is the easiest way to run the entire application stack.

1. **Clone the repository and navigate to the project directory:**
```bash
cd Cashflow
```

2. **Copy environment file (optional, defaults are already set):**
```bash
cp .env.example .env
# Edit .env if you need to change default values
```

3. **Start all services:**
```bash
docker-compose up -d
```

This will start:
- PostgreSQL on port 5432
- RabbitMQ on port 5672 (Management UI on port 15672)
- API server on port 8080
- 1 worker instance

4. **Check service status:**
```bash
docker-compose ps
```

5. **View logs:**
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api
docker-compose logs -f worker
```

6. **Run multiple workers for concurrency testing:**
```bash
docker-compose up -d --scale worker=2
```

7. **Wait for services to be healthy (about 10-15 seconds)**

8. **Test the API:**
```bash
# Create a payment
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 100.50,
    "currency": "USD",
    "reference": "REF-001"
  }'

# Get payment status (replace {payment-id} with actual ID from response)
curl http://localhost:8080/api/v1/payments/{payment-id}

# Health check
curl http://localhost:8080/health
```

9. **Stop services:**
```bash
docker-compose down
```

10. **Stop and remove volumes (clean database):**
```bash
docker-compose down -v
```

### Local Development

For development without Docker containers.

#### Step 1: Start Infrastructure Services

Start only PostgreSQL and RabbitMQ using Docker:
```bash
docker-compose up -d postgres rabbitmq
```

Or run them locally if you have them installed.

#### Step 2: Setup Environment Variables

Copy the example environment file:
```bash
cp .env.example .env
```

Edit `.env` with your local configuration:
```bash
# .env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/payments?sslmode=disable
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
PORT=8080
ENV=development
```

#### Step 3: Run Database Migrations

The application uses GORM auto-migration, but you can also run the SQL migration manually:
```bash
# Using psql
psql -h localhost -U postgres -d payments -f migrations/001_create_payments_table.sql

# Or connect to PostgreSQL and run manually
psql -h localhost -U postgres -d payments
```

#### Step 4: Download Dependencies

```bash
go mod download
```

#### Step 5: Run the API Server

**Option A: Using environment variables from .env file**

Install a package to load .env (if not already using one):
```bash
go get github.com/joho/godotenv
```

Or run with explicit environment variables:
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/payments?sslmode=disable"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export PORT=8080
go run cmd/api/main.go
```

**Option B: Using environment variables directly**
```bash
DATABASE_URL="postgres://postgres:postgres@localhost:5432/payments?sslmode=disable" \
RABBITMQ_URL="amqp://guest:guest@localhost:5672/" \
PORT=8080 \
go run cmd/api/main.go
```

The API server will start on `http://localhost:8080`

#### Step 6: Run the Worker (in a separate terminal)

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/payments?sslmode=disable"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
go run cmd/worker/main.go
```

Or using inline environment variables:
```bash
DATABASE_URL="postgres://postgres:postgres@localhost:5432/payments?sslmode=disable" \
RABBITMQ_URL="amqp://guest:guest@localhost:5672/" \
go run cmd/worker/main.go
```

#### Step 7: Build and Run (Production-like)

Build the binaries:
```bash
# Build API
go build -o bin/api cmd/api/main.go

# Build Worker
go build -o bin/worker cmd/worker/main.go
```

Run the binaries:
```bash
# Terminal 1: API
./bin/api

# Terminal 2: Worker
./bin/worker
```

### Running with Makefile (Optional)

If you prefer using Make commands, you can create a `Makefile`:

```makefile
.PHONY: run-api run-worker build clean

run-api:
	@go run cmd/api/main.go

run-worker:
	@go run cmd/worker/main.go

build:
	@go build -o bin/api cmd/api/main.go
	@go build -o bin/worker cmd/worker/main.go

clean:
	@rm -rf bin/
```

Then run:
```bash
make run-api    # Run API server
make run-worker # Run worker
make build      # Build binaries
```

### Development Workflow

1. **Start infrastructure:**
   ```bash
   docker-compose up -d postgres rabbitmq
   ```

2. **Run API in development mode:**
   ```bash
   go run cmd/api/main.go
   ```

3. **Run worker in development mode (separate terminal):**
   ```bash
   go run cmd/worker/main.go
   ```

4. **Make changes and restart** (Go's fast compilation makes this quick)

5. **Test your changes:**
   ```bash
   curl -X POST http://localhost:8080/api/v1/payments \
     -H "Content-Type: application/json" \
     -d '{"amount": 50.00, "currency": "ETB", "reference": "TEST-001"}'
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

