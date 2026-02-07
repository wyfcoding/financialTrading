FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o order cmd/order/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/order .
COPY --from=builder /app/configs/order/config.toml ./configs/order/

EXPOSE 8088 9088

CMD ["./order", "-config", "configs/order/config.toml"]
