#!/bin/bash

# 生成所有 protobuf 文件的脚本
# 使用方法: bash scripts/generate-pb.sh

set -e

# 定义服务列表
SERVICES=(
    "market-data"
    "order"
    "matching-engine"
    "execution"
    "account"
    "position"
    "risk"
    "clearing"
    "reference-data"
    "notification"
    "quant"
    "market-simulation"
    "pricing"
    "market-making"
    "monitoring-analytics"
)

echo "Generating protobuf files for all services..."

for service in "${SERVICES[@]}"; do
    echo "Generating proto for $service..."
    
    # 检查 proto 文件是否存在
    if [ -f "api/$service/${service//-/_}.proto" ]; then
        protoc \
            --go_out=. \
            --go-grpc_out=. \
            --go_opt=module=github.com/trading-system/financial-exchange \
            --go-grpc_opt=module=github.com/trading-system/financial-exchange \
            "api/$service/${service//-/_}.proto" || echo "Warning: Failed to generate proto for $service"
    else
        echo "Warning: Proto file not found for $service"
    fi
done

echo "Protobuf generation completed!"
