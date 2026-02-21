//go:build settlement_experimental
// +build settlement_experimental

package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MarketType 市场类型
type MarketType string

const (
	MarketEquity     MarketType = "EQUITY"     // 股票市场
	MarketBond       MarketType = "BOND"       // 债券市场
	MarketFX         MarketType = "FX"         // 外汇市场
	MarketCommodity  MarketType = "COMMODITY"  // 商品市场
	MarketDerivative MarketType = "DERIVATIVE" // 衍生品市场
)

// MarketInfo 市场信息
type MarketInfo struct {
	MarketID        string           `json:"market_id"`
	MarketName      string           `json:"market_name"`
	MarketType      MarketType       `json:"market_type"`
	Country         string           `json:"country"`
	Currency        string           `json:"currency"`
	TimeZone        string           `json:"time_zone"`
	TradingHours    *TradingHours    `json:"trading_hours"`
	SettlementRules *SettlementRules `json:"settlement_rules"`
	Holidays        []time.Time      `json:"holidays"`
	Status          string           `json:"status"` // ACTIVE, SUSPENDED, CLOSED
}

// TradingHours 交易时间
type TradingHours struct {
	OpenTime  string `json:"open_time"`  // HH:MM
	CloseTime string `json:"close_time"` // HH:MM
	TimeZone  string `json:"time_zone"`
}

// SettlementRules 结算规则
type SettlementRules struct {
	DefaultCycle  SettlementCycle `json:"default_cycle"`
	CutOffTime    string          `json:"cut_off_time"` // HH:MM
	ValueDateRule string          `json:"value_date_rule"`
	HolidayRule   string          `json:"holiday_rule"`  // FOLLOWING, MODIFIED_FOLLOWING, PRECEDING
	BusinessDays  []string        `json:"business_days"` // MON, TUE, WED, THU, FRI
}

// CrossMarketSettlement 跨市场结算
type CrossMarketSettlement struct {
	ID                 string           `json:"id"`
	SettlementRef      string           `json:"settlement_ref"`
	TradeID            string           `json:"trade_id"`
	BuyMarket          string           `json:"buy_market"`
	SellMarket         string           `json:"sell_market"`
	Symbol             string           `json:"symbol"`
	Quantity           float64          `json:"quantity"`
	Price              float64          `json:"price"`
	BuyCurrency        string           `json:"buy_currency"`
	SellCurrency       string           `json:"sell_currency"`
	ExchangeRate       float64          `json:"exchange_rate"`
	Amount             float64          `json:"amount"`
	ConvertedAmount    float64          `json:"converted_amount"`
	BuySettlementDate  time.Time        `json:"buy_settlement_date"`
	SellSettlementDate time.Time        `json:"sell_settlement_date"`
	Status             SettlementStatus `json:"status"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

// CrossMarketManager 跨市场管理器
type CrossMarketManager struct {
	marketRepo        MarketRepository
	instructionRepo   SettlementInstructionRepository
	exchangeService   ExchangeRateService
	settlementManager *SettlementManager
	mu                sync.RWMutex
	markets           map[string]*MarketInfo
	settlements       map[string]*CrossMarketSettlement
}

// NewCrossMarketManager 创建跨市场管理器
func NewCrossMarketManager(marketRepo MarketRepository,
	instructionRepo SettlementInstructionRepository,
	exchangeService ExchangeRateService,
	settlementManager *SettlementManager) *CrossMarketManager {

	return &CrossMarketManager{
		marketRepo:        marketRepo,
		instructionRepo:   instructionRepo,
		exchangeService:   exchangeService,
		settlementManager: settlementManager,
		markets:           make(map[string]*MarketInfo),
		settlements:       make(map[string]*CrossMarketSettlement),
	}
}

// Initialize 初始化跨市场管理器
func (cmm *CrossMarketManager) Initialize(ctx context.Context) error {
	// 加载市场信息
	markets, err := cmm.marketRepo.GetActiveMarkets(ctx)
	if err != nil {
		return fmt.Errorf("failed to load markets: %w", err)
	}

	cmm.mu.Lock()
	for _, market := range markets {
		cmm.markets[market.MarketID] = market
	}
	cmm.mu.Unlock()

	return nil
}

// CreateCrossMarketSettlement 创建跨市场结算
func (cmm *CrossMarketManager) CreateCrossMarketSettlement(ctx context.Context, trade *Trade) (*CrossMarketSettlement, error) {
	// 获取市场信息
	buyMarket, sellMarket, err := cmm.getMarketInfo(trade.BuyMarket, trade.SellMarket)
	if err != nil {
		return nil, fmt.Errorf("failed to get market info: %w", err)
	}

	// 检查是否为跨市场交易
	if !cmm.isCrossMarket(buyMarket, sellMarket) {
		return nil, fmt.Errorf("not a cross-market trade")
	}

	// 计算结算日期
	buySettlementDate, sellSettlementDate, err := cmm.calculateSettlementDates(trade.TradeDate, buyMarket, sellMarket)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate settlement dates: %w", err)
	}

	// 获取汇率
	exchangeRate, err := cmm.getExchangeRate(ctx, buyMarket.Currency, sellMarket.Currency, trade.TradeDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// 计算转换金额
	amount := trade.Quantity * trade.Price
	convertedAmount := amount * exchangeRate

	// 创建跨市场结算
	settlement := &CrossMarketSettlement{
		ID:                 generateCrossMarketID(),
		SettlementRef:      generateCrossMarketRef(),
		TradeID:            trade.TradeID,
		BuyMarket:          trade.BuyMarket,
		SellMarket:         trade.SellMarket,
		Symbol:             trade.Symbol,
		Quantity:           trade.Quantity,
		Price:              trade.Price,
		BuyCurrency:        buyMarket.Currency,
		SellCurrency:       sellMarket.Currency,
		ExchangeRate:       exchangeRate,
		Amount:             amount,
		ConvertedAmount:    convertedAmount,
		BuySettlementDate:  buySettlementDate,
		SellSettlementDate: sellSettlementDate,
		Status:             SettlementPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// 创建结算指令
	err = cmm.createSettlementInstructions(ctx, settlement, trade)
	if err != nil {
		return nil, fmt.Errorf("failed to create settlement instructions: %w", err)
	}

	cmm.mu.Lock()
	cmm.settlements[settlement.ID] = settlement
	cmm.settlements[settlement.SettlementRef] = settlement
	cmm.mu.Unlock()

	return settlement, nil
}

// getMarketInfo 获取市场信息
func (cmm *CrossMarketManager) getMarketInfo(buyMarketID, sellMarketID string) (*MarketInfo, *MarketInfo, error) {
	cmm.mu.RLock()
	defer cmm.mu.RUnlock()

	buyMarket, exists := cmm.markets[buyMarketID]
	if !exists {
		return nil, nil, fmt.Errorf("buy market not found: %s", buyMarketID)
	}

	sellMarket, exists := cmm.markets[sellMarketID]
	if !exists {
		return nil, nil, fmt.Errorf("sell market not found: %s", sellMarketID)
	}

	return buyMarket, sellMarket, nil
}

// isCrossMarket 检查是否为跨市场交易
func (cmm *CrossMarketManager) isCrossMarket(buyMarket, sellMarket *MarketInfo) bool {
	// 如果市场不同，或者货币不同，就是跨市场交易
	return buyMarket.MarketID != sellMarket.MarketID || buyMarket.Currency != sellMarket.Currency
}

// calculateSettlementDates 计算结算日期
func (cmm *CrossMarketManager) calculateSettlementDates(tradeDate time.Time, buyMarket, sellMarket *MarketInfo) (time.Time, time.Time, error) {
	// 计算买方市场结算日期
	buySettlementDate, err := cmm.calculateMarketSettlementDate(tradeDate, buyMarket)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to calculate buy market settlement date: %w", err)
	}

	// 计算卖方市场结算日期
	sellSettlementDate, err := cmm.calculateMarketSettlementDate(tradeDate, sellMarket)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to calculate sell market settlement date: %w", err)
	}

	return buySettlementDate, sellSettlementDate, nil
}

// calculateMarketSettlementDate 计算市场结算日期
func (cmm *CrossMarketManager) calculateMarketSettlementDate(tradeDate time.Time, market *MarketInfo) (time.Time, error) {
	// 根据市场规则计算结算日期
	cycle := market.SettlementRules.DefaultCycle

	switch cycle {
	case CycleT0:
		return tradeDate, nil
	case CycleT1:
		return cmm.addBusinessDays(tradeDate, 1, market), nil
	case CycleT2:
		return cmm.addBusinessDays(tradeDate, 2, market), nil
	case CycleT3:
		return cmm.addBusinessDays(tradeDate, 3, market), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported settlement cycle: %s", cycle)
	}
}

// addBusinessDays 添加工作日
func (cmm *CrossMarketManager) addBusinessDays(date time.Time, days int, market *MarketInfo) time.Time {
	// 简化实现，实际应该考虑节假日和工作日
	return date.AddDate(0, 0, days)
}

// getExchangeRate 获取汇率
func (cmm *CrossMarketManager) getExchangeRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (float64, error) {
	if fromCurrency == toCurrency {
		return 1.0, nil
	}

	rate, err := cmm.exchangeService.GetExchangeRate(ctx, fromCurrency, toCurrency, date)
	if err != nil {
		return 0, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	return rate, nil
}

// createSettlementInstructions 创建结算指令
func (cmm *CrossMarketManager) createSettlementInstructions(ctx context.Context, settlement *CrossMarketSettlement, trade *Trade) error {
	// 创建买方结算指令
	buyInstruction := &SettlementInstruction{
		ID:             generateInstructionID(),
		InstructionRef: fmt.Sprintf("%s_BUY", settlement.SettlementRef),
		TradeID:        trade.TradeID,
		OrderID:        trade.OrderID,
		Symbol:         trade.Symbol,
		SettlementType: SettlementTrade,
		SettlementDate: settlement.BuySettlementDate,
		ValueDate:      settlement.BuySettlementDate,
		Status:         SettlementPending,

		BuyerID:  trade.BuyerID,
		SellerID: trade.SellerID,
		Quantity: trade.Quantity,
		Price:    trade.Price,
		Amount:   settlement.ConvertedAmount, // 使用转换后的金额
		Currency: settlement.BuyCurrency,

		SettlementAmount: settlement.ConvertedAmount,
		NetAmount:        settlement.ConvertedAmount,

		BuyerAccount:  trade.BuyerAccount,
		SellerAccount: trade.SellerAccount,
		Custodian:     trade.Custodian,

		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 创建卖方结算指令
	sellInstruction := &SettlementInstruction{
		ID:             generateInstructionID(),
		InstructionRef: fmt.Sprintf("%s_SELL", settlement.SettlementRef),
		TradeID:        trade.TradeID,
		OrderID:        trade.OrderID,
		Symbol:         trade.Symbol,
		SettlementType: SettlementTrade,
		SettlementDate: settlement.SellSettlementDate,
		ValueDate:      settlement.SellSettlementDate,
		Status:         SettlementPending,

		BuyerID:  trade.BuyerID,
		SellerID: trade.SellerID,
		Quantity: trade.Quantity,
		Price:    trade.Price,
		Amount:   settlement.Amount,
		Currency: settlement.SellCurrency,

		SettlementAmount: settlement.Amount,
		NetAmount:        settlement.Amount,

		BuyerAccount:  trade.BuyerAccount,
		SellerAccount: trade.SellerAccount,
		Custodian:     trade.Custodian,

		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 保存结算指令
	err := cmm.instructionRepo.SaveInstruction(ctx, buyInstruction)
	if err != nil {
		return fmt.Errorf("failed to save buy instruction: %w", err)
	}

	err = cmm.instructionRepo.SaveInstruction(ctx, sellInstruction)
	if err != nil {
		return fmt.Errorf("failed to save sell instruction: %w", err)
	}

	return nil
}

// ProcessCrossMarketSettlement 处理跨市场结算
func (cmm *CrossMarketManager) ProcessCrossMarketSettlement(ctx context.Context, settlementID string) error {
	// 获取跨市场结算
	settlement, err := cmm.getSettlement(ctx, settlementID)
	if err != nil {
		return fmt.Errorf("failed to get settlement: %w", err)
	}

	// 检查状态
	if settlement.Status != SettlementPending {
		return fmt.Errorf("settlement is not in pending state: %s", settlement.Status)
	}

	// 处理买方结算
	err = cmm.processBuySideSettlement(ctx, settlement)
	if err != nil {
		return fmt.Errorf("failed to process buy side settlement: %w", err)
	}

	// 处理卖方结算
	err = cmm.processSellSideSettlement(ctx, settlement)
	if err != nil {
		return fmt.Errorf("failed to process sell side settlement: %w", err)
	}

	// 更新状态
	settlement.Status = SettlementSettled
	settlement.UpdatedAt = time.Now()
	cmm.mu.Lock()
	cmm.settlements[settlement.ID] = settlement
	cmm.settlements[settlement.SettlementRef] = settlement
	cmm.mu.Unlock()

	return nil
}

// processBuySideSettlement 处理买方结算
func (cmm *CrossMarketManager) processBuySideSettlement(ctx context.Context, settlement *CrossMarketSettlement) error {
	// 查找买方结算指令
	instruction, err := cmm.findInstructionByRef(ctx, fmt.Sprintf("%s_BUY", settlement.SettlementRef))
	if err != nil {
		return fmt.Errorf("failed to find buy instruction: %w", err)
	}

	// 处理结算
	err = cmm.settlementManager.processSettlementInstruction(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to process buy instruction: %w", err)
	}

	return nil
}

// processSellSideSettlement 处理卖方结算
func (cmm *CrossMarketManager) processSellSideSettlement(ctx context.Context, settlement *CrossMarketSettlement) error {
	// 查找卖方结算指令
	instruction, err := cmm.findInstructionByRef(ctx, fmt.Sprintf("%s_SELL", settlement.SettlementRef))
	if err != nil {
		return fmt.Errorf("failed to find sell instruction: %w", err)
	}

	// 处理结算
	err = cmm.settlementManager.processSettlementInstruction(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to process sell instruction: %w", err)
	}

	return nil
}

// findInstructionByRef 通过引用查找指令
func (cmm *CrossMarketManager) findInstructionByRef(ctx context.Context, ref string) (*SettlementInstruction, error) {
	instruction, err := cmm.instructionRepo.GetInstructionByRef(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get instruction by ref %s: %w", ref, err)
	}
	if instruction == nil {
		return nil, fmt.Errorf("instruction not found by ref: %s", ref)
	}
	return instruction, nil
}

// getSettlement 获取结算
func (cmm *CrossMarketManager) getSettlement(ctx context.Context, settlementID string) (*CrossMarketSettlement, error) {
	cmm.mu.RLock()
	defer cmm.mu.RUnlock()

	settlement, exists := cmm.settlements[settlementID]
	if !exists || settlement == nil {
		return nil, fmt.Errorf("settlement not found: %s", settlementID)
	}
	return settlement, nil
}

// SettlementReport 结算报告
type SettlementReport struct {
	ID          string    `json:"id"`
	ReportNo    string    `json:"report_no"`
	ReportType  string    `json:"report_type"` // DAILY, WEEKLY, MONTHLY
	ReportDate  time.Time `json:"report_date"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	GeneratedAt time.Time `json:"generated_at"`

	// 报告内容
	Summary            *ReportSummary       `json:"summary"`
	MarketBreakdown    []*MarketStats       `json:"market_breakdown"`
	CurrencyBreakdown  []*CurrencyStats     `json:"currency_breakdown"`
	FailedSettlements  []*FailedSettlement  `json:"failed_settlements"`
	PendingSettlements []*PendingSettlement `json:"pending_settlements"`

	// 格式和分发
	Format         string     `json:"format"` // PDF, HTML, CSV
	Recipients     []string   `json:"recipients"`
	DeliveryStatus string     `json:"delivery_status"`
	DeliveryTime   *time.Time `json:"delivery_time"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReportSummary 报告摘要
type ReportSummary struct {
	TotalSettlements  int           `json:"total_settlements"`
	Successful        int           `json:"successful"`
	Failed            int           `json:"failed"`
	Pending           int           `json:"pending"`
	TotalAmount       float64       `json:"total_amount"`
	AvgSettlementTime time.Duration `json:"avg_settlement_time"`
	SuccessRate       float64       `json:"success_rate"`
}

// MarketStats 市场统计
type MarketStats struct {
	MarketID        string  `json:"market_id"`
	MarketName      string  `json:"market_name"`
	SettlementCount int     `json:"settlement_count"`
	TotalAmount     float64 `json:"total_amount"`
	SuccessCount    int     `json:"success_count"`
	FailureCount    int     `json:"failure_count"`
	SuccessRate     float64 `json:"success_rate"`
}

// CurrencyStats 货币统计
type CurrencyStats struct {
	Currency        string  `json:"currency"`
	SettlementCount int     `json:"settlement_count"`
	TotalAmount     float64 `json:"total_amount"`
	AvgExchangeRate float64 `json:"avg_exchange_rate"`
}

// FailedSettlement 失败结算
type FailedSettlement struct {
	SettlementID  string    `json:"settlement_id"`
	TradeID       string    `json:"trade_id"`
	Symbol        string    `json:"symbol"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	FailureReason string    `json:"failure_reason"`
	FailedAt      time.Time `json:"failed_at"`
	RetryCount    int       `json:"retry_count"`
}

// PendingSettlement 待处理结算
type PendingSettlement struct {
	SettlementID   string    `json:"settlement_id"`
	TradeID        string    `json:"trade_id"`
	Symbol         string    `json:"symbol"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	SettlementDate time.Time `json:"settlement_date"`
	DaysPending    int       `json:"days_pending"`
}

// ReportGenerator 报告生成器
type ReportGenerator struct {
	instructionRepo SettlementInstructionRepository
	marketRepo      MarketRepository
	mu              sync.RWMutex
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator(instructionRepo SettlementInstructionRepository,
	marketRepo MarketRepository) *ReportGenerator {

	return &ReportGenerator{
		instructionRepo: instructionRepo,
		marketRepo:      marketRepo,
	}
}

// GenerateDailyReport 生成日报
func (rg *ReportGenerator) GenerateDailyReport(ctx context.Context, reportDate time.Time) (*SettlementReport, error) {
	periodStart := time.Date(reportDate.Year(), reportDate.Month(), reportDate.Day(), 0, 0, 0, 0, reportDate.Location())
	periodEnd := periodStart.Add(24 * time.Hour)

	return rg.generateReport(ctx, "DAILY", periodStart, periodEnd)
}

// GenerateWeeklyReport 生成周报
func (rg *ReportGenerator) GenerateWeeklyReport(ctx context.Context, weekStart time.Time) (*SettlementReport, error) {
	periodStart := weekStart
	periodEnd := weekStart.Add(7 * 24 * time.Hour)

	return rg.generateReport(ctx, "WEEKLY", periodStart, periodEnd)
}

// GenerateMonthlyReport 生成月报
func (rg *ReportGenerator) GenerateMonthlyReport(ctx context.Context, monthStart time.Time) (*SettlementReport, error) {
	periodStart := monthStart
	// 计算月末
	nextMonth := monthStart.AddDate(0, 1, 0)
	periodEnd := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, nextMonth.Location())

	return rg.generateReport(ctx, "MONTHLY", periodStart, periodEnd)
}

// generateReport 生成报告
func (rg *ReportGenerator) generateReport(ctx context.Context, reportType string, periodStart, periodEnd time.Time) (*SettlementReport, error) {
	// 收集结算数据
	settlements, err := rg.collectSettlementData(ctx, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to collect settlement data: %w", err)
	}

	// 生成报告
	report := &SettlementReport{
		ID:             generateReportID(),
		ReportNo:       generateReportNo(),
		ReportType:     reportType,
		ReportDate:     time.Now(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		GeneratedAt:    time.Now(),
		Format:         "PDF",
		DeliveryStatus: "PENDING",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 生成摘要
	report.Summary = rg.generateSummary(settlements)

	// 生成市场细分
	report.MarketBreakdown = rg.generateMarketBreakdown(settlements)

	// 生成货币细分
	report.CurrencyBreakdown = rg.generateCurrencyBreakdown(settlements)

	// 生成失败结算列表
	report.FailedSettlements = rg.generateFailedSettlements(settlements)

	// 生成待处理结算列表
	report.PendingSettlements = rg.generatePendingSettlements(settlements)

	return report, nil
}

// collectSettlementData 收集结算数据
func (rg *ReportGenerator) collectSettlementData(ctx context.Context, periodStart, periodEnd time.Time) ([]*SettlementInstruction, error) {
	if periodEnd.Before(periodStart) {
		return nil, fmt.Errorf("invalid period: end before start")
	}

	seen := make(map[string]struct{})
	var results []*SettlementInstruction

	day := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, periodStart.Location())
	end := time.Date(periodEnd.Year(), periodEnd.Month(), periodEnd.Day(), 0, 0, 0, 0, periodEnd.Location())
	for !day.After(end) {
		instructions, err := rg.instructionRepo.GetPendingInstructionsByDate(ctx, day)
		if err != nil {
			return nil, fmt.Errorf("failed to query settlement data for %s: %w", day.Format("2006-01-02"), err)
		}
		for _, instruction := range instructions {
			if instruction == nil || instruction.ID == "" {
				continue
			}
			if _, ok := seen[instruction.ID]; ok {
				continue
			}
			seen[instruction.ID] = struct{}{}
			results = append(results, instruction)
		}
		day = day.AddDate(0, 0, 1)
	}

	return results, nil
}

// generateSummary 生成摘要
func (rg *ReportGenerator) generateSummary(settlements []*SettlementInstruction) *ReportSummary {
	summary := &ReportSummary{}

	for _, settlement := range settlements {
		summary.TotalSettlements++
		summary.TotalAmount += settlement.Amount

		switch settlement.Status {
		case SettlementSettled:
			summary.Successful++
		case SettlementFailed:
			summary.Failed++
		case SettlementPending:
			summary.Pending++
		}
	}

	if summary.TotalSettlements > 0 {
		summary.SuccessRate = float64(summary.Successful) / float64(summary.TotalSettlements) * 100
	}

	return summary
}

// generateMarketBreakdown 生成市场细分
func (rg *ReportGenerator) generateMarketBreakdown(settlements []*SettlementInstruction) []*MarketStats {
	marketStats := make(map[string]*MarketStats)

	for _, settlement := range settlements {
		marketID := resolveMarketID(settlement)

		if _, exists := marketStats[marketID]; !exists {
			marketStats[marketID] = &MarketStats{
				MarketID:   marketID,
				MarketName: marketID,
			}
		}

		stats := marketStats[marketID]
		stats.SettlementCount++
		stats.TotalAmount += settlement.Amount

		switch settlement.Status {
		case SettlementSettled:
			stats.SuccessCount++
		case SettlementFailed:
			stats.FailureCount++
		}
	}

	// 计算成功率
	var result []*MarketStats
	for _, stats := range marketStats {
		if stats.SettlementCount > 0 {
			stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.SettlementCount) * 100
		}
		result = append(result, stats)
	}

	return result
}

// generateCurrencyBreakdown 生成货币细分
func (rg *ReportGenerator) generateCurrencyBreakdown(settlements []*SettlementInstruction) []*CurrencyStats {
	currencyStats := make(map[string]*CurrencyStats)

	for _, settlement := range settlements {
		currency := settlement.Currency

		if _, exists := currencyStats[currency]; !exists {
			currencyStats[currency] = &CurrencyStats{
				Currency: currency,
			}
		}

		stats := currencyStats[currency]
		stats.SettlementCount++
		stats.TotalAmount += settlement.Amount
	}

	var result []*CurrencyStats
	for _, stats := range currencyStats {
		result = append(result, stats)
	}

	return result
}

// generateFailedSettlements 生成失败结算列表
func (rg *ReportGenerator) generateFailedSettlements(settlements []*SettlementInstruction) []*FailedSettlement {
	var failed []*FailedSettlement

	for _, settlement := range settlements {
		if settlement.Status == SettlementFailed {
			failedSettlement := &FailedSettlement{
				SettlementID:  settlement.ID,
				TradeID:       settlement.TradeID,
				Symbol:        settlement.Symbol,
				Amount:        settlement.Amount,
				Currency:      settlement.Currency,
				FailureReason: settlement.FailureReason,
				FailedAt:      resolveFailureTime(settlement),
				RetryCount:    settlement.RetryCount,
			}
			failed = append(failed, failedSettlement)
		}
	}

	return failed
}

// generatePendingSettlements 生成待处理结算列表
func (rg *ReportGenerator) generatePendingSettlements(settlements []*SettlementInstruction) []*PendingSettlement {
	var pending []*PendingSettlement

	for _, settlement := range settlements {
		if settlement.Status == SettlementPending {
			daysPending := int(time.Since(settlement.CreatedAt).Hours() / 24)

			pendingSettlement := &PendingSettlement{
				SettlementID:   settlement.ID,
				TradeID:        settlement.TradeID,
				Symbol:         settlement.Symbol,
				Amount:         settlement.Amount,
				Currency:       settlement.Currency,
				SettlementDate: settlement.SettlementDate,
				DaysPending:    daysPending,
			}
			pending = append(pending, pendingSettlement)
		}
	}

	return pending
}

// Helper functions

func generateCrossMarketID() string {
	return fmt.Sprintf("CROSS_MARKET_%d", time.Now().UnixNano())
}

func generateCrossMarketRef() string {
	return fmt.Sprintf("CMREF%d", time.Now().UnixNano())
}

func generateReportID() string {
	return fmt.Sprintf("REPORT_%d", time.Now().UnixNano())
}

func generateReportNo() string {
	return fmt.Sprintf("RPT%d", time.Now().UnixNano())
}

func resolveMarketID(settlement *SettlementInstruction) string {
	if settlement == nil {
		return "UNKNOWN"
	}

	if settlement.Metadata != "" {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(settlement.Metadata), &payload); err == nil {
			if v, ok := payload["market_id"].(string); ok && v != "" {
				return strings.ToUpper(v)
			}
			if v, ok := payload["market"].(string); ok && v != "" {
				return strings.ToUpper(v)
			}
		}
	}

	symbol := strings.TrimSpace(settlement.Symbol)
	if symbol == "" {
		return "UNKNOWN"
	}
	if idx := strings.Index(symbol, "."); idx > 0 {
		return strings.ToUpper(symbol[:idx])
	}
	return "UNKNOWN"
}

func resolveFailureTime(settlement *SettlementInstruction) time.Time {
	if settlement == nil {
		return time.Now()
	}
	if !settlement.UpdatedAt.IsZero() {
		return settlement.UpdatedAt
	}
	return time.Now()
}

// Repository interfaces

type MarketRepository interface {
	GetMarket(ctx context.Context, marketID string) (*MarketInfo, error)
	GetActiveMarkets(ctx context.Context) ([]*MarketInfo, error)
	SaveMarket(ctx context.Context, market *MarketInfo) error
	UpdateMarket(ctx context.Context, market *MarketInfo) error
	DeleteMarket(ctx context.Context, marketID string) error
}

type ExchangeRateService interface {
	GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (float64, error)
	GetHistoricalRates(ctx context.Context, fromCurrency, toCurrency string, startDate, endDate time.Time) ([]*ExchangeRate, error)
	ConvertAmount(ctx context.Context, amount float64, fromCurrency, toCurrency string, date time.Time) (float64, error)
}

type ExchangeRate struct {
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	Rate         float64   `json:"rate"`
	Date         time.Time `json:"date"`
	Source       string    `json:"source"`
}
