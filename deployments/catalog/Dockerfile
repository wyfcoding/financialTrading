FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o catalog cmd/catalog/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/catalog .
COPY --from=builder /app/configs/catalog/config.toml ./configs/catalog/
EXPOSE 9097
CMD ["./catalog", "-config", "configs/catalog/config.toml"]
