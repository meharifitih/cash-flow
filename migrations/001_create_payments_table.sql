-- Create payments table
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    amount DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL CHECK (currency IN ('ETB', 'USD')),
    reference VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on reference for faster lookups
CREATE INDEX IF NOT EXISTS idx_payments_reference ON payments(reference);

-- Create index on status for faster queries
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);

-- Create index on created_at for time-based queries
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at);

