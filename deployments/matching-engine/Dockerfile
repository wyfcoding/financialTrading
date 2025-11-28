# 多阶段构建：编译阶段
FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git gcc musl-dev

# 设置工作目录
WORKDIR /build

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o matching-engine-service ./cmd/matching-engine

# 多阶段构建：运行阶段
FROM alpine:3.18

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/matching-engine-service .

# 创建日志目录
RUN mkdir -p /var/log/matching-engine-service && chown -R appuser:appuser /var/log/matching-engine-service

# 创建临时目录
RUN mkdir -p /tmp && chown -R appuser:appuser /tmp

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 8080 50051 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
ENTRYPOINT ["./matching-engine-service"]
