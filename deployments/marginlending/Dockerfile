# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o marginlending financialTrading/cmd/marginlending/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/marginlending .
COPY --from=builder /app/financialTrading/configs/marginlending/config.toml ./configs/

EXPOSE 50055

CMD ["./marginlending"]
