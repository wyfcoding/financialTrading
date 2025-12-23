#!/bin/bash

# 遇到错误立即停止
set -e

# 配置路径
API_ROOT="./api"
OUT_ROOT="./go-api"

# 检查必要的工具
check_tool() {
    if ! command -v "$1" &> /dev/null;
    then
        echo "Error: $1 is not installed. Please install it first."
        exit 1
    fi
}

check_tool "protoc"
check_tool "protoc-gen-go"
check_tool "protoc-gen-go-grpc"

echo "Cleanup old generated files in $OUT_ROOT..."
mkdir -p "$OUT_ROOT"
# 清理旧的生成文件，避免残留（可选，根据需求开启）
# find "$OUT_ROOT" -name "*.pb.go" -delete

echo "Scanning for proto files in $API_ROOT..."
# 找到所有 proto 文件并转换为相对于 API_ROOT 的路径
PROTO_FILES=$(find "$API_ROOT" -name "*.proto" | sed "s|^$API_ROOT/||")

if [ -z "$PROTO_FILES" ]; then
    echo "No proto files found."
    exit 0
fi

# 批量生成，比循环调用 protoc 快得多
echo "Generating code for all services..."
protoc --proto_path="$API_ROOT" \
       --go_out="$OUT_ROOT" --go_opt=paths=source_relative \
       --go-grpc_out="$OUT_ROOT" --go-grpc_opt=paths=source_relative \
       $PROTO_FILES

echo "Protobuf generation completed successfully."
