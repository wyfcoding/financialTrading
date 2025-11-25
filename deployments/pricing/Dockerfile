# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/pricing-service ./cmd/pricing

# Final stage
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/bin/pricing-service .
COPY --from=builder /app/configs/pricing/config.toml ./configs/pricing/config.toml

# Expose ports
EXPOSE 50055 8084

# Run the application
CMD ["./pricing-service"]
