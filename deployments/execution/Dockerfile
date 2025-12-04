# === 构建阶段 (Builder Stage) ===
# 使用官方的 Go alpine 镜像作为构建环境，以保持镜像体积小。
FROM golang:1.21-alpine AS builder

# 标记镜像元数据
LABEL stage="builder"
LABEL maintainer="fynnwu <fynn.wu@example.com>"
LABEL description="编译 execution-service Go 二进制文件"

# 安装构建时依赖。
# - git: 用于下载 Go 模块。
# - gcc, musl-dev: CGO 工具链，用于静态编译。
RUN apk add --no-cache git gcc musl-dev

# 设置工作目录。
WORKDIR /build

# 优先复制 go.mod 和 go.sum，利用 Docker 层缓存加速依赖下载。
COPY go.mod go.sum ./

# 下载 Go 模块依赖。
RUN go mod download

# 复制整个项目的源代码。
COPY . .

# 编译 Go 应用程序为静态链接的二进制文件。
# - CGO_ENABLED=1: 启用 CGO。
# - GOOS=linux GOARCH=amd64: 交叉编译为 Linux amd64 平台的可执行文件。
# - ldflags="-s -w": 剥离符号表和调试信息，减小二进制文件体积。
# -o execution-service: 指定输出文件名。
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o execution-service ./cmd/execution

# === 运行阶段 (Runtime Stage) ===
# 使用轻量的 Alpine 镜像作为最终的运行环境。
FROM alpine:3.18

# 标记镜像元数据
LABEL maintainer="fynnwu <fynn.wu@example.com>"
LABEL description="运行 execution-service"

# 安装运行时依赖。
# - ca-certificates: 用于进行 HTTPS/TLS 通信。
# - tzdata: 提供时区数据。
RUN apk add --no-cache ca-certificates tzdata

# 创建一个专用的、非 root 的用户和组，以增强容器安全性。
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser

# 设置工作目录。
WORKDIR /app

# 从构建阶段复制已编译的二进制文件。
COPY --from=builder /build/execution-service .

# 创建并授权日志目录，确保非 root 用户有权限写入。
RUN mkdir -p /var/log/execution-service && chown -R appuser:appuser /var/log/execution-service

# 创建并授权临时文件目录。
RUN mkdir -p /tmp && chown -R appuser:appuser /tmp

# 切换到非 root 用户执行后续命令。
USER appuser

# 暴露服务监听的端口 (HTTP, gRPC, Metrics)。
# 这些端口号应与 configs/execution/config.toml 中的配置保持一致。
EXPOSE 8083 50054 9093

# 配置容器健康检查，供容器编排系统（如 K8s）使用。
# - --interval: 检查间隔
# - --timeout: 检查超时
# - --start-period: 启动保护期
# - --retries: 重试次数
# 使用正确的端口号 8083。
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

# 定义容器的启动命令。
ENTRYPOINT ["./execution-service"]
