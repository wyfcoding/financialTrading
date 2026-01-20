FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o matchingengine cmd/matchingengine/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/matchingengine .
COPY --from=builder /app/configs/matchingengine/config.toml ./configs/matchingengine/

EXPOSE 9085

CMD ["./matchingengine", "-config", "configs/matchingengine/config.toml"]
