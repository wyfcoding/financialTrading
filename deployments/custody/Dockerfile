# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o custody financialTrading/cmd/custody/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/custody .
COPY --from=builder /app/financialTrading/configs/custody/config.toml ./configs/

EXPOSE 50053

CMD ["./custody"]
