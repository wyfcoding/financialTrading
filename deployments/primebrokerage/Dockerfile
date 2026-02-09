# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o primebrokerage financialTrading/cmd/primebrokerage/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/primebrokerage .
COPY --from=builder /app/financialTrading/configs/primebrokerage/config.toml ./configs/

EXPOSE 50051 8081

CMD ["./primebrokerage"]
