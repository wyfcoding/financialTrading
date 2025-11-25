.PHONY: help build clean proto generate-all deploy-all

# 项目名称
PROJECT_NAME := financial-exchange
SERVICES := market-data order matching-engine execution account position risk clearing reference-data notification quant market-simulation pricing market-making monitoring-analytics

help:
	@echo "Financial Trading System - Makefile"
	@echo "Available targets:"
	@echo "  make build-all              - Build all services"
	@echo "  make build-<service>        - Build specific service"
	@echo "  make proto                  - Generate protobuf files"
	@echo "  make clean                  - Clean build artifacts"
	@echo "  make docker-build-all       - Build all Docker images"
	@echo "  make docker-push-all        - Push all Docker images"
	@echo "  make deploy-all             - Deploy all services to K8s"
	@echo "  make test-all               - Run all tests"

# 生成所有 protobuf 文件
proto:
	@echo "Generating protobuf files..."
	@for service in $(SERVICES); do \
		echo "Generating proto for $$service..."; \
		protoc --go_out=. --go-grpc_out=. --grpc-gateway_out=. api/$$service/*.proto || true; \
	done

# 编译所有服务
build-all:
	@echo "Building all services..."
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		CGO_ENABLED=1 go build -o bin/$$service ./cmd/$$service || true; \
	done

# 编译特定服务
build-%:
	@echo "Building $*..."
	@CGO_ENABLED=1 go build -o bin/$* ./cmd/$*

# 清理
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@find . -name "*.pb.go" -delete
	@find . -name "*.pb.gw.go" -delete

# Docker 构建所有服务
docker-build-all:
	@echo "Building Docker images for all services..."
	@for service in $(SERVICES); do \
		echo "Building Docker image for $$service..."; \
		docker build -f deployments/$$service/Dockerfile -t $(PROJECT_NAME)/$$service:latest . || true; \
	done

# Docker 推送所有镜像
docker-push-all:
	@echo "Pushing Docker images..."
	@for service in $(SERVICES); do \
		echo "Pushing $$service..."; \
		docker push $(PROJECT_NAME)/$$service:latest || true; \
	done

# 部署所有服务到 Kubernetes
deploy-all:
	@echo "Deploying all services to Kubernetes..."
	@for service in $(SERVICES); do \
		echo "Deploying $$service..."; \
		helm upgrade --install $$service deployments/$$service/helm -n trading-system --create-namespace || true; \
	done

# 运行所有测试
test-all:
	@echo "Running all tests..."
	@go test -v ./...

# 格式化代码
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# 代码检查
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# 生成依赖
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
