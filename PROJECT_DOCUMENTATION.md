# 金融交易系统项目文档

## 目录
- [项目概述](#项目概述)
- [系统架构](#系统架构)
- [技术栈](#技术栈)
- [核心模块](#核心模块)
- [领域驱动设计](#领域驱动设计)
- [基础设施](#基础设施)
- [部署架构](#部署架构)
- [API 接口](#api-接口)
- [数据模型](#数据模型)
- [开发指南](#开发指南)
- [运维指南](#运维指南)

---

## 项目概述

### 项目简介
这是一个基于 **领域驱动设计（DDD）** 和 **微服务架构** 的现代化金融交易系统。系统采用 Go 语言开发，支持高并发、低延迟的交易场景，提供完整的交易生命周期管理，包括订单管理、撮合引擎、清算结算、风险管理等核心功能。

### 核心特性
- ✅ **微服务架构**：15+ 独立微服务，职责清晰，易于扩展
- ✅ **领域驱动设计**：清晰的领域模型，业务逻辑与技术实现分离
- ✅ **高性能撮合引擎**：基于价格-时间优先算法的订单撮合
- ✅ **实时行情数据**：支持行情订阅、K线数据、订单簿快照
- ✅ **风险管理系统**：VaR、CVaR、最大回撤等风险指标计算
- ✅ **量化交易支持**：技术指标计算、回测框架、策略执行
- ✅ **期权定价**：Black-Scholes 模型、希腊字母计算
- ✅ **做市商系统**：网格策略、流动性提供
- ✅ **市场模拟**：蒙特卡洛模拟、几何布朗运动
- ✅ **完整的可观测性**：日志、指标、追踪三位一体

### 项目结构
```
FinancialTrading/
├── api/                    # gRPC 协议定义文件
├── cmd/                    # 各微服务的入口程序
├── configs/                # 配置文件
├── deployments/            # 部署相关文件（Docker、Helm）
├── go-api/                 # 生成的 gRPC 代码
├── internal/               # 内部业务逻辑
│   ├── account/           # 账户服务
│   ├── clearing/          # 清算服务
│   ├── execution/         # 执行服务
│   ├── market-data/       # 行情数据服务
│   ├── market-making/     # 做市商服务
│   ├── market-simulation/ # 市场模拟服务
│   ├── matching-engine/   # 撮合引擎
│   ├── monitoring-analytics/ # 监控分析服务
│   ├── notification/      # 通知服务
│   ├── order/             # 订单服务
│   ├── position/          # 持仓服务
│   ├── pricing/           # 定价服务
│   ├── quant/             # 量化服务
│   ├── reference-data/    # 参考数据服务
│   └── risk/              # 风险管理服务
├── pkg/                    # 公共库
│   ├── algos/             # 算法库
│   ├── cache/             # 缓存
│   ├── config/            # 配置管理
│   ├── db/                # 数据库
│   ├── grpcclient/        # gRPC 客户端
│   ├── logger/            # 日志
│   ├── metrics/           # 指标
│   ├── middleware/        # 中间件
│   ├── mq/                # 消息队列
│   └── utils/             # 工具函数
└── scripts/                # 脚本文件
```

---

## 系统架构

### 整体架构图
```
┌─────────────────────────────────────────────────────────────────┐
│                         客户端层                                  │
│  Web UI / Mobile App / Trading Terminal / API Clients           │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      API Gateway / Load Balancer                 │
│                    (Nginx / Istio / Envoy)                       │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                         微服务层                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │  订单服务  │  │  撮合引擎  │  │  执行服务  │  │  清算服务  │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │  账户服务  │  │  持仓服务  │  │  风险服务  │  │  行情服务  │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │  量化服务  │  │  定价服务  │  │  做市商   │  │  通知服务  │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      基础设施层                                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │   MySQL   │  │   Redis   │  │   Kafka   │  │  Jaeger   │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
│  ┌──────────┐  ┌──────────┐                                     │
│  │Prometheus │  │  Grafana  │                                     │
│  └──────────┘  └──────────┘                                     │
└─────────────────────────────────────────────────────────────────┘
```

### 架构特点

#### 1. 微服务架构
- **服务拆分**：按业务领域拆分为 15+ 独立服务
- **独立部署**：每个服务可独立开发、测试、部署
- **技术异构**：支持不同服务使用不同技术栈（当前统一使用 Go）
- **弹性伸缩**：根据负载动态调整服务实例数量

#### 2. 领域驱动设计（DDD）
每个服务内部采用 DDD 分层架构：
```
service/
├── application/        # 应用层：用例编排
├── domain/            # 领域层：核心业务逻辑
├── infrastructure/    # 基础设施层：技术实现
└── interfaces/        # 接口层：API 暴露
```

#### 3. 通信模式
- **同步通信**：gRPC（服务间调用）
- **异步通信**：Kafka（事件驱动）
- **HTTP REST**：对外 API（部分服务）

---

## 技术栈

### 后端技术
| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.25 | 主要编程语言 |
| gRPC | 1.77.0 | 服务间通信 |
| Protocol Buffers | 3.x | 接口定义语言 |
| GORM | 1.31.1 | ORM 框架 |
| Gin | 1.11.0 | HTTP 框架 |
| Zap | 1.27.0 | 结构化日志 |
| Viper | 1.21.0 | 配置管理 |

### 数据存储
| 技术 | 版本 | 用途 |
|------|------|------|
| MySQL | 8.0 | 关系型数据库 |
| Redis | 7.0 | 缓存、分布式锁 |
| Kafka | 7.4.0 | 消息队列 |

### 可观测性
| 技术 | 版本 | 用途 |
|------|------|------|
| Prometheus | latest | 指标收集 |
| Grafana | latest | 可视化 |
| Jaeger | latest | 分布式追踪 |

### 部署工具
| 技术 | 用途 |
|------|------|
| Docker | 容器化 |
| Docker Compose | 本地开发环境 |
| Kubernetes | 生产环境编排 |
| Helm | K8s 包管理 |
| Istio | 服务网格 |

---

## 核心模块

### 1. 订单服务（Order Service）

#### 功能描述
负责订单的创建、修改、取消和查询，是交易系统的核心入口。

#### 核心功能
- ✅ 创建订单（限价单、市价单、止损单等）
- ✅ 取消订单
- ✅ 修改订单
- ✅ 查询订单状态
- ✅ 订单历史查询
- ✅ 订单验证（价格、数量、余额等）

#### 领域模型
```go
type Order struct {
    OrderID        string          // 订单 ID
    UserID         string          // 用户 ID
    Symbol         string          // 交易对
    Side           OrderSide       // 买卖方向（BUY/SELL）
    Type           OrderType       // 订单类型（LIMIT/MARKET/STOP）
    Price          decimal.Decimal // 价格
    Quantity       decimal.Decimal // 数量
    FilledQuantity decimal.Decimal // 已成交数量
    Status         OrderStatus     // 状态（NEW/FILLED/CANCELLED）
    TimeInForce    TimeInForce     // 有效期（GTC/IOC/FOK）
    CreatedAt      time.Time       // 创建时间
    UpdatedAt      time.Time       // 更新时间
}
```

#### API 接口
- `CreateOrder(req)` - 创建订单
- `CancelOrder(orderID, userID)` - 取消订单
- `GetOrder(orderID, userID)` - 获取订单详情
- `ListOrders(userID, status, limit, offset)` - 查询订单列表

#### 数据库表
```sql
CREATE TABLE orders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id VARCHAR(50) UNIQUE NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL,
    type VARCHAR(20) NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    filled_quantity DECIMAL(20,8) NOT NULL,
    status VARCHAR(20) NOT NULL,
    time_in_force VARCHAR(10) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_symbol (symbol),
    INDEX idx_status (status)
);
```

---

### 2. 撮合引擎（Matching Engine）

#### 功能描述
核心交易撮合系统，负责订单的匹配和成交。采用价格-时间优先算法。

#### 核心功能
- ✅ 订单撮合（价格优先、时间优先）
- ✅ 订单簿管理（买单堆、卖单堆）
- ✅ 成交记录生成
- ✅ 订单簿快照
- ✅ 实时成交推送

#### 撮合算法
```
价格-时间优先算法：
1. 价格优先：买单价格高者优先，卖单价格低者优先
2. 时间优先：同价格下，时间早者优先
3. 数据结构：使用最大堆（买单）和最小堆（卖单）
```

#### 核心数据结构
```go
type OrderBook struct {
    mu     sync.RWMutex
    bids   *PriceHeap  // 买单堆（最大堆）
    asks   *PriceHeap  // 卖单堆（最小堆）
    orders map[string]*Order
}

type Trade struct {
    TradeID     string          // 成交 ID
    Symbol      string          // 交易对
    BuyOrderID  string          // 买方订单 ID
    SellOrderID string          // 卖方订单 ID
    Price       decimal.Decimal // 成交价格
    Quantity    decimal.Decimal // 成交数量
    Timestamp   int64           // 时间戳
}
```

#### 性能优化
- 使用堆数据结构，O(log n) 时间复杂度
- 读写锁优化并发性能
- 内存订单簿，减少数据库访问
- 批量成交记录持久化

---

### 3. 行情数据服务（Market Data Service）

#### 功能描述
提供实时行情数据、历史数据、K线数据、订单簿快照等。

#### 核心功能
- ✅ 实时行情推送（WebSocket/gRPC Stream）
- ✅ 历史行情查询
- ✅ K线数据（1m, 5m, 15m, 1h, 4h, 1d）
- ✅ 订单簿快照
- ✅ 最新成交记录
- ✅ 行情数据存储和清理

#### 领域模型
```go
type Quote struct {
    Symbol    string          // 交易对
    BidPrice  decimal.Decimal // 买价
    AskPrice  decimal.Decimal // 卖价
    BidSize   decimal.Decimal // 买量
    AskSize   decimal.Decimal // 卖量
    LastPrice decimal.Decimal // 最后成交价
    LastSize  decimal.Decimal // 最后成交量
    Timestamp int64           // 时间戳
    Source    string          // 数据来源
}

type Kline struct {
    Symbol    string          // 交易对
    Interval  string          // 时间周期
    OpenTime  int64           // 开盘时间
    Open      decimal.Decimal // 开盘价
    High      decimal.Decimal // 最高价
    Low       decimal.Decimal // 最低价
    Close     decimal.Decimal // 收盘价
    Volume    decimal.Decimal // 成交量
    CloseTime int64           // 收盘时间
}
```

#### API 接口
- `GetLatestQuote(symbol)` - 获取最新行情
- `GetHistoricalQuotes(symbol, startTime, endTime)` - 获取历史行情
- `GetKlines(symbol, interval, limit)` - 获取 K线数据
- `GetOrderBook(symbol, depth)` - 获取订单簿
- `SubscribeQuotes(symbols)` - 订阅行情（流式）

---

### 4. 账户服务（Account Service）

#### 功能描述
管理用户账户、余额、资金流水等。

#### 核心功能
- ✅ 创建账户
- ✅ 充值/提现
- ✅ 余额查询
- ✅ 资金冻结/解冻
- ✅ 交易流水记录
- ✅ 多币种支持

#### 领域模型
```go
type Account struct {
    AccountID        string          // 账户 ID
    UserID           string          // 用户 ID
    AccountType      string          // 账户类型（SPOT/MARGIN/FUTURES）
    Currency         string          // 货币
    Balance          decimal.Decimal // 余额
    AvailableBalance decimal.Decimal // 可用余额
    FrozenBalance    decimal.Decimal // 冻结余额
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type Transaction struct {
    TransactionID string          // 交易 ID
    AccountID     string          // 账户 ID
    Type          string          // 类型（DEPOSIT/WITHDRAW/TRADE/FEE）
    Amount        decimal.Decimal // 金额
    Status        string          // 状态
    CreatedAt     time.Time
}
```

---

### 5. 风险管理服务（Risk Service）

#### 功能描述
实时风险评估、风险指标计算、风险告警。

#### 核心功能
- ✅ 交易前风险检查
- ✅ VaR（Value at Risk）计算
- ✅ CVaR（Conditional VaR）计算
- ✅ 最大回撤计算
- ✅ 夏普比率计算
- ✅ 风险限额管理
- ✅ 风险告警

#### 风险指标
```go
type RiskMetrics struct {
    UserID      string          // 用户 ID
    VaR95       decimal.Decimal // 95% VaR
    VaR99       decimal.Decimal // 99% VaR
    MaxDrawdown decimal.Decimal // 最大回撤
    SharpeRatio decimal.Decimal // 夏普比率
    Correlation decimal.Decimal // 相关系数
    UpdatedAt   time.Time
}

type RiskAssessment struct {
    AssessmentID      string          // 评估 ID
    UserID            string          // 用户 ID
    Symbol            string          // 交易对
    RiskLevel         RiskLevel       // 风险等级（LOW/MEDIUM/HIGH）
    RiskScore         decimal.Decimal // 风险分数（0-100）
    MarginRequirement decimal.Decimal // 保证金要求
    IsAllowed         bool            // 是否允许交易
    Reason            string          // 原因
}
```

#### 风险计算算法
```go
// VaR 计算（参数法）
func CalculateVaR(returns []decimal.Decimal, confidenceLevel float64) decimal.Decimal {
    mean := calculateMean(returns)
    stdDev := calculateStdDev(returns)
    zScore := getZScore(confidenceLevel) // 95% -> 1.645, 99% -> 2.326
    return mean.Sub(stdDev.Mul(decimal.NewFromFloat(zScore)))
}

// 最大回撤计算
func CalculateMaxDrawdown(prices []decimal.Decimal) decimal.Decimal {
    maxPrice := prices[0]
    maxDrawdown := decimal.Zero
    for _, price := range prices {
        if price.GreaterThan(maxPrice) {
            maxPrice = price
        }
        drawdown := maxPrice.Sub(price).Div(maxPrice)
        if drawdown.GreaterThan(maxDrawdown) {
            maxDrawdown = drawdown
        }
    }
    return maxDrawdown
}
```

---

### 6. 清算服务（Clearing Service）

#### 功能描述
负责交易清算、结算、日终清算等。

#### 核心功能
- ✅ 实时清算
- ✅ 日终清算（EOD Clearing）
- ✅ 清算记录管理
- ✅ 清算状态跟踪
- ✅ 清算报告生成

#### 领域模型
```go
type Settlement struct {
    SettlementID   string          // 清算 ID
    TradeID        string          // 交易 ID
    BuyUserID      string          // 买方用户 ID
    SellUserID     string          // 卖方用户 ID
    Symbol         string          // 交易对
    Quantity       decimal.Decimal // 数量
    Price          decimal.Decimal // 价格
    Status         string          // 状态
    SettlementTime time.Time       // 清算时间
}

type EODClearing struct {
    ClearingID    string     // 清算 ID
    ClearingDate  string     // 清算日期
    Status        string     // 状态（PENDING/PROCESSING/COMPLETED）
    StartTime     time.Time  // 开始时间
    EndTime       *time.Time // 结束时间
    TradesSettled int64      // 已清算交易数
    TotalTrades   int64      // 总交易数
}
```

---

### 7. 持仓服务（Position Service）

#### 功能描述
管理用户持仓、盈亏计算、持仓查询。

#### 核心功能
- ✅ 持仓管理
- ✅ 实时盈亏计算
- ✅ 持仓查询
- ✅ 平仓操作
- ✅ 持仓历史

#### 领域模型
```go
type Position struct {
    PositionID    string          // 持仓 ID
    UserID        string          // 用户 ID
    Symbol        string          // 交易对
    Side          string          // 方向（LONG/SHORT）
    Quantity      decimal.Decimal // 数量
    EntryPrice    decimal.Decimal // 开仓价格
    CurrentPrice  decimal.Decimal // 当前价格
    UnrealizedPnL decimal.Decimal // 未实现盈亏
    RealizedPnL   decimal.Decimal // 已实现盈亏
    OpenedAt      time.Time       // 开仓时间
    ClosedAt      *time.Time      // 平仓时间
    Status        string          // 状态（OPEN/CLOSED）
}
```

---

### 8. 量化服务（Quant Service）

#### 功能描述
提供量化交易相关功能，包括技术指标计算、回测、策略执行。

#### 核心功能
- ✅ 技术指标计算（MA、EMA、RSI、MACD、布林带等）
- ✅ 回测框架
- ✅ 策略信号生成
- ✅ 策略性能分析

#### 技术指标
```go
// 移动平均线（MA）
func CalculateMA(prices []decimal.Decimal, period int) []decimal.Decimal

// 指数移动平均线（EMA）
func CalculateEMA(prices []decimal.Decimal, period int) []decimal.Decimal

// 相对强弱指标（RSI）
func CalculateRSI(prices []decimal.Decimal, period int) []decimal.Decimal

// MACD
func CalculateMACD(prices []decimal.Decimal, fastPeriod, slowPeriod, signalPeriod int) (macd, signal, histogram []decimal.Decimal)

// 布林带
func CalculateBollingerBands(prices []decimal.Decimal, period int, stdDev float64) (upper, middle, lower []decimal.Decimal)
```

---

### 9. 定价服务（Pricing Service）

#### 功能描述
期权定价、衍生品定价。

#### 核心功能
- ✅ Black-Scholes 期权定价
- ✅ 希腊字母计算（Delta、Gamma、Vega、Theta、Rho）
- ✅ 隐含波动率计算
- ✅ 期权策略分析

#### Black-Scholes 模型
```go
type BlackScholesCalculator struct{}

// 计算看涨期权价格
func (bsc *BlackScholesCalculator) CalculateCallPrice(
    S, K, T, r, sigma, q decimal.Decimal,
) (decimal.Decimal, error)

// 计算看跌期权价格
func (bsc *BlackScholesCalculator) CalculatePutPrice(
    S, K, T, r, sigma, q decimal.Decimal,
) (decimal.Decimal, error)

// 计算 Delta
func (bsc *BlackScholesCalculator) CalculateDelta(
    optionType string, S, K, T, r, sigma, q decimal.Decimal,
) (decimal.Decimal, error)

// 计算隐含波动率
func (bsc *BlackScholesCalculator) CalculateImpliedVolatility(
    optionType string, S, K, T, r, q, marketPrice decimal.Decimal,
) (decimal.Decimal, error)
```

---

### 10. 做市商服务（Market Making Service）

#### 功能描述
自动做市、流动性提供。

#### 核心功能
- ✅ 网格交易策略
- ✅ 双边报价
- ✅ 库存管理
- ✅ 价差管理
- ✅ 做市商绩效分析

#### 网格策略
```go
type GridStrategy struct {
    Symbol      string          // 交易对
    GridLevels  int             // 网格层数
    GridSize    decimal.Decimal // 网格大小
    BasePrice   decimal.Decimal // 基准价格
    TotalAmount decimal.Decimal // 总投入金额
}

// 生成网格订单
func (gs *GridStrategy) GenerateGridOrders() []*Order
```

---

### 11. 市场模拟服务（Market Simulation Service）

#### 功能描述
市场数据模拟、压力测试。

#### 核心功能
- ✅ 几何布朗运动（GBM）模拟
- ✅ 蒙特卡洛模拟
- ✅ 价格路径生成
- ✅ 市场场景模拟

#### 模拟算法
```go
type GeometricBrownianMotion struct {
    initialPrice decimal.Decimal // 初始价格
    drift        decimal.Decimal // 漂移率
    volatility   decimal.Decimal // 波动率
    timeStep     decimal.Decimal // 时间步长
}

// 模拟价格路径
func (gbm *GeometricBrownianMotion) Simulate(steps int) []decimal.Decimal

// 蒙特卡洛模拟
type MonteCarlo struct {
    gbm *GeometricBrownianMotion
}

// 计算期权价格
func (mc *MonteCarlo) CalculateOptionPrice(
    optionType string, strikePrice decimal.Decimal, steps, paths int, riskFreeRate decimal.Decimal,
) (decimal.Decimal, error)
```

---

## 领域驱动设计

### DDD 分层架构

每个微服务内部采用标准的 DDD 四层架构：

```
service/
├── application/        # 应用层
│   └── service.go     # 应用服务（用例编排）
├── domain/            # 领域层
│   ├── model.go       # 领域模型（实体、值对象）
│   ├── repository.go  # 仓储接口
│   └── service.go     # 领域服务
├── infrastructure/    # 基础设施层
│   └── repository/    # 仓储实现
│       └── impl.go
└── interfaces/        # 接口层
    ├── grpc_handler.go # gRPC 处理器
    └── http_handler.go # HTTP 处理器
```

### 各层职责

#### 1. 接口层（Interfaces）
- **职责**：对外暴露 API，处理请求和响应
- **组件**：gRPC Handler、HTTP Handler
- **示例**：
```go
type OrderHandler struct {
    order.UnimplementedOrderServiceServer
    useCase *application.OrderApplicationService
    logger  *zap.Logger
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *order.CreateOrderRequest) (*order.CreateOrderResponse, error) {
    // 1. 参数验证
    // 2. 调用应用服务
    // 3. 返回响应
}
```

#### 2. 应用层（Application）
- **职责**：用例编排，协调领域对象完成业务流程
- **组件**：Application Service、DTO
- **示例**：
```go
type OrderApplicationService struct {
    orderRepo domain.OrderRepository
    snowflake *utils.SnowflakeID
}

func (oas *OrderApplicationService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
    // 1. 验证输入参数
    // 2. 生成订单 ID
    // 3. 创建订单领域对象
    // 4. 保存到仓储
    // 5. 发布事件
    // 6. 返回 DTO
}
```

#### 3. 领域层（Domain）
- **职责**：核心业务逻辑，领域模型
- **组件**：实体（Entity）、值对象（Value Object）、领域服务、仓储接口
- **示例**：
```go
// 实体
type Order struct {
    OrderID   string
    UserID    string
    Symbol    string
    Side      OrderSide
    Type      OrderType
    Price     decimal.Decimal
    Quantity  decimal.Decimal
    Status    OrderStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}

// 领域方法
func (o *Order) CanBeCancelled() bool {
    return o.Status == OrderStatusNew || o.Status == OrderStatusPartiallyFilled
}

func (o *Order) IsFilled() bool {
    return o.FilledQuantity.Equal(o.Quantity)
}

// 仓储接口
type OrderRepository interface {
    Save(order *Order) error
    Get(orderID string) (*Order, error)
    ListByUser(userID string, status OrderStatus, limit, offset int) ([]*Order, int64, error)
    UpdateStatus(orderID string, status OrderStatus) error
}
```

#### 4. 基础设施层（Infrastructure）
- **职责**：技术实现，数据持久化
- **组件**：仓储实现、数据库模型、外部服务客户端
- **示例**：
```go
type OrderRepositoryImpl struct {
    db *db.DB
}

func (or *OrderRepositoryImpl) Save(order *domain.Order) error {
    model := &OrderModel{
        OrderID:  order.OrderID,
        UserID:   order.UserID,
        Symbol:   order.Symbol,
        // ... 其他字段
    }
    return or.db.Create(model).Error
}

func (or *OrderRepositoryImpl) Get(orderID string) (*domain.Order, error) {
    var model OrderModel
    if err := or.db.Where("order_id = ?", orderID).First(&model).Error; err != nil {
        return nil, err
    }
    return or.modelToDomain(&model), nil
}
```

### DDD 核心概念

#### 实体（Entity）
具有唯一标识的对象，如 Order、Account、Position。

#### 值对象（Value Object）
没有唯一标识，通过属性值来区分，如 Money、Price、Quantity。

#### 聚合（Aggregate）
一组相关对象的集合，有一个聚合根（Aggregate Root）。

#### 仓储（Repository）
提供领域对象的持久化和查询接口。

#### 领域服务（Domain Service）
不属于任何实体的业务逻辑。

#### 领域事件（Domain Event）
领域中发生的重要事件，如 OrderCreated、OrderFilled。

---

## 基础设施

### 1. 数据库（MySQL）

#### 连接池配置
```go
type DatabaseConfig struct {
    Driver              string
    DSN                 string
    MaxOpenConns        int  // 最大连接数：25
    MaxIdleConns        int  // 最大空闲连接数：5
    ConnMaxLifetime     int  // 连接最大生命周期：300秒
    LogEnabled          bool
    SlowQueryThreshold  int  // 慢查询阈值：1000ms
}
```

#### 事务支持
```go
// 自动事务管理
func (d *DB) WithTx(ctx context.Context, fn func(*gorm.DB) error) error {
    return d.DB.Transaction(func(tx *gorm.DB) error {
        return fn(tx)
    })
}

// 指定隔离级别
func (d *DB) WithTxIsolation(ctx context.Context, isolation string, fn func(*gorm.DB) error) error
```

#### 批量操作
```go
// 批量插入
func (d *DB) BatchInsert(ctx context.Context, records interface{}, batchSize int) error

// 批量更新
func (d *DB) BatchUpdate(ctx context.Context, model interface{}, updates map[string]interface{}, conditions map[string]interface{}) error
```

---

### 2. 缓存（Redis）

#### 功能特性
- ✅ 连接池管理
- ✅ 二级缓存（本地缓存 + Redis）
- ✅ 自动重试
- ✅ 分布式锁（SetNX）
- ✅ 支持多种数据结构（String、Hash、List、Set、ZSet）

#### 配置
```go
type RedisConfig struct {
    Host         string
    Port         int
    Password     string
    DB           int
    MaxPoolSize  int           // 最大连接数：10
    ConnTimeout  int           // 连接超时：5秒
    ReadTimeout  int           // 读超时：3秒
    WriteTimeout int           // 写超时：3秒
}
```

#### 使用示例
```go
// 基本操作
cache.Set(ctx, "key", "value", 10*time.Minute)
value, err := cache.Get(ctx, "key")

// JSON 操作
cache.SetJSON(ctx, "user:123", user, 1*time.Hour)
var user User
cache.GetJSON(ctx, "user:123", &user)

// 分布式锁
ok, err := cache.SetNX(ctx, "lock:order:123", "locked", 30*time.Second)

// 原子操作
cache.Incr(ctx, "counter")
cache.IncrBy(ctx, "counter", 10)

// 哈希操作
cache.HSet(ctx, "user:123", "name", "Alice", "age", 30)
cache.HGetAll(ctx, "user:123")

// 有序集合
cache.ZAdd(ctx, "leaderboard", redis.Z{Score: 100, Member: "user1"})
cache.ZRange(ctx, "leaderboard", 0, 9)
```

---

### 3. 消息队列（Kafka）

#### 功能特性
- ✅ 生产者（同步/异步发送）
- ✅ 消费者（单条/批量消费）
- ✅ 消费者组
- ✅ 自动重试
- ✅ 死信队列（DLQ）
- ✅ 消息压缩

#### 配置
```go
type KafkaConfig struct {
    Brokers          []string
    GroupID          string
    Partitions       int  // 分区数：3
    Replication      int  // 副本数：1
    SessionTimeout   int  // 会话超时：10秒
    MaxRetries       int  // 最大重试次数：3
    RetryBackoff     int  // 重试延迟：100ms
    EnableCompression bool
}
```

#### 使用示例
```go
// 生产者
producer, _ := mq.NewProducer(cfg)
producer.SendMessage(ctx, "orders", "order-123", orderData)
producer.SendMessages(ctx, "orders", []map[string]interface{}{...})

// 消费者
consumer, _ := mq.NewConsumer(cfg, "orders")
message, _ := consumer.ReadMessage(ctx)
messages, _ := consumer.ReadMessages(ctx, 100)

// 提交偏移量
consumer.CommitMessages(ctx, messages...)

// 死信队列
dlq := mq.NewDeadLetterQueue(producer, "orders-dlq")
dlq.Send(ctx, originalMessage, "processing failed", err)
```

---

### 4. 日志（Zap）

#### 功能特性
- ✅ 结构化日志
- ✅ 多级别（Debug、Info、Warn、Error、Fatal）
- ✅ 多输出（控制台、文件）
- ✅ 日志轮转
- ✅ 上下文追踪（trace_id、span_id）
- ✅ 性能优化（零分配）

#### 配置
```go
type LoggerConfig struct {
    Level          string // debug, info, warn, error
    Format         string // json, console
    Output         string // stdout, file, both
    FilePath       string
    MaxSize        int    // MB
    MaxBackups     int
    MaxAge         int    // 天
    Compress       bool
    WithCaller     bool
    WithStacktrace bool
}
```

#### 使用示例
```go
// 基本日志
logger.Info("order created", 
    zap.String("order_id", orderID),
    zap.String("user_id", userID),
    zap.String("symbol", symbol))

// 带上下文的日志
logger.WithContext(ctx).Info("processing order")

// 带字段的日志
logger.WithFields(
    zap.String("service", "order"),
    zap.String("version", "1.0.0"),
).Info("service started")

// 记录耗时
defer logger.LogDuration("process_order", 
    zap.String("order_id", orderID))()

// 错误日志
logger.Error("failed to create order", 
    zap.Error(err),
    zap.String("order_id", orderID))
```

---

### 5. 指标（Prometheus）

#### 指标类型
- **Counter**：计数器（只增不减）
- **Gauge**：仪表（可增可减）
- **Histogram**：直方图（分布统计）
- **Summary**：摘要（分位数）

#### 内置指标
```go
// HTTP 请求
HTTPRequestsTotal       // 请求总数
HTTPRequestDuration     // 请求耗时
HTTPRequestSize         // 请求大小
HTTPResponseSize        // 响应大小

// gRPC 请求
GRPCRequestsTotal       // 请求总数
GRPCRequestDuration     // 请求耗时

// 数据库
DBQueriesTotal          // 查询总数
DBQueryDuration         // 查询耗时
DBConnections           // 连接数

// Redis
RedisOpsTotal           // 操作总数
RedisOpDuration         // 操作耗时

// 业务指标
OrdersTotal             // 订单总数
OrdersActive            // 活跃订单数
TradesTotal             // 交易总数
PositionsActive         // 活跃持仓数
```

#### 使用示例
```go
// 记录 HTTP 请求
metrics.RecordHTTPRequest("POST", "/api/orders", 200, duration, requestSize, responseSize)

// 记录 gRPC 请求
metrics.RecordGRPCRequest("CreateOrder", duration)

// 记录数据库查询
metrics.RecordDBQuery(duration)

// 记录业务指标
metrics.RecordOrder()
metrics.UpdateActiveOrders(count)
```

---

### 6. 分布式追踪（Jaeger）

#### 功能特性
- ✅ 请求链路追踪
- ✅ 服务依赖分析
- ✅ 性能瓶颈定位
- ✅ 错误追踪

#### 配置
```go
type TracingConfig struct {
    Enabled        bool
    Type           string  // jaeger, otlp
    JaegerEndpoint string
    SamplingRate   float64 // 0.1 = 10%
}
```

#### 追踪信息
- **trace_id**：全局唯一的追踪 ID
- **span_id**：单个操作的 ID
- **parent_span_id**：父操作的 ID
- **tags**：标签（service、method、status 等）
- **logs**：日志事件

---

## 部署架构

### 本地开发环境（Docker Compose）

#### 启动服务
```bash
# 启动所有基础设施
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f [service_name]

# 停止服务
docker-compose down
```

#### 服务列表
- MySQL: `localhost:3306`
- Redis: `localhost:6379`
- Kafka: `localhost:9092`
- Zookeeper: `localhost:2181`
- Jaeger UI: `http://localhost:16686`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

---

### 生产环境（Kubernetes + Helm）

#### 部署架构
```
┌─────────────────────────────────────────────────────────────┐
│                      Ingress / Istio Gateway                 │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      Service Mesh (Istio)                    │
│  - 流量管理（路由、负载均衡、熔断）                              │
│  - 安全（mTLS、认证、授权）                                     │
│  - 可观测性（指标、日志、追踪）                                  │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      Kubernetes Services                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Order   │  │ Matching │  │  Market  │  │   Risk   │   │
│  │ Service  │  │  Engine  │  │   Data   │  │  Service │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Account  │  │ Position │  │ Clearing │  │   Quant  │   │
│  │ Service  │  │ Service  │  │ Service  │  │  Service │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      Persistent Storage                      │
│  - MySQL (StatefulSet)                                       │
│  - Redis (StatefulSet)                                       │
│  - Kafka (StatefulSet)                                       │
└─────────────────────────────────────────────────────────────┘
```

#### Helm Chart 结构
```
deployments/market-data/helm/
├── Chart.yaml              # Chart 元数据
├── values.yaml             # 默认配置
└── templates/
    ├── deployment.yaml     # Deployment
    ├── service.yaml        # Service
    ├── configmap.yaml      # ConfigMap
    ├── hpa.yaml            # HorizontalPodAutoscaler
    ├── pdb.yaml            # PodDisruptionBudget
    ├── networkpolicy.yaml  # NetworkPolicy
    ├── istio-gateway.yaml  # Istio Gateway
    ├── virtualservice.yaml # Istio VirtualService
    └── destinationrule.yaml # Istio DestinationRule
```

#### 部署命令
```bash
# 安装 Chart
helm install market-data ./deployments/market-data/helm

# 升级 Chart
helm upgrade market-data ./deployments/market-data/helm

# 卸载 Chart
helm uninstall market-data

# 查看状态
kubectl get pods -l app=market-data
kubectl get svc market-data
```

#### 高可用配置
- **副本数**：每个服务至少 3 个副本
- **反亲和性**：Pod 分散在不同节点
- **健康检查**：Liveness 和 Readiness Probe
- **资源限制**：CPU 和内存限制
- **自动伸缩**：HPA 基于 CPU/内存/自定义指标
- **滚动更新**：零停机部署
- **PDB**：保证最小可用副本数

---

## API 接口

### gRPC 接口

#### 订单服务（Order Service）
```protobuf
service OrderService {
    rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
    rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
    rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
    rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
    rpc ModifyOrder(ModifyOrderRequest) returns (ModifyOrderResponse);
}

message CreateOrderRequest {
    string user_id = 1;
    string symbol = 2;
    string side = 3;           // BUY, SELL
    string order_type = 4;     // LIMIT, MARKET, STOP
    string price = 5;
    string quantity = 6;
    string time_in_force = 7;  // GTC, IOC, FOK
    string client_order_id = 8;
}

message OrderResponse {
    string order_id = 1;
    string user_id = 2;
    string symbol = 3;
    string side = 4;
    string order_type = 5;
    string price = 6;
    string quantity = 7;
    string filled_quantity = 8;
    string status = 9;
    string time_in_force = 10;
    int64 created_at = 11;
    int64 updated_at = 12;
    string error_message = 13;
}
```

#### 行情数据服务（Market Data Service）
```protobuf
service MarketDataService {
    rpc GetLatestQuote(GetLatestQuoteRequest) returns (QuoteResponse);
    rpc GetKlines(GetKlinesRequest) returns (KlinesResponse);
    rpc GetOrderBook(GetOrderBookRequest) returns (OrderBookResponse);
    rpc GetTrades(GetTradesRequest) returns (TradesResponse);
    rpc SubscribeQuotes(SubscribeQuotesRequest) returns (stream QuoteResponse);
}

message GetLatestQuoteRequest {
    string symbol = 1;
}

message QuoteResponse {
    string symbol = 1;
    double bid_price = 2;
    double ask_price = 3;
    double bid_size = 4;
    double ask_size = 5;
    double last_price = 6;
    double last_size = 7;
    int64 timestamp = 8;
}
```

### HTTP REST 接口

#### 行情数据服务
```
GET /api/v1/market-data/quote?symbol=BTC/USDT
GET /api/v1/market-data/quotes?symbol=BTC/USDT&start_time=xxx&end_time=xxx
GET /api/v1/market-data/klines?symbol=BTC/USDT&interval=1h&limit=100
GET /api/v1/market-data/orderbook?symbol=BTC/USDT&depth=20
GET /api/v1/market-data/trades?symbol=BTC/USDT&limit=100
```

---

## 数据模型

### 核心表结构

#### 订单表（orders）
```sql
CREATE TABLE orders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id VARCHAR(50) UNIQUE NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL,
    type VARCHAR(20) NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    filled_quantity DECIMAL(20,8) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL,
    time_in_force VARCHAR(10) NOT NULL,
    client_order_id VARCHAR(100),
    remark TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_symbol (symbol),
    INDEX idx_status (status),
    INDEX idx_client_order_id (client_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 账户表（accounts）
```sql
CREATE TABLE accounts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    account_id VARCHAR(50) UNIQUE NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    account_type VARCHAR(20) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    balance DECIMAL(20,8) NOT NULL DEFAULT 0,
    available_balance DECIMAL(20,8) NOT NULL DEFAULT 0,
    frozen_balance DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_account_type (account_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 持仓表（positions）
```sql
CREATE TABLE positions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    position_id VARCHAR(50) UNIQUE NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    entry_price DECIMAL(20,8) NOT NULL,
    current_price DECIMAL(20,8) NOT NULL,
    unrealized_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    realized_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    closed_at BIGINT,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_symbol (symbol),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 开发指南

### 环境准备

#### 1. 安装依赖
```bash
# Go 1.25+
go version

# Protocol Buffers 编译器
brew install protobuf

# gRPC 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Docker & Docker Compose
docker --version
docker-compose --version
```

#### 2. 克隆项目
```bash
git clone https://github.com/fynnwu/FinancialTrading.git
cd FinancialTrading
```

#### 3. 安装 Go 依赖
```bash
go mod download
```

#### 4. 启动基础设施
```bash
docker-compose up -d
```

### 开发流程

#### 1. 生成 gRPC 代码
```bash
# 生成所有服务的 gRPC 代码
./scripts/generate-pb.sh

# 或手动生成单个服务
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/order/order.proto
```

#### 2. 运行服务
```bash
# 运行订单服务
go run cmd/order/main.go

# 运行撮合引擎
go run cmd/matching-engine/main.go

# 运行行情数据服务
go run cmd/market-data/main.go
```

#### 3. 测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/order/...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### 4. 构建
```bash
# 构建所有服务
make build

# 构建单个服务
go build -o bin/order cmd/order/main.go

# 构建 Docker 镜像
docker build -f deployments/order/Dockerfile -t order-service:latest .
```

### 代码规范

#### 1. 命名规范
- **包名**：小写，简短，有意义（如 `order`、`account`）
- **文件名**：小写，下划线分隔（如 `order_service.go`）
- **类型名**：大驼峰（如 `OrderService`）
- **函数名**：大驼峰（公开）或小驼峰（私有）
- **常量**：大驼峰或全大写下划线分隔

#### 2. 注释规范
```go
// CreateOrder 创建订单
// 用例流程：
// 1. 验证输入参数
// 2. 生成订单 ID
// 3. 创建订单对象
// 4. 保存到仓储
// 5. 发布订单创建事件
func (oas *OrderApplicationService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
    // 实现...
}
```

#### 3. 错误处理
```go
// 使用 errors.New 或 fmt.Errorf
if err != nil {
    return nil, fmt.Errorf("failed to create order: %w", err)
}

// 使用自定义错误
if order == nil {
    return nil, utils.NewErrorWrapper("ORDER_NOT_FOUND", "order not found", nil)
}
```

#### 4. 日志规范
```go
// 使用结构化日志
logger.Info("order created",
    zap.String("order_id", orderID),
    zap.String("user_id", userID),
    zap.String("symbol", symbol),
    zap.String("side", side))

// 错误日志
logger.Error("failed to create order",
    zap.Error(err),
    zap.String("user_id", userID))
```

---

## 运维指南

### 监控

#### 1. Prometheus 指标
访问 `http://localhost:9090`

常用查询：
```promql
# HTTP 请求速率
rate(http_requests_total[5m])

# HTTP 请求延迟 P99
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# 活跃订单数
orders_active

# 数据库连接数
db_connections
```

#### 2. Grafana 仪表板
访问 `http://localhost:3000`（admin/admin）

预置仪表板：
- **系统概览**：CPU、内存、网络、磁盘
- **服务监控**：请求速率、延迟、错误率
- **业务指标**：订单量、交易量、持仓量
- **数据库监控**：查询速率、慢查询、连接数

#### 3. Jaeger 追踪
访问 `http://localhost:16686`

功能：
- 查看请求链路
- 分析服务依赖
- 定位性能瓶颈
- 错误追踪

### 日志

#### 1. 日志级别
- **Debug**：调试信息
- **Info**：一般信息
- **Warn**：警告信息
- **Error**：错误信息
- **Fatal**：致命错误（会退出程序）

#### 2. 日志查询
```bash
# 查看实时日志
tail -f logs/app.log

# 查询错误日志
grep "ERROR" logs/app.log

# 查询特定订单的日志
grep "order_id=123" logs/app.log

# 使用 jq 解析 JSON 日志
cat logs/app.log | jq 'select(.level == "error")'
```

### 告警

#### 1. Prometheus 告警规则
```yaml
groups:
  - name: trading_system
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} for {{ $labels.service }}"

      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "P99 latency is {{ $value }}s for {{ $labels.service }}"
```

### 备份与恢复

#### 1. MySQL 备份
```bash
# 全量备份
mysqldump -u trading -p trading_system > backup_$(date +%Y%m%d).sql

# 恢复
mysql -u trading -p trading_system < backup_20240101.sql
```

#### 2. Redis 备份
```bash
# RDB 快照
redis-cli SAVE

# AOF 持久化
redis-cli BGREWRITEAOF
```

### 性能优化

#### 1. 数据库优化
- 添加索引
- 优化查询
- 使用连接池
- 读写分离
- 分库分表

#### 2. 缓存优化
- 热点数据缓存
- 缓存预热
- 缓存穿透防护
- 缓存雪崩防护

#### 3. 服务优化
- 异步处理
- 批量操作
- 连接复用
- 限流熔断

---

## 总结

这是一个功能完整、架构清晰、可扩展性强的金融交易系统。主要特点：

1. **微服务架构**：15+ 独立服务，职责清晰
2. **领域驱动设计**：清晰的领域模型，业务逻辑与技术实现分离
3. **高性能**：撮合引擎、缓存、异步处理
4. **高可用**：多副本、健康检查、自动伸缩
5. **可观测性**：日志、指标、追踪
6. **完整的业务功能**：订单、撮合、清算、风险、量化、定价等

适用场景：
- 数字货币交易所
- 证券交易系统
- 期货交易系统
- 量化交易平台
- 金融科技创新

---

## 附录

### 常用命令

```bash
# 启动所有基础设施
docker-compose up -d

# 生成 gRPC 代码
./scripts/generate-pb.sh

# 运行服务
go run cmd/order/main.go

# 运行测试
go test ./...

# 构建
make build

# 部署
helm install market-data ./deployments/market-data/helm

# 查看日志
kubectl logs -f pod/market-data-xxx

# 查看指标
curl http://localhost:9090/metrics
```

### 参考资料

- [Go 官方文档](https://golang.org/doc/)
- [gRPC 官方文档](https://grpc.io/docs/)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [GORM 文档](https://gorm.io/docs/)
- [Kubernetes 文档](https://kubernetes.io/docs/)
- [Istio 文档](https://istio.io/docs/)
- [Prometheus 文档](https://prometheus.io/docs/)

---

**文档版本**：1.0  
**最后更新**：2024-01-01  
**维护者**：FinancialTrading Team
