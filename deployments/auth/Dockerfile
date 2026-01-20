FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o auth cmd/auth/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/auth .
COPY --from=builder /app/configs/auth/config.toml ./configs/auth/
EXPOSE 9095 8085
CMD ["./auth", "-config", "configs/auth/config.toml"]
