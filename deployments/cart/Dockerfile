FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o cart cmd/cart/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/cart .
COPY --from=builder /app/configs/cart/config.toml ./configs/cart/
EXPOSE 9096 8086
CMD ["./cart", "-config", "configs/cart/config.toml"]
