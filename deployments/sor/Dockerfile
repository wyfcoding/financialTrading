# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o sor financialTrading/cmd/sor/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/sor .
COPY --from=builder /app/financialTrading/configs/sor/config.toml ./configs/

EXPOSE 50052

CMD ["./sor"]
