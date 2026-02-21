//go:build settlement_experimental
// +build settlement_experimental

package domain

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SettlementType 结算类型
type SettlementType string

const (
	SettlementTrade     SettlementType = "TRADE"     // 交易结算
	SettlementFee       SettlementType = "FEE"       // 费用结算
	SettlementTax       SettlementType = "TAX"       // 税费结算
	SettlementDividend  SettlementType = "DIVIDEND"  // 股息结算
	SettlementInterest  SettlementType = "INTEREST"  // 利息结算
	SettlementCorporate SettlementType = "CORPORATE" // 公司行动结算
)

// SettlementStatus 结算状态
type SettlementStatus string

const (
	SettlementPending   SettlementStatus = "PENDING"   // 待结算
	SettlementMatched   SettlementStatus = "MATCHED"   // 已匹配
	SettlementConfirmed SettlementStatus = "CONFIRMED" // 已确认
	SettlementSettled   SettlementStatus = "SETTLED"   // 已结算
	SettlementFailed    SettlementStatus = "FAILED"    // 结算失败
	SettlementCancelled SettlementStatus = "CANCELLED" // 已取消
)

// SettlementCycle 结算周期
type SettlementCycle string

const (
	CycleT0 SettlementCycle = "T0" // 当日结算
	CycleT1 SettlementCycle = "T1" // T+1结算
	CycleT2 SettlementCycle = "T2" // T+2结算
	CycleT3 SettlementCycle = "T3" // T+3结算
)

// SettlementInstruction 结算指令
type SettlementInstruction struct {
	ID             string           `json:"id"`
	InstructionRef string           `json:"instruction_ref"`
	TradeID        string           `json:"trade_id"`
	OrderID        string           `json:"order_id"`
	Symbol         string           `json:"symbol"`
	SettlementType SettlementType   `json:"settlement_type"`
	SettlementDate time.Time        `json:"settlement_date"`
	ValueDate      time.Time        `json:"value_date"`
	BatchID        string           `json:"batch_id"`
	Status         SettlementStatus `json:"status"`

	// 交易详情
	BuyerID  string  `json:"buyer_id"`
	SellerID string  `json:"seller_id"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`

	// 结算详情
	SettlementAmount float64      `json:"settlement_amount"`
	Fees             []*FeeDetail `json:"fees"`
	Taxes            []*TaxDetail `json:"taxes"`
	NetAmount        float64      `json:"net_amount"`

	// 账户信息
	BuyerAccount  string `json:"buyer_account"`
	SellerAccount string `json:"seller_account"`
	Custodian     string `json:"custodian"`

	// 匹配信息
	MatchStatus    string     `json:"match_status"`
	MatchReference string     `json:"match_reference"`
	MatchDate      *time.Time `json:"match_date"`

	// 确认信息
	ConfirmStatus    string     `json:"confirm_status"`
	ConfirmReference string     `json:"confirm_reference"`
	ConfirmDate      *time.Time `json:"confirm_date"`

	// 结算信息
	SettlementMethod string     `json:"settlement_method"`
	SettlementRef    string     `json:"settlement_ref"`
	SettledAt        *time.Time `json:"settled_at"`

	// 失败信息
	FailureReason string `json:"failure_reason"`
	RetryCount    int    `json:"retry_count"`
	MaxRetries    int    `json:"max_retries"`

	// 元数据
	Metadata  string    `json:"metadata"` // JSON格式
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FeeDetail 费用明细
type FeeDetail struct {
	FeeType     string  `json:"fee_type"`
	FeeAmount   float64 `json:"fee_amount"`
	FeeRate     float64 `json:"fee_rate"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

// TaxDetail 税费明细
type TaxDetail struct {
	TaxType     string  `json:"tax_type"`
	TaxAmount   float64 `json:"tax_amount"`
	TaxRate     float64 `json:"tax_rate"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

// SettlementBatch 结算批次
type SettlementBatch struct {
	ID                string          `json:"id"`
	BatchNo           string          `json:"batch_no"`
	SettlementDate    time.Time       `json:"settlement_date"`
	Cycle             SettlementCycle `json:"cycle"`
	Status            string          `json:"status"`
	TotalInstructions int             `json:"total_instructions"`
	TotalAmount       float64         `json:"total_amount"`
	SuccessCount      int             `json:"success_count"`
	FailedCount       int             `json:"failed_count"`
	PendingCount      int             `json:"pending_count"`
	StartTime         *time.Time      `json:"start_time"`
	EndTime           *time.Time      `json:"end_time"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// SettlementRule 结算规则
type SettlementRule struct {
	ID              string          `json:"id"`
	RuleName        string          `json:"rule_name"`
	Symbol          string          `json:"symbol"`
	Market          string          `json:"market"`
	SettlementCycle SettlementCycle `json:"settlement_cycle"`
	CutOffTime      string          `json:"cut_off_time"` // 截止时间
	ValueDateRule   string          `json:"value_date_rule"`
	HolidayCalendar string          `json:"holiday_calendar"`
	Enabled         bool            `json:"enabled"`
	Priority        int             `json:"priority"`
	Conditions      string          `json:"conditions"` // JSON格式
	Description     string          `json:"description"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// SettlementManager 结算管理器
type SettlementManager struct {
	instructionRepo SettlementInstructionRepository
	batchRepo       SettlementBatchRepository
	ruleRepo        SettlementRuleRepository
	accountRepo     AccountRepository
	paymentGateway  PaymentGateway
	mu              sync.RWMutex
	rules           map[string]*SettlementRule
	batches         map[string]*SettlementBatch
}

// NewSettlementManager 创建结算管理器
func NewSettlementManager(
	instructionRepo SettlementInstructionRepository,
	batchRepo SettlementBatchRepository,
	ruleRepo SettlementRuleRepository,
	accountRepo AccountRepository,
	paymentGateway PaymentGateway,
) *SettlementManager {
	return &SettlementManager{
		instructionRepo: instructionRepo,
		batchRepo:       batchRepo,
		ruleRepo:        ruleRepo,
		accountRepo:     accountRepo,
		paymentGateway:  paymentGateway,
		rules:           make(map[string]*SettlementRule),
		batches:         make(map[string]*SettlementBatch),
	}
}

// Initialize 初始化结算管理器
func (sm *SettlementManager) Initialize(ctx context.Context) error {
	// 加载结算规则
	rules, err := sm.ruleRepo.GetEnabledRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load settlement rules: %w", err)
	}

	sm.mu.Lock()
	for _, rule := range rules {
		sm.rules[rule.ID] = rule
	}
	sm.mu.Unlock()

	return nil
}

// CreateSettlementInstruction 创建结算指令
func (sm *SettlementManager) CreateSettlementInstruction(ctx context.Context, trade *Trade) (*SettlementInstruction, error) {
	// 查找适用的结算规则
	rule, err := sm.findApplicableRule(trade.Symbol, trade.Market)
	if err != nil {
		return nil, fmt.Errorf("failed to find applicable rule: %w", err)
	}

	if rule == nil {
		return nil, fmt.Errorf("no settlement rule found for symbol: %s, market: %s",
			trade.Symbol, trade.Market)
	}

	// 计算结算日期
	settlementDate, valueDate, err := sm.calculateSettlementDates(trade.TradeDate, rule)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate settlement dates: %w", err)
	}

	// 计算金额和费用
	amount := trade.Quantity * trade.Price
	fees := sm.calculateFees(trade, rule)
	taxes := sm.calculateTaxes(trade, rule)

	// 计算净额
	netAmount := amount
	for _, fee := range fees {
		netAmount -= fee.FeeAmount
	}
	for _, tax := range taxes {
		netAmount -= tax.TaxAmount
	}

	// 创建结算指令
	instruction := &SettlementInstruction{
		ID:             generateInstructionID(),
		InstructionRef: generateInstructionRef(),
		TradeID:        trade.TradeID,
		OrderID:        trade.OrderID,
		Symbol:         trade.Symbol,
		SettlementType: SettlementTrade,
		SettlementDate: settlementDate,
		ValueDate:      valueDate,
		Status:         SettlementPending,

		BuyerID:  trade.BuyerID,
		SellerID: trade.SellerID,
		Quantity: trade.Quantity,
		Price:    trade.Price,
		Amount:   amount,
		Currency: trade.Currency,

		SettlementAmount: amount,
		Fees:             fees,
		Taxes:            taxes,
		NetAmount:        netAmount,

		BuyerAccount:  trade.BuyerAccount,
		SellerAccount: trade.SellerAccount,
		Custodian:     trade.Custodian,

		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 保存结算指令
	err = sm.instructionRepo.SaveInstruction(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to save settlement instruction: %w", err)
	}

	return instruction, nil
}

// ProcessSettlementBatch 处理结算批次
func (sm *SettlementManager) ProcessSettlementBatch(ctx context.Context, batchID string) error {
	// 获取批次
	batch, err := sm.batchRepo.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get settlement batch: %w", err)
	}

	// 检查批次状态
	if batch.Status != "PENDING" && batch.Status != "PROCESSING" {
		return fmt.Errorf("batch is not in processable state: %s", batch.Status)
	}

	// 更新批次状态
	batch.Status = "PROCESSING"
	startTime := time.Now()
	batch.StartTime = &startTime
	batch.UpdatedAt = time.Now()

	err = sm.batchRepo.UpdateBatch(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	// 获取批次中的结算指令
	instructions, err := sm.instructionRepo.GetInstructionsByBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch instructions: %w", err)
	}

	// 处理每个结算指令
	for _, instruction := range instructions {
		err := sm.processSettlementInstruction(ctx, instruction)
		if err != nil {
			// 记录失败但继续处理其他指令
			instruction.Status = SettlementFailed
			instruction.FailureReason = err.Error()
			instruction.UpdatedAt = time.Now()

			sm.instructionRepo.UpdateInstruction(ctx, instruction)
			batch.FailedCount++
		} else {
			batch.SuccessCount++
		}

		batch.UpdatedAt = time.Now()
		sm.batchRepo.UpdateBatch(ctx, batch)
	}

	// 更新批次完成状态
	batch.Status = "COMPLETED"
	endTime := time.Now()
	batch.EndTime = &endTime
	batch.UpdatedAt = time.Now()

	err = sm.batchRepo.UpdateBatch(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to update batch completion: %w", err)
	}

	return nil
}

// processSettlementInstruction 处理结算指令
func (sm *SettlementManager) processSettlementInstruction(ctx context.Context, instruction *SettlementInstruction) error {
	// 检查指令状态
	if instruction.Status != SettlementPending && instruction.Status != SettlementMatched {
		return fmt.Errorf("instruction is not in processable state: %s", instruction.Status)
	}

	// 执行资金结算
	err := sm.executeFundSettlement(ctx, instruction)
	if err != nil {
		// 检查是否需要重试
		if instruction.RetryCount < instruction.MaxRetries {
			instruction.RetryCount++
			instruction.UpdatedAt = time.Now()
			sm.instructionRepo.UpdateInstruction(ctx, instruction)

			// 延迟重试
			time.Sleep(time.Duration(instruction.RetryCount) * time.Minute)
			return sm.processSettlementInstruction(ctx, instruction)
		}
		return fmt.Errorf("fund settlement failed after %d retries: %w", instruction.MaxRetries, err)
	}

	// 执行证券结算
	err = sm.executeSecuritySettlement(ctx, instruction)
	if err != nil {
		return fmt.Errorf("security settlement failed: %w", err)
	}

	// 更新指令状态
	instruction.Status = SettlementSettled
	settlementDate := time.Now()
	instruction.SettledAt = &settlementDate
	instruction.UpdatedAt = time.Now()

	err = sm.instructionRepo.UpdateInstruction(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to update instruction status: %w", err)
	}

	return nil
}

// executeFundSettlement 执行资金结算
func (sm *SettlementManager) executeFundSettlement(ctx context.Context, instruction *SettlementInstruction) error {
	// 检查买方账户余额
	buyerBalance, err := sm.accountRepo.GetAccountBalance(ctx, instruction.BuyerAccount, instruction.Currency)
	if err != nil {
		return fmt.Errorf("failed to get buyer account balance: %w", err)
	}

	if buyerBalance.Available < instruction.NetAmount {
		return fmt.Errorf("insufficient buyer balance: available=%.2f, required=%.2f",
			buyerBalance.Available, instruction.NetAmount)
	}

	// 执行资金转账
	transferReq := &FundTransferRequest{
		FromAccount: instruction.BuyerAccount,
		ToAccount:   instruction.SellerAccount,
		Amount:      instruction.NetAmount,
		Currency:    instruction.Currency,
		Reference:   instruction.InstructionRef,
		Description: fmt.Sprintf("Trade settlement for %s", instruction.TradeID),
	}

	transferResp, err := sm.paymentGateway.TransferFunds(ctx, transferReq)
	if err != nil {
		return fmt.Errorf("fund transfer failed: %w", err)
	}

	// 记录资金转账
	fundTransfer := &FundTransfer{
		TransferID:    transferResp.TransferID,
		InstructionID: instruction.ID,
		FromAccount:   instruction.BuyerAccount,
		ToAccount:     instruction.SellerAccount,
		Amount:        instruction.NetAmount,
		Currency:      instruction.Currency,
		Status:        "COMPLETED",
		Reference:     instruction.InstructionRef,
		ExecutedAt:    time.Now(),
	}

	err = sm.accountRepo.RecordFundTransfer(ctx, fundTransfer)
	if err != nil {
		return fmt.Errorf("failed to record fund transfer: %w", err)
	}

	return nil
}

// executeSecuritySettlement 执行证券结算
func (sm *SettlementManager) executeSecuritySettlement(ctx context.Context, instruction *SettlementInstruction) error {
	// 检查卖方证券持仓
	sellerPosition, err := sm.accountRepo.GetSecurityPosition(ctx, instruction.SellerAccount, instruction.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get seller security position: %w", err)
	}

	if sellerPosition.Available < instruction.Quantity {
		return fmt.Errorf("insufficient seller position: available=%.2f, required=%.2f",
			sellerPosition.Available, instruction.Quantity)
	}

	// 执行证券转账
	secTransferReq := &SecurityTransferRequest{
		FromAccount: instruction.SellerAccount,
		ToAccount:   instruction.BuyerAccount,
		Symbol:      instruction.Symbol,
		Quantity:    instruction.Quantity,
		Reference:   instruction.InstructionRef,
		Description: fmt.Sprintf("Security delivery for trade %s", instruction.TradeID),
	}

	secTransferResp, err := sm.accountRepo.TransferSecurities(ctx, secTransferReq)
	if err != nil {
		return fmt.Errorf("security transfer failed: %w", err)
	}

	// 记录证券转账
	secTransfer := &SecurityTransfer{
		TransferID:    secTransferResp.TransferID,
		InstructionID: instruction.ID,
		FromAccount:   instruction.SellerAccount,
		ToAccount:     instruction.BuyerAccount,
		Symbol:        instruction.Symbol,
		Quantity:      instruction.Quantity,
		Status:        "COMPLETED",
		Reference:     instruction.InstructionRef,
		ExecutedAt:    time.Now(),
	}

	err = sm.accountRepo.RecordSecurityTransfer(ctx, secTransfer)
	if err != nil {
		return fmt.Errorf("failed to record security transfer: %w", err)
	}

	return nil
}

// MatchSettlementInstructions 匹配结算指令
func (sm *SettlementManager) MatchSettlementInstructions(ctx context.Context, instruction1, instruction2 *SettlementInstruction) error {
	// 检查指令是否可匹配
	if err := sm.validateMatching(instruction1, instruction2); err != nil {
		return fmt.Errorf("matching validation failed: %w", err)
	}

	// 更新匹配状态
	instruction1.Status = SettlementMatched
	instruction1.MatchStatus = "MATCHED"
	matchDate := time.Now()
	instruction1.MatchDate = &matchDate
	instruction1.MatchReference = generateMatchReference()
	instruction1.UpdatedAt = time.Now()

	instruction2.Status = SettlementMatched
	instruction2.MatchStatus = "MATCHED"
	instruction2.MatchDate = &matchDate
	instruction2.MatchReference = instruction1.MatchReference
	instruction2.UpdatedAt = time.Now()

	// 保存更新
	err := sm.instructionRepo.UpdateInstruction(ctx, instruction1)
	if err != nil {
		return fmt.Errorf("failed to update instruction 1: %w", err)
	}

	err = sm.instructionRepo.UpdateInstruction(ctx, instruction2)
	if err != nil {
		return fmt.Errorf("failed to update instruction 2: %w", err)
	}

	return nil
}

// validateMatching 验证匹配
func (sm *SettlementManager) validateMatching(instruction1, instruction2 *SettlementInstruction) error {
	// 检查是否为同一交易
	if instruction1.TradeID == instruction2.TradeID {
		return fmt.Errorf("cannot match instructions from same trade")
	}

	// 检查符号是否匹配
	if instruction1.Symbol != instruction2.Symbol {
		return fmt.Errorf("symbol mismatch: %s vs %s", instruction1.Symbol, instruction2.Symbol)
	}

	// 检查数量是否匹配
	if instruction1.Quantity != instruction2.Quantity {
		return fmt.Errorf("quantity mismatch: %.2f vs %.2f", instruction1.Quantity, instruction2.Quantity)
	}

	// 检查价格是否匹配
	if instruction1.Price != instruction2.Price {
		return fmt.Errorf("price mismatch: %.2f vs %.2f", instruction1.Price, instruction2.Price)
	}

	// 检查结算日期是否匹配
	if !instruction1.SettlementDate.Equal(instruction2.SettlementDate) {
		return fmt.Errorf("settlement date mismatch: %s vs %s",
			instruction1.SettlementDate, instruction2.SettlementDate)
	}

	// 检查买卖方向
	if instruction1.BuyerID == instruction2.BuyerID || instruction1.SellerID == instruction2.SellerID {
		return fmt.Errorf("same party on both sides of trade")
	}

	return nil
}

// CreateSettlementBatch 创建结算批次
func (sm *SettlementManager) CreateSettlementBatch(ctx context.Context, settlementDate time.Time, cycle SettlementCycle) (*SettlementBatch, error) {
	// 检查是否已存在批次
	existingBatch, err := sm.batchRepo.GetBatchByDate(ctx, settlementDate, cycle)
	if err == nil && existingBatch != nil {
		return existingBatch, nil
	}

	// 创建新批次
	batch := &SettlementBatch{
		ID:                generateBatchID(),
		BatchNo:           generateBatchNo(),
		SettlementDate:    settlementDate,
		Cycle:             cycle,
		Status:            "PENDING",
		TotalInstructions: 0,
		TotalAmount:       0,
		SuccessCount:      0,
		FailedCount:       0,
		PendingCount:      0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 保存批次
	err = sm.batchRepo.SaveBatch(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("failed to save settlement batch: %w", err)
	}

	// 查找待结算的指令
	instructions, err := sm.instructionRepo.GetPendingInstructionsByDate(ctx, settlementDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending instructions: %w", err)
	}

	// 将指令分配到批次
	for _, instruction := range instructions {
		instruction.BatchID = batch.ID
		instruction.UpdatedAt = time.Now()

		err = sm.instructionRepo.UpdateInstruction(ctx, instruction)
		if err != nil {
			// 记录错误但继续处理其他指令
			fmt.Printf("Failed to assign instruction %s to batch: %v\n", instruction.ID, err)
			continue
		}

		batch.TotalInstructions++
		batch.TotalAmount += instruction.NetAmount
		batch.PendingCount++
	}

	// 更新批次统计
	batch.UpdatedAt = time.Now()
	err = sm.batchRepo.UpdateBatch(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("failed to update batch statistics: %w", err)
	}

	return batch, nil
}

// Helper methods

func (sm *SettlementManager) findApplicableRule(symbol, market string) (*SettlementRule, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 查找最匹配的规则
	var bestMatch *SettlementRule
	var bestScore int

	for _, rule := range sm.rules {
		if !rule.Enabled {
			continue
		}

		score := 0

		// 检查符号匹配
		if rule.Symbol == symbol || rule.Symbol == "*" {
			score += 10
		} else if rule.Symbol != "" {
			continue // 符号不匹配
		}

		// 检查市场匹配
		if rule.Market == market || rule.Market == "*" {
			score += 5
		} else if rule.Market != "" {
			continue // 市场不匹配
		}

		// 检查优先级
		score += rule.Priority

		if bestMatch == nil || score > bestScore {
			bestMatch = rule
			bestScore = score
		}
	}

	return bestMatch, nil
}

func (sm *SettlementManager) calculateSettlementDates(tradeDate time.Time, rule *SettlementRule) (time.Time, time.Time, error) {
	// 计算结算日期
	var settlementDate time.Time

	switch rule.SettlementCycle {
	case CycleT0:
		settlementDate = tradeDate
	case CycleT1:
		settlementDate = sm.addBusinessDays(tradeDate, 1, rule.HolidayCalendar)
	case CycleT2:
		settlementDate = sm.addBusinessDays(tradeDate, 2, rule.HolidayCalendar)
	case CycleT3:
		settlementDate = sm.addBusinessDays(tradeDate, 3, rule.HolidayCalendar)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported settlement cycle: %s", rule.SettlementCycle)
	}

	// 计算价值日期（通常与结算日期相同）
	valueDate := settlementDate

	return settlementDate, valueDate, nil
}

func (sm *SettlementManager) addBusinessDays(date time.Time, days int, holidayCalendar string) time.Time {
	// 简单实现，不考虑节假日
	return date.AddDate(0, 0, days)
}

func (sm *SettlementManager) calculateFees(trade *Trade, rule *SettlementRule) []*FeeDetail {
	var fees []*FeeDetail

	// 计算交易佣金
	commission := &FeeDetail{
		FeeType:     "COMMISSION",
		FeeAmount:   trade.Quantity * trade.Price * 0.001, // 0.1%佣金
		FeeRate:     0.001,
		Currency:    trade.Currency,
		Description: "Trading commission",
	}
	fees = append(fees, commission)

	// 计算结算费用
	settlementFee := &FeeDetail{
		FeeType:     "SETTLEMENT_FEE",
		FeeAmount:   1.0, // 固定费用
		FeeRate:     0,
		Currency:    trade.Currency,
		Description: "Settlement processing fee",
	}
	fees = append(fees, settlementFee)

	return fees
}

func (sm *SettlementManager) calculateTaxes(trade *Trade, rule *SettlementRule) []*TaxDetail {
	var taxes []*TaxDetail

	// 计算印花税（仅对买方）
	stampDuty := &TaxDetail{
		TaxType:     "STAMP_DUTY",
		TaxAmount:   trade.Quantity * trade.Price * 0.001, // 0.1%印花税
		TaxRate:     0.001,
		Currency:    trade.Currency,
		Description: "Stamp duty",
	}
	taxes = append(taxes, stampDuty)

	return taxes
}

// Helper functions

func generateInstructionID() string {
	return fmt.Sprintf("SETTLE_INST_%d", time.Now().UnixNano())
}

func generateInstructionRef() string {
	return fmt.Sprintf("SIREF%d", time.Now().UnixNano())
}

func generateBatchID() string {
	return fmt.Sprintf("BATCH_%d", time.Now().UnixNano())
}

func generateBatchNo() string {
	return fmt.Sprintf("BATCH%d", time.Now().UnixNano())
}

func generateMatchReference() string {
	return fmt.Sprintf("MATCH%d", time.Now().UnixNano())
}

// Data structures

type Trade struct {
	TradeID       string    `json:"trade_id"`
	OrderID       string    `json:"order_id"`
	Symbol        string    `json:"symbol"`
	Market        string    `json:"market"`
	BuyMarket     string    `json:"buy_market"`
	SellMarket    string    `json:"sell_market"`
	BuyerID       string    `json:"buyer_id"`
	SellerID      string    `json:"seller_id"`
	Quantity      float64   `json:"quantity"`
	Price         float64   `json:"price"`
	Currency      string    `json:"currency"`
	TradeDate     time.Time `json:"trade_date"`
	BuyerAccount  string    `json:"buyer_account"`
	SellerAccount string    `json:"seller_account"`
	Custodian     string    `json:"custodian"`
}

type AccountBalance struct {
	AccountID string    `json:"account_id"`
	Currency  string    `json:"currency"`
	Balance   float64   `json:"balance"`
	Available float64   `json:"available"`
	Frozen    float64   `json:"frozen"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SecurityPosition struct {
	AccountID   string    `json:"account_id"`
	Symbol      string    `json:"symbol"`
	Quantity    float64   `json:"quantity"`
	Available   float64   `json:"available"`
	Frozen      float64   `json:"frozen"`
	Cost        float64   `json:"cost"`
	MarketValue float64   `json:"market_value"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FundTransferRequest struct {
	FromAccount string  `json:"from_account"`
	ToAccount   string  `json:"to_account"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Reference   string  `json:"reference"`
	Description string  `json:"description"`
}

type FundTransferResponse struct {
	TransferID string    `json:"transfer_id"`
	Status     string    `json:"status"`
	ExecutedAt time.Time `json:"executed_at"`
	Reference  string    `json:"reference"`
}

type FundTransfer struct {
	TransferID    string    `json:"transfer_id"`
	InstructionID string    `json:"instruction_id"`
	FromAccount   string    `json:"from_account"`
	ToAccount     string    `json:"to_account"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Reference     string    `json:"reference"`
	ExecutedAt    time.Time `json:"executed_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type SecurityTransferRequest struct {
	FromAccount string  `json:"from_account"`
	ToAccount   string  `json:"to_account"`
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	Reference   string  `json:"reference"`
	Description string  `json:"description"`
}

type SecurityTransferResponse struct {
	TransferID string    `json:"transfer_id"`
	Status     string    `json:"status"`
	ExecutedAt time.Time `json:"executed_at"`
	Reference  string    `json:"reference"`
}

type SecurityTransfer struct {
	TransferID    string    `json:"transfer_id"`
	InstructionID string    `json:"instruction_id"`
	FromAccount   string    `json:"from_account"`
	ToAccount     string    `json:"to_account"`
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	Status        string    `json:"status"`
	Reference     string    `json:"reference"`
	ExecutedAt    time.Time `json:"executed_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// Repository interfaces

type SettlementInstructionRepository interface {
	SaveInstruction(ctx context.Context, instruction *SettlementInstruction) error
	GetInstruction(ctx context.Context, id string) (*SettlementInstruction, error)
	GetInstructionByRef(ctx context.Context, ref string) (*SettlementInstruction, error)
	GetInstructionsByTrade(ctx context.Context, tradeID string) ([]*SettlementInstruction, error)
	GetInstructionsByBatch(ctx context.Context, batchID string) ([]*SettlementInstruction, error)
	GetPendingInstructionsByDate(ctx context.Context, settlementDate time.Time) ([]*SettlementInstruction, error)
	UpdateInstruction(ctx context.Context, instruction *SettlementInstruction) error
	DeleteInstruction(ctx context.Context, id string) error
}

type SettlementBatchRepository interface {
	SaveBatch(ctx context.Context, batch *SettlementBatch) error
	GetBatch(ctx context.Context, id string) (*SettlementBatch, error)
	GetBatchByDate(ctx context.Context, settlementDate time.Time, cycle SettlementCycle) (*SettlementBatch, error)
	GetBatchesByStatus(ctx context.Context, status string, startDate, endDate time.Time) ([]*SettlementBatch, error)
	UpdateBatch(ctx context.Context, batch *SettlementBatch) error
	DeleteBatch(ctx context.Context, id string) error
}

type SettlementRuleRepository interface {
	SaveRule(ctx context.Context, rule *SettlementRule) error
	GetRule(ctx context.Context, id string) (*SettlementRule, error)
	GetRulesBySymbol(ctx context.Context, symbol string) ([]*SettlementRule, error)
	GetEnabledRules(ctx context.Context) ([]*SettlementRule, error)
	UpdateRule(ctx context.Context, rule *SettlementRule) error
	DeleteRule(ctx context.Context, id string) error
}

type AccountRepository interface {
	GetAccountBalance(ctx context.Context, accountID, currency string) (*AccountBalance, error)
	GetSecurityPosition(ctx context.Context, accountID, symbol string) (*SecurityPosition, error)
	RecordFundTransfer(ctx context.Context, transfer *FundTransfer) error
	TransferSecurities(ctx context.Context, req *SecurityTransferRequest) (*SecurityTransferResponse, error)
	RecordSecurityTransfer(ctx context.Context, transfer *SecurityTransfer) error
}

type PaymentGateway interface {
	TransferFunds(ctx context.Context, req *FundTransferRequest) (*FundTransferResponse, error)
	GetTransferStatus(ctx context.Context, transferID string) (*FundTransferResponse, error)
	CancelTransfer(ctx context.Context, transferID string) error
}
