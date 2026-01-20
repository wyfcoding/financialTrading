FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o pricing cmd/pricing/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/pricing .
COPY --from=builder /app/configs/pricing/config.toml ./configs/pricing/

EXPOSE 9090

CMD ["./pricing", "-config", "configs/pricing/config.toml"]
