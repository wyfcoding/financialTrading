FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o marketsimulation cmd/marketsimulation/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/marketsimulation .
COPY --from=builder /app/configs/marketsimulation/config.toml ./configs/marketsimulation/

EXPOSE 8084 9084

CMD ["./marketsimulation", "-config", "configs/marketsimulation/config.toml"]
