// 变更说明：完善智能订单路由（SOR）领域模型，增加多种路由策略、流动性评估、延迟优化、成本分析等完整功能
package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// RoutingStrategy 路由策略类型
type RoutingStrategy int

const (
	StrategyBestPrice RoutingStrategy = iota // 最优价格
	StrategyVWAP                              // 成交量加权平均价格
	StrategyTWAP                              // 时间加权平均价格
	StrategyPOV                               // 成交量占比
	StrategyIS                                // 实施差额
	StrategyMinImpact                         // 最小市场冲击
	StrategyDarkPool                          // 暗池优先
)

func (s RoutingStrategy) String() string {
	switch s {
	case StrategyBestPrice:
		return "BEST_PRICE"
	case StrategyVWAP:
		return "VWAP"
	case StrategyTWAP:
		return "TWAP"
	case StrategyPOV:
		return "POV"
	case StrategyIS:
		return "IMPLEMENTATION_SHORTFALL"
	case StrategyMinImpact:
		return "MIN_IMPACT"
	case StrategyDarkPool:
		return "DARK_POOL"
	default:
		return "UNKNOWN"
	}
}

// OrderSide 订单方向
type OrderSide int

const (
	SideBuy OrderSide = iota
	SideSell
)

func (s OrderSide) String() string {
	if s == SideBuy {
		return "BUY"
	}
	return "SELL"
}

// Venue 交易场所
type Venue struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         VenueType `json:"type"`
	Region       string    `json:"region"`
	Currency     string    `json:"currency"`
	Latency      time.Duration `json:"latency"`
	FeeRate      float64   `json:"fee_rate"`
	IsActive     bool      `json:"is_active"`
	IsDarkPool   bool      `json:"is_dark_pool"`
	MinOrderSize int64     `json:"min_order_size"`
	MaxOrderSize int64     `json:"max_order_size"`
}

// VenueType 交易场所类型
type VenueType int

const (
	VenueTypeExchange VenueType = iota
	VenueTypeDarkPool
	VenueTypeECN
	VenueTypeOTC
)

// MarketDepth 市场深度
type MarketDepth struct {
	VenueID     string        `json:"venue_id"`
	Symbol      string        `json:"symbol"`
	Bids        []PriceLevel  `json:"bids"`
	Asks        []PriceLevel  `json:"asks"`
	Timestamp   time.Time     `json:"timestamp"`
	Latency     time.Duration `json:"latency"`
	Liquidity   float64       `json:"liquidity"`
	Spread      float64       `json:"spread"`
	Volume24h   int64         `json:"volume_24h"`
}

// PriceLevel 价格档位
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity int64   `json:"quantity"`
	Orders   int     `json:"orders"`
}

// OrderRoute 订单路由
type OrderRoute struct {
	RouteID      string    `json:"route_id"`
	VenueID      string    `json:"venue_id"`
	VenueName    string    `json:"venue_name"`
	Price        float64   `json:"price"`
	Quantity     int64     `json:"quantity"`
	ExpectedFill float64   `json:"expected_fill"`
	Fee          float64   `json:"fee"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}

// SORPlan 智能路由计划
type SORPlan struct {
	PlanID          string        `json:"plan_id"`
	ParentOrderID   string        `json:"parent_order_id"`
	Symbol          string        `json:"symbol"`
	Side            OrderSide     `json:"side"`
	TotalQuantity   int64         `json:"total_quantity"`
	Strategy        RoutingStrategy `json:"strategy"`
	Routes          []OrderRoute  `json:"routes"`
	AveragePrice    float64       `json:"average_price"`
	TotalFee        float64       `json:"total_fee"`
	ExpectedCost    float64       `json:"expected_cost"`
	MarketImpact    float64       `json:"market_impact"`
	ConfidenceScore float64       `json:"confidence_score"`
	GeneratedAt     time.Time     `json:"generated_at"`
	ExpiresAt       time.Time     `json:"expires_at"`
	Status          PlanStatus    `json:"status"`
}

// PlanStatus 计划状态
type PlanStatus int

const (
	PlanStatusPending PlanStatus = iota
	PlanStatusExecuting
	PlanStatusCompleted
	PlanStatusPartial
	PlanStatusCancelled
	PlanStatusExpired
)

// ExecutionReport 执行报告
type ExecutionReport struct {
	ReportID    string          `json:"report_id"`
	PlanID      string          `json:"plan_id"`
	RouteID     string          `json:"route_id"`
	VenueID     string          `json:"venue_id"`
	FilledQty   int64           `json:"filled_qty"`
	FilledPrice float64         `json:"filled_price"`
	Fee         float64         `json:"fee"`
	Slippage    float64         `json:"slippage"`
	Latency     time.Duration   `json:"latency"`
	ExecutedAt  time.Time       `json:"executed_at"`
}

// RoutingRequest 路由请求
type RoutingRequest struct {
	RequestID     string          `json:"request_id"`
	Symbol        string          `json:"symbol"`
	Side          OrderSide       `json:"side"`
	Quantity      int64           `json:"quantity"`
	Strategy      RoutingStrategy `json:"strategy"`
	MaxVenues     int             `json:"max_venues"`
	MinFillRate   float64         `json:"min_fill_rate"`
	MaxSlippage   float64         `json:"max_slippage"`
	TimeLimit     time.Duration   `json:"time_limit"`
	AllowDarkPool bool            `json:"allow_dark_pool"`
	VenueFilter   []string        `json:"venue_filter"`
}

// RoutingResult 路由结果
type RoutingResult struct {
	Plan        *SORPlan        `json:"plan"`
	Execution   *ExecutionStats `json:"execution"`
	Warnings    []string        `json:"warnings"`
}

// ExecutionStats 执行统计
type ExecutionStats struct {
	TotalFilled      int64         `json:"total_filled"`
	AveragePrice     float64       `json:"average_price"`
	TotalFee         float64       `json:"total_fee"`
	Slippage         float64       `json:"slippage"`
	MarketImpact     float64       `json:"market_impact"`
	ExecutionTime    time.Duration `json:"execution_time"`
	VenuesUsed       int           `json:"venues_used"`
	FillRate         float64       `json:"fill_rate"`
}

// LiquidityScore 流动性评分
type LiquidityScore struct {
	VenueID         string    `json:"venue_id"`
	Score           float64   `json:"score"`
	DepthScore      float64   `json:"depth_score"`
	SpreadScore     float64   `json:"spread_score"`
	VolumeScore     float64   `json:"volume_score"`
	ReliabilityScore float64  `json:"reliability_score"`
	Timestamp       time.Time `json:"timestamp"`
}

// SOREngine 智能路由引擎接口
type SOREngine interface {
	// AggregateDepths 聚合市场深度
	AggregateDepths(ctx context.Context, symbol string, venues []string) ([]*MarketDepth, error)
	
	// CreateRoutingPlan 创建路由计划
	CreateRoutingPlan(ctx context.Context, req *RoutingRequest) (*SORPlan, error)
	
	// CalculateLiquidityScore 计算流动性评分
	CalculateLiquidityScore(ctx context.Context, venueID, symbol string) (*LiquidityScore, error)
	
	// EstimateMarketImpact 估算市场冲击
	EstimateMarketImpact(ctx context.Context, symbol string, side OrderSide, quantity int64) (float64, error)
	
	// OptimizeRoutes 优化路由
	OptimizeRoutes(ctx context.Context, plan *SORPlan) (*SORPlan, error)
}

// DefaultSOREngine 默认智能路由引擎实现
type DefaultSOREngine struct {
	venues      map[string]*Venue
	depthCache  sync.Map
	scoreCache  sync.Map
	logger      interface{}
}

// NewDefaultSOREngine 创建默认引擎
func NewDefaultSOREngine() *DefaultSOREngine {
	return &DefaultSOREngine{
		venues: make(map[string]*Venue),
	}
}

// AddVenue 添加交易场所
func (e *DefaultSOREngine) AddVenue(venue *Venue) {
	e.venues[venue.ID] = venue
}

// AggregateDepths 聚合市场深度
func (e *DefaultSOREngine) AggregateDepths(ctx context.Context, symbol string, venues []string) ([]*MarketDepth, error) {
	var depths []*MarketDepth
	
	for _, venueID := range venues {
		if venue, ok := e.venues[venueID]; ok && venue.IsActive {
			depth := &MarketDepth{
				VenueID:   venueID,
				Symbol:    symbol,
				Timestamp: time.Now(),
				Latency:   venue.Latency,
			}
			
			if cached, ok := e.depthCache.Load(venueID + ":" + symbol); ok {
				depth = cached.(*MarketDepth)
			}
			
			depths = append(depths, depth)
		}
	}
	
	return depths, nil
}

// CreateRoutingPlan 创建路由计划
func (e *DefaultSOREngine) CreateRoutingPlan(ctx context.Context, req *RoutingRequest) (*SORPlan, error) {
	venues := req.VenueFilter
	if len(venues) == 0 {
		for id, v := range e.venues {
			if v.IsActive && (!v.IsDarkPool || req.AllowDarkPool) {
				venues = append(venues, id)
			}
		}
	}
	
	depths, err := e.AggregateDepths(ctx, req.Symbol, venues)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate depths: %w", err)
	}
	
	plan := &SORPlan{
		PlanID:        fmt.Sprintf("SOR%d", time.Now().UnixNano()),
		Symbol:        req.Symbol,
		Side:          req.Side,
		TotalQuantity: req.Quantity,
		Strategy:      req.Strategy,
		Routes:        []OrderRoute{},
		GeneratedAt:   time.Now(),
		ExpiresAt:     time.Now().Add(req.TimeLimit),
		Status:        PlanStatusPending,
	}
	
	switch req.Strategy {
	case StrategyBestPrice:
		plan = e.bestPriceRouting(plan, depths, req)
	case StrategyVWAP:
		plan = e.vwapRouting(plan, depths, req)
	case StrategyTWAP:
		plan = e.twapRouting(plan, depths, req)
	case StrategyMinImpact:
		plan = e.minImpactRouting(plan, depths, req)
	default:
		plan = e.bestPriceRouting(plan, depths, req)
	}
	
	plan.AveragePrice = e.calculateAveragePrice(plan.Routes)
	plan.TotalFee = e.calculateTotalFee(plan.Routes)
	plan.ExpectedCost = plan.AveragePrice * float64(plan.TotalQuantity) + plan.TotalFee
	
	return plan, nil
}

// bestPriceRouting 最优价格路由
func (e *DefaultSOREngine) bestPriceRouting(plan *SORPlan, depths []*MarketDepth, req *RoutingRequest) *SORPlan {
	type level struct {
		venueID string
		price   float64
		qty     int64
	}
	
	var levels []level
	
	for _, d := range depths {
		levels = append(levels, e.extractLevels(d, plan.Side)...)
	}
	
	if plan.Side == SideBuy {
		sort.Slice(levels, func(i, j int) bool { return levels[i].price < levels[j].price })
	} else {
		sort.Slice(levels, func(i, j int) bool { return levels[i].price > levels[j].price })
	}
	
	remaining := req.Quantity
	priority := 1
	
	for _, l := range levels {
		if remaining <= 0 {
			break
		}
		
		fillQty := l.qty
		if fillQty > remaining {
			fillQty = remaining
		}
		
		venue := e.venues[l.venueID]
		venueName := l.venueID
		if venue != nil {
			venueName = venue.Name
		}
		
		plan.Routes = append(plan.Routes, OrderRoute{
			RouteID:      fmt.Sprintf("R%d", time.Now().UnixNano()),
			VenueID:      l.venueID,
			VenueName:    venueName,
			Price:        l.price,
			Quantity:     fillQty,
			ExpectedFill: 0.95,
			Fee:          l.price * float64(fillQty) * e.getFeeRate(l.venueID),
			Priority:     priority,
			CreatedAt:    time.Now(),
		})
		
		remaining -= fillQty
		priority++
	}
	
	return plan
}

// vwapRouting VWAP路由
func (e *DefaultSOREngine) vwapRouting(plan *SORPlan, depths []*MarketDepth, req *RoutingRequest) *SORPlan {
	totalVolume := int64(0)
	volumeByVenue := make(map[string]int64)
	
	for _, d := range depths {
		vol := d.Volume24h
		if vol > 0 {
			totalVolume += vol
			volumeByVenue[d.VenueID] = vol
		}
	}
	
	if totalVolume == 0 {
		return e.bestPriceRouting(plan, depths, req)
	}
	
	remaining := req.Quantity
	priority := 1
	
	for venueID, vol := range volumeByVenue {
		if remaining <= 0 {
			break
		}
		
		weight := float64(vol) / float64(totalVolume)
		allocQty := int64(float64(req.Quantity) * weight)
		
		if allocQty > remaining {
			allocQty = remaining
		}
		
		price := e.getBestPrice(venueID, depths, plan.Side)
		venue := e.venues[venueID]
		venueName := venueID
		if venue != nil {
			venueName = venue.Name
		}
		
		plan.Routes = append(plan.Routes, OrderRoute{
			RouteID:      fmt.Sprintf("R%d", time.Now().UnixNano()),
			VenueID:      venueID,
			VenueName:    venueName,
			Price:        price,
			Quantity:     allocQty,
			ExpectedFill: weight,
			Fee:          price * float64(allocQty) * e.getFeeRate(venueID),
			Priority:     priority,
			CreatedAt:    time.Now(),
		})
		
		remaining -= allocQty
		priority++
	}
	
	return plan
}

// twapRouting TWAP路由
func (e *DefaultSOREngine) twapRouting(plan *SORPlan, depths []*MarketDepth, req *RoutingRequest) *SORPlan {
	intervals := 10
	qtyPerInterval := req.Quantity / int64(intervals)
	
	for i := 0; i < intervals; i++ {
		remaining := qtyPerInterval
		if i == intervals-1 {
			remaining = req.Quantity - qtyPerInterval*int64(i)
		}
		
		for _, d := range depths {
			if remaining <= 0 {
				break
			}
			
			price := e.getBestPrice(d.VenueID, depths, plan.Side)
			venue := e.venues[d.VenueID]
			venueName := d.VenueID
			if venue != nil {
				venueName = venue.Name
			}
			
			plan.Routes = append(plan.Routes, OrderRoute{
				RouteID:      fmt.Sprintf("R%d-%d", time.Now().UnixNano(), i),
				VenueID:      d.VenueID,
				VenueName:    venueName,
				Price:        price,
				Quantity:     remaining,
				ExpectedFill: 1.0 / float64(intervals),
				Fee:          price * float64(remaining) * e.getFeeRate(d.VenueID),
				Priority:     i + 1,
				CreatedAt:    time.Now().Add(time.Duration(i) * time.Minute),
			})
			remaining = 0
		}
	}
	
	return plan
}

// minImpactRouting 最小冲击路由
func (e *DefaultSOREngine) minImpactRouting(plan *SORPlan, depths []*MarketDepth, req *RoutingRequest) *SORPlan {
	type venueScore struct {
		venueID string
		score   float64
		depth   *MarketDepth
	}
	
	var scores []venueScore
	
	for _, d := range depths {
		impact := e.calculateImpactScore(d, req.Quantity, plan.Side)
		scores = append(scores, venueScore{
			venueID: d.VenueID,
			score:   impact,
			depth:   d,
		})
	}
	
	sort.Slice(scores, func(i, j int) bool { return scores[i].score < scores[j].score })
	
	remaining := req.Quantity
	priority := 1
	
	for _, s := range scores {
		if remaining <= 0 {
			break
		}
		
		availableQty := e.getAvailableQuantity(s.depth, plan.Side)
		allocQty := int64(float64(availableQty) * 0.1)
		if allocQty > remaining {
			allocQty = remaining
		}
		
		price := e.getBestPrice(s.venueID, depths, plan.Side)
		venue := e.venues[s.venueID]
		venueName := s.venueID
		if venue != nil {
			venueName = venue.Name
		}
		
		plan.Routes = append(plan.Routes, OrderRoute{
			RouteID:      fmt.Sprintf("R%d", time.Now().UnixNano()),
			VenueID:      s.venueID,
			VenueName:    venueName,
			Price:        price,
			Quantity:     allocQty,
			ExpectedFill: 0.9,
			Fee:          price * float64(allocQty) * e.getFeeRate(s.venueID),
			Priority:     priority,
			CreatedAt:    time.Now(),
		})
		
		remaining -= allocQty
		priority++
	}
	
	return plan
}

// CalculateLiquidityScore 计算流动性评分
func (e *DefaultSOREngine) CalculateLiquidityScore(ctx context.Context, venueID, symbol string) (*LiquidityScore, error) {
	depth, err := e.getDepth(venueID, symbol)
	if err != nil {
		return nil, err
	}
	
	score := &LiquidityScore{
		VenueID:   venueID,
		Timestamp: time.Now(),
	}
	
	totalDepth := int64(0)
	for _, b := range depth.Bids {
		totalDepth += b.Quantity
	}
	for _, a := range depth.Asks {
		totalDepth += a.Quantity
	}
	
	score.DepthScore = math.Min(float64(totalDepth)/1000000, 1.0)
	
	if len(depth.Bids) > 0 && len(depth.Asks) > 0 {
		bestBid := depth.Bids[0].Price
		bestAsk := depth.Asks[0].Price
		if bestBid > 0 {
			score.SpreadScore = 1.0 - (depth.Spread / bestBid * 100)
		}
	}
	
	score.VolumeScore = math.Min(float64(depth.Volume24h)/10000000, 1.0)
	score.ReliabilityScore = 1.0 - float64(depth.Latency.Milliseconds())/1000
	
	score.Score = (score.DepthScore + score.SpreadScore + score.VolumeScore + score.ReliabilityScore) / 4
	
	e.scoreCache.Store(venueID+":"+symbol, score)
	
	return score, nil
}

// EstimateMarketImpact 估算市场冲击
func (e *DefaultSOREngine) EstimateMarketImpact(ctx context.Context, symbol string, side OrderSide, quantity int64) (float64, error) {
	depths, err := e.AggregateDepths(ctx, symbol, nil)
	if err != nil {
		return 0, err
	}
	
	totalImpact := 0.0
	for _, d := range depths {
		totalImpact += e.calculateImpactScore(d, quantity, side)
	}
	
	if len(depths) > 0 {
		return totalImpact / float64(len(depths)), nil
	}
	return 0, nil
}

// OptimizeRoutes 优化路由
func (e *DefaultSOREngine) OptimizeRoutes(ctx context.Context, plan *SORPlan) (*SORPlan, error) {
	if len(plan.Routes) <= 1 {
		return plan, nil
	}
	
	sort.Slice(plan.Routes, func(i, j int) bool {
		return plan.Routes[i].Priority < plan.Routes[j].Priority
	})
	
	merged := make(map[string]*OrderRoute)
	for _, r := range plan.Routes {
		if existing, ok := merged[r.VenueID]; ok {
			existing.Quantity += r.Quantity
			existing.Fee += r.Fee
		} else {
			merged[r.VenueID] = &r
		}
	}
	
	plan.Routes = make([]OrderRoute, 0, len(merged))
	priority := 1
	for _, r := range merged {
		r.Priority = priority
		plan.Routes = append(plan.Routes, *r)
		priority++
	}
	
	return plan, nil
}

// 辅助方法

func (e *DefaultSOREngine) extractLevels(depth *MarketDepth, side OrderSide) []struct {
	venueID string
	price   float64
	qty     int64
} {
	var levels []struct {
		venueID string
		price   float64
		qty     int64
	}
	
	if side == SideBuy {
		for _, ask := range depth.Asks {
			levels = append(levels, struct {
				venueID string
				price   float64
				qty     int64
			}{depth.VenueID, ask.Price, ask.Quantity})
		}
	} else {
		for _, bid := range depth.Bids {
			levels = append(levels, struct {
				venueID string
				price   float64
				qty     int64
			}{depth.VenueID, bid.Price, bid.Quantity})
		}
	}
	
	return levels
}

func (e *DefaultSOREngine) getBestPrice(venueID string, depths []*MarketDepth, side OrderSide) float64 {
	for _, d := range depths {
		if d.VenueID == venueID {
			if side == SideBuy && len(d.Asks) > 0 {
				return d.Asks[0].Price
			} else if side == SideSell && len(d.Bids) > 0 {
				return d.Bids[0].Price
			}
		}
	}
	return 0
}

func (e *DefaultSOREngine) getAvailableQuantity(depth *MarketDepth, side OrderSide) int64 {
	var total int64
	if side == SideBuy {
		for _, ask := range depth.Asks {
			total += ask.Quantity
		}
	} else {
		for _, bid := range depth.Bids {
			total += bid.Quantity
		}
	}
	return total
}

func (e *DefaultSOREngine) calculateImpactScore(depth *MarketDepth, quantity int64, side OrderSide) float64 {
	available := e.getAvailableQuantity(depth, side)
	if available == 0 {
		return 1.0
	}
	return float64(quantity) / float64(available)
}

func (e *DefaultSOREngine) getFeeRate(venueID string) float64 {
	if venue, ok := e.venues[venueID]; ok {
		return venue.FeeRate
	}
	return 0.001
}

func (e *DefaultSOREngine) calculateAveragePrice(routes []OrderRoute) float64 {
	if len(routes) == 0 {
		return 0
	}
	
	var totalCost float64
	var totalQty int64
	
	for _, r := range routes {
		totalCost += r.Price * float64(r.Quantity)
		totalQty += r.Quantity
	}
	
	if totalQty == 0 {
		return 0
	}
	return totalCost / float64(totalQty)
}

func (e *DefaultSOREngine) calculateTotalFee(routes []OrderRoute) float64 {
	var total float64
	for _, r := range routes {
		total += r.Fee
	}
	return total
}

func (e *DefaultSOREngine) getDepth(venueID, symbol string) (*MarketDepth, error) {
	if cached, ok := e.depthCache.Load(venueID + ":" + symbol); ok {
		return cached.(*MarketDepth), nil
	}
	return &MarketDepth{
		VenueID:   venueID,
		Symbol:    symbol,
		Timestamp: time.Now(),
	}, nil
}
