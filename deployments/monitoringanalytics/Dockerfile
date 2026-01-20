FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o monitoringanalytics cmd/monitoringanalytics/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/monitoringanalytics .
COPY --from=builder /app/configs/monitoringanalytics/config.toml ./configs/monitoringanalytics/

EXPOSE 9086 8086

CMD ["./monitoringanalytics", "-config", "configs/monitoringanalytics/config.toml"]
