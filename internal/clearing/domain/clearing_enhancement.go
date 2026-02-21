//go:build clearing_experimental
// +build clearing_experimental

package domain

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ClearingType 清算类型
type ClearingType string

const (
	ClearingTypeTrade    ClearingType = "TRADE"        // 交易清算
	ClearingNetting      ClearingType = "NETTING"      // 净额清算
	ClearingMultilateral ClearingType = "MULTILATERAL" // 多边清算
	ClearingBilateral    ClearingType = "BILATERAL"    // 双边清算
	ClearingCCP          ClearingType = "CCP"          // 中央对手方清算
)

// ClearingStatus 清算状态
type ClearingStatus string

const (
	ClearingPending   ClearingStatus = "PENDING"   // 待清算
	ClearingValidated ClearingStatus = "VALIDATED" // 已验证
	ClearingMatched   ClearingStatus = "MATCHED"   // 已匹配
	ClearingConfirmed ClearingStatus = "CONFIRMED" // 已确认
	ClearingSettled   ClearingStatus = "SETTLED"   // 已清算
	ClearingFailed    ClearingStatus = "FAILED"    // 清算失败
	ClearingCancelled ClearingStatus = "CANCELLED" // 已取消
)

// ClearingMember 清算会员
type ClearingMember struct {
	ID                string    `json:"id"`
	MemberCode        string    `json:"member_code"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`   // BROKER, DEALER, INSTITUTION
	Status            string    `json:"status"` // ACTIVE, SUSPENDED, TERMINATED
	CreditRating      string    `json:"credit_rating"`
	CreditLimit       float64   `json:"credit_limit"`
	MarginRequirement float64   `json:"margin_requirement"`
	Collateral        float64   `json:"collateral"`
	NetCapital        float64   `json:"net_capital"`
	RiskLevel         string    `json:"risk_level"`
	JoinedDate        time.Time `json:"joined_date"`
	LastReviewDate    time.Time `json:"last_review_date"`
	Metadata          string    `json:"metadata"` // JSON格式
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ClearingInstruction 清算指令
type ClearingInstruction struct {
	ID             string         `json:"id"`
	InstructionRef string         `json:"instruction_ref"`
	TradeID        string         `json:"trade_id"`
	ClearingType   ClearingType   `json:"clearing_type"`
	Status         ClearingStatus `json:"status"`

	// 交易详情
	BuyMemberID    string    `json:"buy_member_id"`
	SellMemberID   string    `json:"sell_member_id"`
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`
	Price          float64   `json:"price"`
	TradeDate      time.Time `json:"trade_date"`
	SettlementDate time.Time `json:"settlement_date"`

	// 清算详情
	ClearingDate time.Time `json:"clearing_date"`
	NetPosition  float64   `json:"net_position"`
	GrossAmount  float64   `json:"gross_amount"`
	NetAmount    float64   `json:"net_amount"`

	// 保证金要求
	InitialMargin     float64 `json:"initial_margin"`
	VariationMargin   float64 `json:"variation_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	TotalMargin       float64 `json:"total_margin"`

	// 风险控制
	RiskScore        float64 `json:"risk_score"`
	RiskLimit        float64 `json:"risk_limit"`
	LimitUtilization float64 `json:"limit_utilization"`

	// 匹配信息
	MatchStatus    string     `json:"match_status"`
	MatchReference string     `json:"match_reference"`
	MatchDate      *time.Time `json:"match_date"`

	// 确认信息
	ConfirmStatus    string     `json:"confirm_status"`
	ConfirmReference string     `json:"confirm_reference"`
	ConfirmDate      *time.Time `json:"confirm_date"`

	// 失败信息
	FailureReason string `json:"failure_reason"`
	RetryCount    int    `json:"retry_count"`
	MaxRetries    int    `json:"max_retries"`

	// 元数据
	Metadata  string    `json:"metadata"` // JSON格式
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ClearingBatch 清算批次
type ClearingBatch struct {
	ID                string       `json:"id"`
	BatchNo           string       `json:"batch_no"`
	ClearingDate      time.Time    `json:"clearing_date"`
	ClearingType      ClearingType `json:"clearing_type"`
	Status            string       `json:"status"`
	TotalInstructions int          `json:"total_instructions"`
	TotalAmount       float64      `json:"total_amount"`
	SuccessCount      int          `json:"success_count"`
	FailedCount       int          `json:"failed_count"`
	PendingCount      int          `json:"pending_count"`
	StartTime         *time.Time   `json:"start_time"`
	EndTime           *time.Time   `json:"end_time"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

// ClearingRule 清算规则
type ClearingRule struct {
	ID              string       `json:"id"`
	RuleName        string       `json:"rule_name"`
	Symbol          string       `json:"symbol"`
	Market          string       `json:"market"`
	ClearingType    ClearingType `json:"clearing_type"`
	MarginModel     string       `json:"margin_model"`
	NettingMethod   string       `json:"netting_method"`
	CutOffTime      string       `json:"cut_off_time"`
	SettlementCycle string       `json:"settlement_cycle"`
	Enabled         bool         `json:"enabled"`
	Priority        int          `json:"priority"`
	Conditions      string       `json:"conditions"` // JSON格式
	Description     string       `json:"description"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// MarginCall 保证金催缴
type MarginCall struct {
	ID                string     `json:"id"`
	CallRef           string     `json:"call_ref"`
	MemberID          string     `json:"member_id"`
	CallType          string     `json:"call_type"` // INITIAL, VARIATION, MAINTENANCE
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	DueDate           time.Time  `json:"due_date"`
	Status            string     `json:"status"` // PENDING, PARTIAL, COMPLETED, OVERDUE
	PaidAmount        float64    `json:"paid_amount"`
	OutstandingAmount float64    `json:"outstanding_amount"`
	CallDate          time.Time  `json:"call_date"`
	ResponseDate      *time.Time `json:"response_date"`
	FailureReason     string     `json:"failure_reason"`
	Metadata          string     `json:"metadata"` // JSON格式
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// NettingResult 净额结果
type NettingResult struct {
	ID             string    `json:"id"`
	NettingID      string    `json:"netting_id"`
	MemberID       string    `json:"member_id"`
	Symbol         string    `json:"symbol"`
	SettlementDate time.Time `json:"settlement_date"`
	GrossBuy       float64   `json:"gross_buy"`
	GrossSell      float64   `json:"gross_sell"`
	NetPosition    float64   `json:"net_position"`
	NetAmount      float64   `json:"net_amount"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ClearingManager 清算管理器
type ClearingManager struct {
	memberRepo      ClearingMemberRepository
	instructionRepo ClearingInstructionRepository
	batchRepo       ClearingBatchRepository
	ruleRepo        ClearingRuleRepository
	marginRepo      MarginRepository
	nettingRepo     NettingRepository
	riskManager     RiskManager
	mu              sync.RWMutex
	members         map[string]*ClearingMember
	rules           map[string]*ClearingRule
	batches         map[string]*ClearingBatch
}

// NewClearingManager 创建清算管理器
func NewClearingManager(
	memberRepo ClearingMemberRepository,
	instructionRepo ClearingInstructionRepository,
	batchRepo ClearingBatchRepository,
	ruleRepo ClearingRuleRepository,
	marginRepo MarginRepository,
	nettingRepo NettingRepository,
	riskManager RiskManager,
) *ClearingManager {
	return &ClearingManager{
		memberRepo:      memberRepo,
		instructionRepo: instructionRepo,
		batchRepo:       batchRepo,
		ruleRepo:        ruleRepo,
		marginRepo:      marginRepo,
		nettingRepo:     nettingRepo,
		riskManager:     riskManager,
		members:         make(map[string]*ClearingMember),
		rules:           make(map[string]*ClearingRule),
		batches:         make(map[string]*ClearingBatch),
	}
}

// Initialize 初始化清算管理器
func (cm *ClearingManager) Initialize(ctx context.Context) error {
	// 加载清算会员
	members, err := cm.memberRepo.GetActiveMembers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load clearing members: %w", err)
	}

	cm.mu.Lock()
	for _, member := range members {
		cm.members[member.ID] = member
	}
	cm.mu.Unlock()

	// 加载清算规则
	rules, err := cm.ruleRepo.GetEnabledRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load clearing rules: %w", err)
	}

	cm.mu.Lock()
	for _, rule := range rules {
		cm.rules[rule.ID] = rule
	}
	cm.mu.Unlock()

	return nil
}

// CreateClearingInstruction 创建清算指令
func (cm *ClearingManager) CreateClearingInstruction(ctx context.Context, trade *Trade) (*ClearingInstruction, error) {
	// 查找适用的清算规则
	rule, err := cm.findApplicableRule(trade.Symbol, trade.Market)
	if err != nil {
		return nil, fmt.Errorf("failed to find applicable rule: %w", err)
	}

	if rule == nil {
		return nil, fmt.Errorf("no clearing rule found for symbol: %s, market: %s",
			trade.Symbol, trade.Market)
	}

	// 验证清算会员
	buyMember, sellMember, err := cm.validateClearingMembers(ctx, trade.BuyMemberID, trade.SellMemberID)
	if err != nil {
		return nil, fmt.Errorf("member validation failed: %w", err)
	}

	// 计算保证金要求
	marginReq, err := cm.calculateMarginRequirements(ctx, trade, buyMember, sellMember, rule)
	if err != nil {
		return nil, fmt.Errorf("margin calculation failed: %w", err)
	}

	// 创建清算指令
	instruction := &ClearingInstruction{
		ID:             generateInstructionID(),
		InstructionRef: generateInstructionRef(),
		TradeID:        trade.TradeID,
		ClearingType:   rule.ClearingType,
		Status:         ClearingPending,

		BuyMemberID:    trade.BuyMemberID,
		SellMemberID:   trade.SellMemberID,
		Symbol:         trade.Symbol,
		Quantity:       trade.Quantity,
		Price:          trade.Price,
		TradeDate:      trade.TradeDate,
		SettlementDate: trade.SettlementDate,

		ClearingDate: time.Now(),
		GrossAmount:  trade.Quantity * trade.Price,
		NetAmount:    trade.Quantity * trade.Price,

		InitialMargin:     marginReq.InitialMargin,
		VariationMargin:   marginReq.VariationMargin,
		MaintenanceMargin: marginReq.MaintenanceMargin,
		TotalMargin:       marginReq.TotalMargin,

		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 保存清算指令
	err = cm.instructionRepo.SaveInstruction(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to save clearing instruction: %w", err)
	}

	// 检查保证金要求
	err = cm.checkMarginRequirements(ctx, instruction, buyMember, sellMember)
	if err != nil {
		return nil, fmt.Errorf("margin requirement check failed: %w", err)
	}

	return instruction, nil
}

// ProcessClearingBatch 处理清算批次
func (cm *ClearingManager) ProcessClearingBatch(ctx context.Context, batchID string) error {
	// 获取批次
	batch, err := cm.batchRepo.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get clearing batch: %w", err)
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

	err = cm.batchRepo.UpdateBatch(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	// 根据清算类型处理
	switch batch.ClearingType {
	case ClearingNetting:
		err = cm.processNettingBatch(ctx, batch)
	case ClearingMultilateral:
		err = cm.processMultilateralBatch(ctx, batch)
	case ClearingBilateral:
		err = cm.processBilateralBatch(ctx, batch)
	case ClearingCCP:
		err = cm.processCCPBatch(ctx, batch)
	default:
		err = cm.processTradeBatch(ctx, batch)
	}

	if err != nil {
		batch.Status = "FAILED"
		batch.UpdatedAt = time.Now()
		cm.batchRepo.UpdateBatch(ctx, batch)
		return fmt.Errorf("batch processing failed: %w", err)
	}

	// 更新批次完成状态
	batch.Status = "COMPLETED"
	endTime := time.Now()
	batch.EndTime = &endTime
	batch.UpdatedAt = time.Now()

	err = cm.batchRepo.UpdateBatch(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to update batch completion: %w", err)
	}

	return nil
}

// processTradeBatch 处理交易清算批次
func (cm *ClearingManager) processTradeBatch(ctx context.Context, batch *ClearingBatch) error {
	// 获取批次中的清算指令
	instructions, err := cm.instructionRepo.GetInstructionsByBatch(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("failed to get batch instructions: %w", err)
	}

	// 并行处理优化：按 AccountID (BuyMemberID) 分片，确保同一账户串行处理
	workerCount := 10 // 可根据 CPU 核心数调整: runtime.NumCPU() * 2
	shards := make([]chan *ClearingInstruction, workerCount)
	for i := 0; i < workerCount; i++ {
		shards[i] = make(chan *ClearingInstruction, len(instructions)/workerCount+1)
	}

	// 分发任务
	for _, instruction := range instructions {
		// 使用 FNV-1a 哈希或其他简单哈希将 MemberID 映射到 shard
		h := fnv32(instruction.BuyMemberID)
		shardIdx := h % uint32(workerCount)
		shards[shardIdx] <- instruction
	}

	// 关闭所有 channel
	for i := 0; i < workerCount; i++ {
		close(shards[i])
	}

	var wg sync.WaitGroup
	var successCount, failedCount int32

	// 启动 Workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(shardIndex int) {
			defer wg.Done()
			for instruction := range shards[shardIndex] {
				err := cm.processClearingInstruction(ctx, instruction)
				if err != nil {
					instruction.Status = ClearingFailed
					instruction.FailureReason = err.Error()
					instruction.UpdatedAt = time.Now()
					// 忽略更新错误，仅打日志或统计
					_ = cm.instructionRepo.UpdateInstruction(ctx, instruction)
					atomic.AddInt32(&failedCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
			}
		}(i)
	}
	wg.Wait()

	batch.SuccessCount = int(successCount)
	batch.FailedCount = int(failedCount)
	batch.UpdatedAt = time.Now()
	cm.batchRepo.UpdateBatch(ctx, batch)

	return nil
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

// processNettingBatch 处理净额清算批次
func (cm *ClearingManager) processNettingBatch(ctx context.Context, batch *ClearingBatch) error {
	// 获取批次中的清算指令
	instructions, err := cm.instructionRepo.GetInstructionsByBatch(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("failed to get batch instructions: %w", err)
	}

	// 按会员和符号分组
	positions := cm.groupPositionsByMemberAndSymbol(instructions)

	// 计算净额
	nettingResults, err := cm.calculateNetting(ctx, positions, batch.ClearingDate)
	if err != nil {
		return fmt.Errorf("netting calculation failed: %w", err)
	}

	// 保存净额结果
	for _, result := range nettingResults {
		err = cm.nettingRepo.SaveNettingResult(ctx, result)
		if err != nil {
			return fmt.Errorf("failed to save netting result: %w", err)
		}
	}

	// 更新指令状态
	for _, instruction := range instructions {
		instruction.Status = ClearingSettled
		instruction.UpdatedAt = time.Now()
		cm.instructionRepo.UpdateInstruction(ctx, instruction)
	}

	return nil
}

// processMultilateralBatch 处理多边清算批次
func (cm *ClearingManager) processMultilateralBatch(ctx context.Context, batch *ClearingBatch) error {
	// 获取批次中的清算指令
	instructions, err := cm.instructionRepo.GetInstructionsByBatch(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("failed to get batch instructions: %w", err)
	}

	// 构建多边清算矩阵
	matrix, err := cm.buildMultilateralMatrix(instructions)
	if err != nil {
		return fmt.Errorf("failed to build multilateral matrix: %w", err)
	}

	// 执行多边清算
	results, err := cm.executeMultilateralClearing(ctx, matrix)
	if err != nil {
		return fmt.Errorf("multilateral clearing failed: %w", err)
	}

	// 保存清算结果
	for _, result := range results {
		nettingResult, ok := result.(*NettingResult)
		if !ok || nettingResult == nil {
			continue
		}
		if err := cm.nettingRepo.SaveNettingResult(ctx, nettingResult); err != nil {
			return fmt.Errorf("failed to save multilateral netting result: %w", err)
		}
	}

	// 更新指令状态
	for _, instruction := range instructions {
		instruction.Status = ClearingSettled
		instruction.UpdatedAt = time.Now()
		cm.instructionRepo.UpdateInstruction(ctx, instruction)
	}

	return nil
}

// processBilateralBatch 处理双边清算批次
func (cm *ClearingManager) processBilateralBatch(ctx context.Context, batch *ClearingBatch) error {
	// 获取批次中的清算指令
	instructions, err := cm.instructionRepo.GetInstructionsByBatch(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("failed to get batch instructions: %w", err)
	}

	// 按交易对手分组
	counterpartyGroups := cm.groupByCounterparty(instructions)

	// 处理每个双边关系
	for _, group := range counterpartyGroups {
		err := cm.processBilateralClearing(ctx, group)
		if err != nil {
			return fmt.Errorf("bilateral clearing failed: %w", err)
		}
	}

	return nil
}

// processCCPBatch 处理中央对手方清算批次
func (cm *ClearingManager) processCCPBatch(ctx context.Context, batch *ClearingBatch) error {
	// 获取批次中的清算指令
	instructions, err := cm.instructionRepo.GetInstructionsByBatch(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("failed to get batch instructions: %w", err)
	}

	// 中央对手方介入
	for _, instruction := range instructions {
		// 将原始交易拆分为两个新交易
		buyToCCP, sellToCCP, err := cm.splitTradeForCCP(instruction)
		if err != nil {
			return fmt.Errorf("failed to split trade for CCP: %w", err)
		}

		// 处理与CCP的交易
		err = cm.processCCPTrade(ctx, buyToCCP)
		if err != nil {
			return fmt.Errorf("failed to process CCP buy trade: %w", err)
		}

		err = cm.processCCPTrade(ctx, sellToCCP)
		if err != nil {
			return fmt.Errorf("failed to process CCP sell trade: %w", err)
		}

		// 更新原始指令状态
		instruction.Status = ClearingSettled
		instruction.UpdatedAt = time.Now()
		cm.instructionRepo.UpdateInstruction(ctx, instruction)
	}

	return nil
}

// processClearingInstruction 处理清算指令
func (cm *ClearingManager) processClearingInstruction(ctx context.Context, instruction *ClearingInstruction) error {
	// 检查指令状态
	if instruction.Status != ClearingPending && instruction.Status != ClearingValidated {
		return fmt.Errorf("instruction is not in processable state: %s", instruction.Status)
	}

	// 验证指令
	err := cm.validateClearingInstruction(ctx, instruction)
	if err != nil {
		// 检查是否需要重试
		if instruction.RetryCount < instruction.MaxRetries {
			instruction.RetryCount++
			instruction.UpdatedAt = time.Now()
			cm.instructionRepo.UpdateInstruction(ctx, instruction)

			// 延迟重试
			time.Sleep(time.Duration(instruction.RetryCount) * time.Minute)
			return cm.processClearingInstruction(ctx, instruction)
		}
		return fmt.Errorf("clearing validation failed after %d retries: %w", instruction.MaxRetries, err)
	}

	// 执行清算
	err = cm.executeClearing(ctx, instruction)
	if err != nil {
		return fmt.Errorf("clearing execution failed: %w", err)
	}

	// 更新指令状态
	instruction.Status = ClearingSettled
	instruction.UpdatedAt = time.Now()

	err = cm.instructionRepo.UpdateInstruction(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to update instruction status: %w", err)
	}

	return nil
}

// validateClearingInstruction 验证清算指令
func (cm *ClearingManager) validateClearingInstruction(ctx context.Context, instruction *ClearingInstruction) error {
	// 检查会员状态
	buyMember, err := cm.memberRepo.GetMember(ctx, instruction.BuyMemberID)
	if err != nil {
		return fmt.Errorf("failed to get buy member: %w", err)
	}

	if buyMember.Status != "ACTIVE" {
		return fmt.Errorf("buy member is not active: %s", buyMember.Status)
	}

	sellMember, err := cm.memberRepo.GetMember(ctx, instruction.SellMemberID)
	if err != nil {
		return fmt.Errorf("failed to get sell member: %w", err)
	}

	if sellMember.Status != "ACTIVE" {
		return fmt.Errorf("sell member is not active: %s", sellMember.Status)
	}

	// 检查信用限额
	if instruction.NetAmount > buyMember.CreditLimit {
		return fmt.Errorf("buy member credit limit exceeded: amount=%.2f, limit=%.2f",
			instruction.NetAmount, buyMember.CreditLimit)
	}

	// 检查保证金要求
	marginCoverage := (buyMember.Collateral + sellMember.Collateral) / instruction.TotalMargin
	if marginCoverage < 1.0 {
		return fmt.Errorf("insufficient margin coverage: %.2f", marginCoverage)
	}

	// 检查风险限额
	if instruction.RiskScore > instruction.RiskLimit {
		return fmt.Errorf("risk limit exceeded: score=%.2f, limit=%.2f",
			instruction.RiskScore, instruction.RiskLimit)
	}

	return nil
}

// executeClearing 执行清算
func (cm *ClearingManager) executeClearing(ctx context.Context, instruction *ClearingInstruction) error {
	// 记录清算交易
	clearingTrade := &ClearingTrade{
		ClearingID:   instruction.ID,
		TradeID:      instruction.TradeID,
		BuyMemberID:  instruction.BuyMemberID,
		SellMemberID: instruction.SellMemberID,
		Symbol:       instruction.Symbol,
		Quantity:     instruction.Quantity,
		Price:        instruction.Price,
		Amount:       instruction.NetAmount,
		ClearingDate: instruction.ClearingDate,
		Status:       "CLEARED",
		CreatedAt:    time.Now(),
	}

	err := cm.instructionRepo.RecordClearingTrade(ctx, clearingTrade)
	if err != nil {
		return fmt.Errorf("failed to record clearing trade: %w", err)
	}

	// 更新会员头寸
	err = cm.updateMemberPositions(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to update member positions: %w", err)
	}

	// 更新保证金
	err = cm.updateMarginRequirements(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to update margin requirements: %w", err)
	}

	return nil
}

// checkMarginRequirements 检查保证金要求
func (cm *ClearingManager) checkMarginRequirements(ctx context.Context, instruction *ClearingInstruction,
	buyMember, sellMember *ClearingMember) error {

	// 检查买方保证金
	if buyMember.Collateral < instruction.TotalMargin {
		// 创建保证金催缴
		marginCall := &MarginCall{
			ID:                generateMarginCallID(),
			CallRef:           generateCallRef(),
			MemberID:          buyMember.ID,
			CallType:          "INITIAL",
			Amount:            instruction.TotalMargin - buyMember.Collateral,
			Currency:          "USD",
			DueDate:           time.Now().Add(24 * time.Hour),
			Status:            "PENDING",
			OutstandingAmount: instruction.TotalMargin - buyMember.Collateral,
			CallDate:          time.Now(),
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := cm.marginRepo.SaveMarginCall(ctx, marginCall)
		if err != nil {
			return fmt.Errorf("failed to create margin call: %w", err)
		}
	}

	// 检查卖方保证金
	if sellMember.Collateral < instruction.TotalMargin {
		// 创建保证金催缴
		marginCall := &MarginCall{
			ID:                generateMarginCallID(),
			CallRef:           generateCallRef(),
			MemberID:          sellMember.ID,
			CallType:          "INITIAL",
			Amount:            instruction.TotalMargin - sellMember.Collateral,
			Currency:          "USD",
			DueDate:           time.Now().Add(24 * time.Hour),
			Status:            "PENDING",
			OutstandingAmount: instruction.TotalMargin - sellMember.Collateral,
			CallDate:          time.Now(),
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := cm.marginRepo.SaveMarginCall(ctx, marginCall)
		if err != nil {
			return fmt.Errorf("failed to create margin call: %w", err)
		}
	}

	return nil
}

// Helper methods

func (cm *ClearingManager) findApplicableRule(symbol, market string) (*ClearingRule, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 查找最匹配的规则
	var bestMatch *ClearingRule
	var bestScore int

	for _, rule := range cm.rules {
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

func (cm *ClearingManager) validateClearingMembers(ctx context.Context, buyMemberID, sellMemberID string) (*ClearingMember, *ClearingMember, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 检查买方会员
	buyMember, exists := cm.members[buyMemberID]
	if !exists {
		return nil, nil, fmt.Errorf("buy member not found: %s", buyMemberID)
	}

	if buyMember.Status != "ACTIVE" {
		return nil, nil, fmt.Errorf("buy member is not active: %s", buyMember.Status)
	}

	// 检查卖方会员
	sellMember, exists := cm.members[sellMemberID]
	if !exists {
		return nil, nil, fmt.Errorf("sell member not found: %s", sellMemberID)
	}

	if sellMember.Status != "ACTIVE" {
		return nil, nil, fmt.Errorf("sell member is not active: %s", sellMember.Status)
	}

	return buyMember, sellMember, nil
}

func (cm *ClearingManager) calculateMarginRequirements(ctx context.Context, trade *Trade,
	buyMember, sellMember *ClearingMember, rule *ClearingRule) (*MarginRequirements, error) {

	// 根据保证金模型计算
	var marginReq MarginRequirements

	switch rule.MarginModel {
	case "SPAN":
		marginReq = cm.calculateSPANMargin(trade, buyMember, sellMember)
	case "CORE":
		marginReq = cm.calculateCOREMargin(trade, buyMember, sellMember)
	case "VAR":
		marginReq = cm.calculateVARMargin(trade, buyMember, sellMember)
	default:
		marginReq = cm.calculateStandardMargin(trade, buyMember, sellMember)
	}

	return &marginReq, nil
}

func (cm *ClearingManager) calculateStandardMargin(trade *Trade, buyMember, sellMember *ClearingMember) MarginRequirements {
	// 标准保证金计算
	tradeValue := trade.Quantity * trade.Price

	return MarginRequirements{
		InitialMargin:     tradeValue * 0.05, // 5%初始保证金
		VariationMargin:   tradeValue * 0.02, // 2%变动保证金
		MaintenanceMargin: tradeValue * 0.03, // 3%维持保证金
		TotalMargin:       tradeValue * 0.10, // 10%总保证金
	}
}

func (cm *ClearingManager) calculateSPANMargin(trade *Trade, buyMember, sellMember *ClearingMember) MarginRequirements {
	// SPAN保证金计算（简化版）
	// 基于风险阵列计算
	riskArray := cm.calculateRiskArray(trade)

	return MarginRequirements{
		InitialMargin:     riskArray.InitialMargin,
		VariationMargin:   riskArray.VariationMargin,
		MaintenanceMargin: riskArray.MaintenanceMargin,
		TotalMargin:       riskArray.TotalMargin,
	}
}

func (cm *ClearingManager) groupPositionsByMemberAndSymbol(instructions []*ClearingInstruction) map[string]map[string]*MemberPosition {
	positions := make(map[string]map[string]*MemberPosition)

	for _, instruction := range instructions {
		// 买方头寸
		if positions[instruction.BuyMemberID] == nil {
			positions[instruction.BuyMemberID] = make(map[string]*MemberPosition)
		}

		if positions[instruction.BuyMemberID][instruction.Symbol] == nil {
			positions[instruction.BuyMemberID][instruction.Symbol] = &MemberPosition{
				MemberID: instruction.BuyMemberID,
				Symbol:   instruction.Symbol,
			}
		}

		positions[instruction.BuyMemberID][instruction.Symbol].BuyQuantity += instruction.Quantity
		positions[instruction.BuyMemberID][instruction.Symbol].BuyAmount += instruction.NetAmount

		// 卖方头寸
		if positions[instruction.SellMemberID] == nil {
			positions[instruction.SellMemberID] = make(map[string]*MemberPosition)
		}

		if positions[instruction.SellMemberID][instruction.Symbol] == nil {
			positions[instruction.SellMemberID][instruction.Symbol] = &MemberPosition{
				MemberID: instruction.SellMemberID,
				Symbol:   instruction.Symbol,
			}
		}

		positions[instruction.SellMemberID][instruction.Symbol].SellQuantity += instruction.Quantity
		positions[instruction.SellMemberID][instruction.Symbol].SellAmount += instruction.NetAmount
	}

	return positions
}

func (cm *ClearingManager) calculateNetting(ctx context.Context, positions map[string]map[string]*MemberPosition,
	clearingDate time.Time) ([]*NettingResult, error) {

	var results []*NettingResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 为每个会员并行计算净额
	for memberID, symbolPositions := range positions {
		wg.Add(1)
		go func(mid string, sps map[string]*MemberPosition) {
			defer wg.Done()
			var localResults []*NettingResult
			for symbol, position := range sps {
				netPosition := position.BuyQuantity - position.SellQuantity
				netAmount := position.BuyAmount - position.SellAmount

				result := &NettingResult{
					ID:             generateNettingID(),
					NettingID:      generateNettingRef(),
					MemberID:       mid,
					Symbol:         symbol,
					SettlementDate: clearingDate,
					GrossBuy:       position.BuyQuantity,
					GrossSell:      position.SellQuantity,
					NetPosition:    netPosition,
					NetAmount:      netAmount,
					Status:         "CALCULATED",
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}
				localResults = append(localResults, result)
			}
			mu.Lock()
			results = append(results, localResults...)
			mu.Unlock()
		}(memberID, symbolPositions)
	}
	wg.Wait()

	return results, nil
}

// Helper functions

func generateInstructionID() string {
	return fmt.Sprintf("CLEAR_INST_%d", time.Now().UnixNano())
}

func generateInstructionRef() string {
	return fmt.Sprintf("CIREF%d", time.Now().UnixNano())
}

func generateMarginCallID() string {
	return fmt.Sprintf("MARGIN_CALL_%d", time.Now().UnixNano())
}

func generateCallRef() string {
	return fmt.Sprintf("MCREF%d", time.Now().UnixNano())
}

func generateNettingID() string {
	return fmt.Sprintf("NETTING_%d", time.Now().UnixNano())
}

func generateNettingRef() string {
	return fmt.Sprintf("NETREF%d", time.Now().UnixNano())
}

// Data structures

type Trade struct {
	TradeID        string    `json:"trade_id"`
	BuyMemberID    string    `json:"buy_member_id"`
	SellMemberID   string    `json:"sell_member_id"`
	Symbol         string    `json:"symbol"`
	Market         string    `json:"market"`
	Quantity       float64   `json:"quantity"`
	Price          float64   `json:"price"`
	TradeDate      time.Time `json:"trade_date"`
	SettlementDate time.Time `json:"settlement_date"`
}

type MarginRequirements struct {
	InitialMargin     float64 `json:"initial_margin"`
	VariationMargin   float64 `json:"variation_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	TotalMargin       float64 `json:"total_margin"`
}

type ClearingRiskArray struct {
	InitialMargin     float64 `json:"initial_margin"`
	VariationMargin   float64 `json:"variation_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	TotalMargin       float64 `json:"total_margin"`
}

type MemberPosition struct {
	MemberID     string  `json:"member_id"`
	Symbol       string  `json:"symbol"`
	BuyQuantity  float64 `json:"buy_quantity"`
	SellQuantity float64 `json:"sell_quantity"`
	BuyAmount    float64 `json:"buy_amount"`
	SellAmount   float64 `json:"sell_amount"`
}

type ClearingTrade struct {
	ClearingID   string    `json:"clearing_id"`
	TradeID      string    `json:"trade_id"`
	BuyMemberID  string    `json:"buy_member_id"`
	SellMemberID string    `json:"sell_member_id"`
	Symbol       string    `json:"symbol"`
	Quantity     float64   `json:"quantity"`
	Price        float64   `json:"price"`
	Amount       float64   `json:"amount"`
	ClearingDate time.Time `json:"clearing_date"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// Repository interfaces

type ClearingMemberRepository interface {
	SaveMember(ctx context.Context, member *ClearingMember) error
	GetMember(ctx context.Context, id string) (*ClearingMember, error)
	GetMemberByCode(ctx context.Context, code string) (*ClearingMember, error)
	GetActiveMembers(ctx context.Context) ([]*ClearingMember, error)
	UpdateMember(ctx context.Context, member *ClearingMember) error
	DeleteMember(ctx context.Context, id string) error
}

type ClearingInstructionRepository interface {
	SaveInstruction(ctx context.Context, instruction *ClearingInstruction) error
	GetInstruction(ctx context.Context, id string) (*ClearingInstruction, error)
	GetInstructionByRef(ctx context.Context, ref string) (*ClearingInstruction, error)
	GetInstructionsByBatch(ctx context.Context, batchID string) ([]*ClearingInstruction, error)
	GetPendingInstructionsByDate(ctx context.Context, clearingDate time.Time) ([]*ClearingInstruction, error)
	UpdateInstruction(ctx context.Context, instruction *ClearingInstruction) error
	DeleteInstruction(ctx context.Context, id string) error
	RecordClearingTrade(ctx context.Context, trade *ClearingTrade) error
}

type ClearingBatchRepository interface {
	SaveBatch(ctx context.Context, batch *ClearingBatch) error
	GetBatch(ctx context.Context, id string) (*ClearingBatch, error)
	GetBatchByDate(ctx context.Context, clearingDate time.Time, clearingType ClearingType) (*ClearingBatch, error)
	GetBatchesByStatus(ctx context.Context, status string, startDate, endDate time.Time) ([]*ClearingBatch, error)
	UpdateBatch(ctx context.Context, batch *ClearingBatch) error
	DeleteBatch(ctx context.Context, id string) error
}

type ClearingRuleRepository interface {
	SaveRule(ctx context.Context, rule *ClearingRule) error
	GetRule(ctx context.Context, id string) (*ClearingRule, error)
	GetRulesBySymbol(ctx context.Context, symbol string) ([]*ClearingRule, error)
	GetEnabledRules(ctx context.Context) ([]*ClearingRule, error)
	UpdateRule(ctx context.Context, rule *ClearingRule) error
	DeleteRule(ctx context.Context, id string) error
}

type MarginRepository interface {
	SaveMarginCall(ctx context.Context, call *MarginCall) error
	GetMarginCall(ctx context.Context, id string) (*MarginCall, error)
	GetMarginCallsByMember(ctx context.Context, memberID string, status string) ([]*MarginCall, error)
	GetMemberPositions(ctx context.Context, memberID string) ([]*Position, error)
	UpdateMarginCall(ctx context.Context, call *MarginCall) error
	DeleteMarginCall(ctx context.Context, id string) error
}

type NettingRepository interface {
	SaveNettingResult(ctx context.Context, result *NettingResult) error
	GetNettingResult(ctx context.Context, id string) (*NettingResult, error)
	GetNettingResultsByMember(ctx context.Context, memberID string, settlementDate time.Time) ([]*NettingResult, error)
	UpdateNettingResult(ctx context.Context, result *NettingResult) error
	DeleteNettingResult(ctx context.Context, id string) error
}

type RiskManager interface {
	CalculateRiskScore(trade *Trade, member *ClearingMember) float64
	CalculateRiskLimit(member *ClearingMember) float64
	CalculateRiskArray(trade *Trade) *ClearingRiskArray
}

// 以下方法需要实现，但为了简洁省略了具体实现
func (cm *ClearingManager) calculateRiskArray(trade *Trade) *ClearingRiskArray {
	// 简化版风险阵列计算
	tradeValue := trade.Quantity * trade.Price
	return &ClearingRiskArray{
		InitialMargin:     tradeValue * 0.08,
		VariationMargin:   tradeValue * 0.01,
		MaintenanceMargin: tradeValue * 0.05,
		TotalMargin:       tradeValue * 0.14,
	}
}

func (cm *ClearingManager) calculateCOREMargin(trade *Trade, buyMember, sellMember *ClearingMember) MarginRequirements {
	// CORE 保证金模型 (简化版)
	tradeValue := trade.Quantity * trade.Price
	return MarginRequirements{
		InitialMargin:     tradeValue * 0.07,
		VariationMargin:   tradeValue * 0.015,
		MaintenanceMargin: tradeValue * 0.045,
		TotalMargin:       tradeValue * 0.13,
	}
}

func (cm *ClearingManager) calculateVARMargin(trade *Trade, buyMember, sellMember *ClearingMember) MarginRequirements {
	// 基于蒙特卡洛模拟的 VaR 保证金计算 (99% 置信度，1000次模拟)
	const (
		simCount   = 1000
		confidence = 0.99
	)

	tradeValue := trade.Quantity * trade.Price
	volatility := 0.02 // 假设日波动率为 2%

	returns := make([]float64, simCount)
	for i := 0; i < simCount; i++ {
		r := rand.NormFloat64() * volatility
		returns[i] = tradeValue * r
	}

	sort.Float64s(returns)
	varIdx := int(float64(simCount) * (1.0 - confidence))
	varAmount := math.Abs(returns[varIdx])

	return MarginRequirements{
		InitialMargin:     varAmount * 1.5,
		VariationMargin:   tradeValue * 0.01,
		MaintenanceMargin: varAmount * 1.1,
		TotalMargin:       varAmount*1.5 + tradeValue*0.01,
	}
}

func (cm *ClearingManager) buildMultilateralMatrix(instructions []*ClearingInstruction) (interface{}, error) {
	return nil, nil
}

func (cm *ClearingManager) executeMultilateralClearing(ctx context.Context, matrix interface{}) ([]interface{}, error) {
	return nil, nil
}

func (cm *ClearingManager) groupByCounterparty(instructions []*ClearingInstruction) map[string][]*ClearingInstruction {
	return make(map[string][]*ClearingInstruction)
}

func (cm *ClearingManager) processBilateralClearing(ctx context.Context, instructions []*ClearingInstruction) error {
	return nil
}

func (cm *ClearingManager) splitTradeForCCP(instruction *ClearingInstruction) (*ClearingInstruction, *ClearingInstruction, error) {
	return nil, nil, nil
}

func (cm *ClearingManager) processCCPTrade(ctx context.Context, instruction *ClearingInstruction) error {
	return nil
}

func (cm *ClearingManager) updateMemberPositions(ctx context.Context, instruction *ClearingInstruction) error {
	return nil
}

func (cm *ClearingManager) updateMarginRequirements(ctx context.Context, instruction *ClearingInstruction) error {
	return nil
}
