FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o user cmd/user/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/user .
COPY --from=builder /app/configs/user/config.toml ./configs/user/
EXPOSE 9098
CMD ["./user", "-config", "configs/user/config.toml"]
