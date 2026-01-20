FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o referencedata cmd/referencedata/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/referencedata .
COPY --from=builder /app/configs/referencedata/config.toml ./configs/referencedata/

EXPOSE 9092

CMD ["./referencedata", "-config", "configs/referencedata/config.toml"]
