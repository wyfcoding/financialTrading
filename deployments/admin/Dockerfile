FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o admin cmd/admin/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/admin .
COPY --from=builder /app/configs/admin/config.toml ./configs/admin/

EXPOSE 9094 8084

CMD ["./admin", "-config", "configs/admin/config.toml"]
