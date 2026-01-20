FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o notification cmd/notification/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/notification .
COPY --from=builder /app/configs/notification/config.toml ./configs/notification/

EXPOSE 9087

CMD ["./notification", "-config", "configs/notification/config.toml"]
