FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o risk cmd/risk/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/risk .
COPY --from=builder /app/configs/risk/config.toml ./configs/risk/

EXPOSE 9093

CMD ["./risk", "-config", "configs/risk/config.toml"]
