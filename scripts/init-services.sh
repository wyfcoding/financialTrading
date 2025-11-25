#!/bin/bash

# 初始化所有微服务的目录结构
# 使用方法: bash scripts/init-services.sh

set -e

# 定义所有服务
declare -A SERVICES=(
    ["reference-data"]="ReferenceDataService"
    ["notification"]="NotificationService"
    ["quant"]="QuantService"
    ["market-simulation"]="MarketSimulationService"
    ["pricing"]="PricingService"
    ["market-making"]="MarketMakingService"
    ["monitoring-analytics"]="MonitoringAnalyticsService"
)

echo "Initializing directory structure for all services..."

for service in "${!SERVICES[@]}"; do
    echo "Creating directories for $service..."
    
    # 创建目录结构
    mkdir -p "internal/$service/domain"
    mkdir -p "internal/$service/application"
    mkdir -p "internal/$service/infrastructure/repository"
    mkdir -p "internal/$service/interfaces/http"
    mkdir -p "internal/$service/interfaces/grpc"
    mkdir -p "cmd/$service"
    mkdir -p "configs/$service"
    mkdir -p "deployments/$service/helm/templates"
    mkdir -p "go-api/$service"
    
    echo "✓ Directories created for $service"
done

echo "All directory structures initialized!"
