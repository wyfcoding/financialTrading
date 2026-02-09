# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o backtest financialTrading/cmd/backtest/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/backtest .
COPY --from=builder /app/financialTrading/configs/backtest/config.toml ./configs/

EXPOSE 50054

CMD ["./backtest"]
