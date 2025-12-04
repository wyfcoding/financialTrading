# === 构建阶段 (Builder Stage) ===
# 使用官方的 Go 语言镜像作为构建环境。
# 选择 -alpine 版本是为了保持镜像体积小。
FROM golang:1.21-alpine AS builder

# 标记镜像信息
LABEL stage="builder"
LABEL maintainer="fynnwu <fynn.wu@example.com>"
LABEL description="编译 account-service Go 二进制文件"

# 安装构建时依赖。
# - git: 用于下载 Go 模块。
# - gcc, musl-dev: 用于 CGO，如果项目中有 C 代码依赖（例如某些 Go 库需要）。
RUN apk add --no-cache git gcc musl-dev

# 设置工作目录，后续的命令将在此目录下执行。
WORKDIR /build

# 复制 go.mod 和 go.sum 文件。
# 这是为了利用 Docker 的层缓存机制。只要这两个文件没有变化，下一步的 go mod download 就不会重新执行。
COPY go.mod go.sum ./

# 下载 Go 模块依赖。
RUN go mod download

# 复制整个项目的源代码到工作目录。
COPY . .

# 编译 Go 应用程序。
# - CGO_ENABLED=1: 启用 CGO，与 musl-dev 配合使用，确保静态链接。
# - GOOS=linux GOARCH=amd64: 交叉编译，确保生成在 Linux amd64 架构上运行的二进制文件。
# - ldflags="-s -w": 优化二进制文件，-s 去除符号表，-w 去除 DWARF 调试信息，以减小体积。
# -o account-service: 指定输出的二进制文件名。
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o account-service ./cmd/account

# === 运行阶段 (Runtime Stage) ===
# 使用一个非常轻量的 Alpine Linux 作为最终运行环境。
FROM alpine:3.18

# 标记镜像信息
LABEL maintainer="fynnwu <fynn.wu@example.com>"
LABEL description="运行 account-service"

# 安装运行时依赖。
# - ca-certificates: 用于支持 HTTPS/TLS 连接。
# - tzdata: 提供时区信息，确保时间相关的操作正确。
RUN apk add --no-cache ca-certificates tzdata

# 创建一个非 root 用户和用户组，以增强安全性。
# 以非 root 用户运行容器是最佳实践，可以减少潜在的安全风险。
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser

# 设置工作目录。
WORKDIR /app

# 从构建阶段 (builder) 复制已编译的二进制文件到当前阶段。
COPY --from=builder /build/account-service .

# 创建并授权日志目录。
# 将日志目录的所有权交给 appuser，以便应用程序有权限写入日志文件。
RUN mkdir -p /var/log/account-service && chown -R appuser:appuser /var/log/account-service

# 创建并授权临时文件目录。
RUN mkdir -p /tmp && chown -R appuser:appuser /tmp

# 切换到非 root 用户来运行后续的命令和应用程序。
USER appuser

# 暴露服务需要监听的端口。
# - 8080: HTTP 端口 (示例)
# - 50051: gRPC 端口 (示例)
# - 9090: Metrics 端口 (示例)
# 注意：这里的端口号应与 config.toml 中配置的保持一致或通过环境变量传入。
EXPOSE 8084 50055 9094

# 配置容器健康检查。
# K8s 或 Docker Swarm 可以使用这个检查来判断容器是否健康。
# - --interval=30s: 每 30 秒检查一次。
# - --timeout=5s: 每次检查的超时时间为 5 秒。
# - --start-period=10s: 容器启动后，等待 10 秒再开始第一次健康检查。
# - --retries=3: 连续 3 次失败后，将容器标记为不健康。
# - CMD: 使用 wget 访问 /health 端点，如果请求失败则容器不健康。
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8084/health || exit 1

# 定义容器启动时执行的命令。
# 使用 exec 格式（JSON 数组）是推荐的做法，可以避免 shell apoch，并正确处理信号。
ENTRYPOINT ["./account-service"]
