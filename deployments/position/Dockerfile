FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o position cmd/position/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/position .
COPY --from=builder /app/configs/position/config.toml ./configs/position/

EXPOSE 9089

CMD ["./position", "-config", "configs/position/config.toml"]
