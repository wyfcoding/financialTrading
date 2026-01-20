# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum from root
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source
COPY . .

# Build the connectivity service
RUN CGO_ENABLED=0 GOOS=linux go build -o connectivity cmd/connectivity/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/connectivity .
COPY --from=builder /app/configs/connectivity/config.toml ./configs/connectivity/config.toml

EXPOSE 8089 9099 9800

ENTRYPOINT ["./connectivity"]
CMD ["--config", "configs/connectivity/config.toml"]
