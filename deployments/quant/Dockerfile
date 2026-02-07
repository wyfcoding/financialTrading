FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o quant cmd/quant/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/quant .
COPY --from=builder /app/configs/quant/config.toml ./configs/quant/

EXPOSE 8080 9091

CMD ["./quant", "-config", "configs/quant/config.toml"]
