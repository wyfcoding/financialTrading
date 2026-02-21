//go:build clearing_experimental
// +build clearing_experimental

package domain

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultType 违约类型
type DefaultType string

const (
	DefaultMarginCall  DefaultType = "MARGIN_CALL" // 保证金催缴违约
	DefaultSettlement  DefaultType = "SETTLEMENT"  // 结算违约
	DefaultCredit      DefaultType = "CREDIT"      // 信用违约
	DefaultOperational DefaultType = "OPERATIONAL" // 操作违约
)

// DefaultStatus 违约状态
type DefaultStatus string

const (
	DefaultPending    DefaultStatus = "PENDING"     // 待处理
	DefaultActive     DefaultStatus = "ACTIVE"      // 活跃
	DefaultResolved   DefaultStatus = "RESOLVED"    // 已解决
	DefaultWrittenOff DefaultStatus = "WRITTEN_OFF" // 已核销
)

// DefaultEvent 违约事件
type DefaultEvent struct {
	ID          string        `json:"id"`
	DefaultNo   string        `json:"default_no"`
	MemberID    string        `json:"member_id"`
	DefaultType DefaultType   `json:"default_type"`
	Status      DefaultStatus `json:"status"`

	// 违约详情
	TriggerEventID   string  `json:"trigger_event_id"`
	TriggerEventType string  `json:"trigger_event_type"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	Description      string  `json:"description"`

	// 时间戳
	DefaultDate  time.Time  `json:"default_date"`
	ReportedAt   time.Time  `json:"reported_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
	WrittenOffAt *time.Time `json:"written_off_at"`

	// 处理信息
	HandlerID      string  `json:"handler_id"`
	ResolutionType string  `json:"resolution_type"`
	RecoveryAmount float64 `json:"recovery_amount"`
	LossAmount     float64 `json:"loss_amount"`

	// 元数据
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// DefaultHandler 违约处理器
type DefaultHandler struct {
	defaultRepo     DefaultRepository
	marginRepo      MarginRepository
	settlementRepo  SettlementRepository
	legalService    LegalService
	recoveryService RecoveryService
	mu              sync.RWMutex
	config          *DefaultConfig
	activeDefaults  map[string]*DefaultEvent
	notificationLog map[string][]*Notification
	liquidationLog  map[string][]*LiquidationRecord
	writeOffLog     map[string]*WriteOffRecord
}

// DefaultConfig 违约配置
type DefaultConfig struct {
	GracePeriod          time.Duration `json:"grace_period"`
	MaxGracePeriods      int           `json:"max_grace_periods"`
	AutoEscalation       bool          `json:"auto_escalation"`
	EscalationLevels     []string      `json:"escalation_levels"`
	LiquidationThreshold float64       `json:"liquidation_threshold"`
	RecoveryThreshold    float64       `json:"recovery_threshold"`
	LegalThreshold       float64       `json:"legal_threshold"`
}

// NewDefaultHandler 创建违约处理器
func NewDefaultHandler(defaultRepo DefaultRepository,
	marginRepo MarginRepository,
	settlementRepo SettlementRepository,
	legalService LegalService,
	recoveryService RecoveryService) *DefaultHandler {

	return &DefaultHandler{
		defaultRepo:     defaultRepo,
		marginRepo:      marginRepo,
		settlementRepo:  settlementRepo,
		legalService:    legalService,
		recoveryService: recoveryService,
		config: &DefaultConfig{
			GracePeriod:          48 * time.Hour,
			MaxGracePeriods:      2,
			AutoEscalation:       true,
			EscalationLevels:     []string{"SUPERVISOR", "MANAGEMENT", "LEGAL"},
			LiquidationThreshold: 0.7,
			RecoveryThreshold:    0.3,
			LegalThreshold:       0.5,
		},
		activeDefaults:  make(map[string]*DefaultEvent),
		notificationLog: make(map[string][]*Notification),
		liquidationLog:  make(map[string][]*LiquidationRecord),
		writeOffLog:     make(map[string]*WriteOffRecord),
	}
}

// ReportDefault 报告违约
func (dh *DefaultHandler) ReportDefault(ctx context.Context, memberID string,
	defaultType DefaultType, triggerEventID, triggerEventType string,
	amount float64, currency, description string) (*DefaultEvent, error) {

	// 创建违约事件
	defaultEvent := &DefaultEvent{
		ID:               generateDefaultID(),
		DefaultNo:        generateDefaultNo(),
		MemberID:         memberID,
		DefaultType:      defaultType,
		Status:           DefaultPending,
		TriggerEventID:   triggerEventID,
		TriggerEventType: triggerEventType,
		Amount:           amount,
		Currency:         currency,
		Description:      description,
		DefaultDate:      time.Now(),
		ReportedAt:       time.Now(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         make(map[string]interface{}),
	}

	// 保存违约事件
	err := dh.defaultRepo.SaveDefault(ctx, defaultEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to save default event: %w", err)
	}

	// 添加到活跃违约列表
	dh.mu.Lock()
	dh.activeDefaults[defaultEvent.ID] = defaultEvent
	dh.mu.Unlock()

	// 启动违约处理流程
	go dh.handleDefault(ctx, defaultEvent)

	return defaultEvent, nil
}

// handleDefault 处理违约
func (dh *DefaultHandler) handleDefault(ctx context.Context, defaultEvent *DefaultEvent) {
	// 记录开始处理
	defaultEvent.Status = DefaultActive
	defaultEvent.UpdatedAt = time.Now()

	// 根据违约类型处理
	switch defaultEvent.DefaultType {
	case DefaultMarginCall:
		dh.handleMarginCallDefault(ctx, defaultEvent)
	case DefaultSettlement:
		dh.handleSettlementDefault(ctx, defaultEvent)
	case DefaultCredit:
		dh.handleCreditDefault(ctx, defaultEvent)
	case DefaultOperational:
		dh.handleOperationalDefault(ctx, defaultEvent)
	default:
		dh.handleGenericDefault(ctx, defaultEvent)
	}

	// 保存更新
	dh.defaultRepo.UpdateDefault(ctx, defaultEvent)
}

// handleMarginCallDefault 处理保证金催缴违约
func (dh *DefaultHandler) handleMarginCallDefault(ctx context.Context, defaultEvent *DefaultEvent) {
	// 获取保证金催缴
	marginCall, err := dh.marginRepo.GetMarginCall(ctx, defaultEvent.TriggerEventID)
	if err != nil {
		fmt.Printf("Failed to get margin call %s: %v\n", defaultEvent.TriggerEventID, err)
		return
	}

	// 检查是否已解决
	if marginCall.Status == "COMPLETED" || marginCall.Status == "LIQUIDATED" {
		defaultEvent.Status = DefaultResolved
		defaultEvent.ResolvedAt = timePtrNow()
		defaultEvent.ResolutionType = "AUTO_RESOLVED"
		return
	}

	// 给予宽限期
	gracePeriods := 0
	for gracePeriods < dh.config.MaxGracePeriods {
		// 发送违约通知
		dh.sendDefaultNotification(ctx, defaultEvent, gracePeriods+1)

		// 等待宽限期
		time.Sleep(dh.config.GracePeriod)

		// 检查是否已解决
		marginCall, err = dh.marginRepo.GetMarginCall(ctx, defaultEvent.TriggerEventID)
		if err != nil {
			fmt.Printf("Failed to get margin call %s: %v\n", defaultEvent.TriggerEventID, err)
			break
		}

		if marginCall.Status == "COMPLETED" {
			defaultEvent.Status = DefaultResolved
			defaultEvent.ResolvedAt = timePtrNow()
			defaultEvent.ResolutionType = "PAID_DURING_GRACE"
			return
		}

		gracePeriods++
	}

	// 宽限期已过，执行清算
	dh.executeDefaultLiquidation(ctx, defaultEvent, marginCall)
}

// handleSettlementDefault 处理结算违约
func (dh *DefaultHandler) handleSettlementDefault(ctx context.Context, defaultEvent *DefaultEvent) {
	// 获取结算指令
	instruction, err := dh.settlementRepo.GetInstruction(ctx, defaultEvent.TriggerEventID)
	if err != nil {
		fmt.Printf("Failed to get settlement instruction %s: %v\n", defaultEvent.TriggerEventID, err)
		return
	}

	// 检查是否已解决
	if instruction.Status == SettlementSettled {
		defaultEvent.Status = DefaultResolved
		defaultEvent.ResolvedAt = timePtrNow()
		defaultEvent.ResolutionType = "SETTLED"
		return
	}

	// 给予宽限期
	gracePeriods := 0
	for gracePeriods < dh.config.MaxGracePeriods {
		// 发送违约通知
		dh.sendDefaultNotification(ctx, defaultEvent, gracePeriods+1)

		// 等待宽限期
		time.Sleep(dh.config.GracePeriod)

		// 检查是否已解决
		instruction, err = dh.settlementRepo.GetInstruction(ctx, defaultEvent.TriggerEventID)
		if err != nil {
			fmt.Printf("Failed to get settlement instruction %s: %v\n", defaultEvent.TriggerEventID, err)
			break
		}

		if instruction.Status == SettlementSettled {
			defaultEvent.Status = DefaultResolved
			defaultEvent.ResolvedAt = timePtrNow()
			defaultEvent.ResolutionType = "SETTLED_DURING_GRACE"
			return
		}

		gracePeriods++
	}

	// 宽限期已过，启动追偿
	dh.initiateRecovery(ctx, defaultEvent, instruction)
}

// handleCreditDefault 处理信用违约
func (dh *DefaultHandler) handleCreditDefault(ctx context.Context, defaultEvent *DefaultEvent) {
	defaultEvent.Metadata["credit_default"] = true
	defaultEvent.Metadata["credit_default_at"] = time.Now()

	assessment := dh.assessRecovery(ctx, defaultEvent)
	if assessment.RecoveryProbability >= dh.config.RecoveryThreshold {
		if err := dh.recoveryService.InitiateRecovery(ctx, defaultEvent); err != nil {
			if defaultEvent.Amount >= dh.config.LegalThreshold {
				dh.initiateLegalAction(ctx, defaultEvent)
				defaultEvent.ResolutionType = "CREDIT_LEGAL_ACTION"
			} else {
				dh.writeOffDefault(ctx, defaultEvent)
			}
			return
		}
		defaultEvent.Status = DefaultResolved
		defaultEvent.ResolvedAt = timePtrNow()
		defaultEvent.ResolutionType = "CREDIT_RECOVERY_INITIATED"
		return
	}

	if defaultEvent.Amount >= dh.config.LegalThreshold {
		dh.initiateLegalAction(ctx, defaultEvent)
		defaultEvent.Status = DefaultResolved
		defaultEvent.ResolvedAt = timePtrNow()
		defaultEvent.ResolutionType = "CREDIT_ESCALATED_TO_LEGAL"
		return
	}

	dh.writeOffDefault(ctx, defaultEvent)
}

// handleOperationalDefault 处理操作违约
func (dh *DefaultHandler) handleOperationalDefault(ctx context.Context, defaultEvent *DefaultEvent) {
	defaultEvent.Metadata["operational_default"] = true
	defaultEvent.Metadata["operational_default_at"] = time.Now()

	dh.escalateDefault(ctx, defaultEvent)
	if defaultEvent.Status == DefaultActive || defaultEvent.Status == DefaultPending {
		defaultEvent.Status = DefaultResolved
		defaultEvent.ResolvedAt = timePtrNow()
		defaultEvent.ResolutionType = "OPERATIONAL_ESCALATED"
	}
}

// handleGenericDefault 处理通用违约
func (dh *DefaultHandler) handleGenericDefault(ctx context.Context, defaultEvent *DefaultEvent) {
	// 通用违约处理逻辑
	// 发送通知，给予宽限期，然后升级

	gracePeriods := 0
	for gracePeriods < dh.config.MaxGracePeriods {
		// 发送违约通知
		dh.sendDefaultNotification(ctx, defaultEvent, gracePeriods+1)

		// 等待宽限期
		time.Sleep(dh.config.GracePeriod)

		// 检查是否已解决
		if dh.isResolved(ctx, defaultEvent.ID) {
			defaultEvent.Status = DefaultResolved
			defaultEvent.ResolvedAt = timePtrNow()
			defaultEvent.ResolutionType = "GENERIC_RESOLVED_DURING_GRACE"
			return
		}

		gracePeriods++
	}

	// 宽限期已过，升级处理
	dh.escalateDefault(ctx, defaultEvent)
}

// executeDefaultLiquidation 执行违约清算
func (dh *DefaultHandler) executeDefaultLiquidation(ctx context.Context, defaultEvent *DefaultEvent, marginCall *MarginCall) {
	// 获取会员持仓
	positions, err := dh.marginRepo.GetMemberPositions(ctx, defaultEvent.MemberID)
	if err != nil {
		fmt.Printf("Failed to get member positions for %s: %v\n", defaultEvent.MemberID, err)
		return
	}

	// 计算需要清算的金额
	requiredAmount := marginCall.OutstandingAmount

	// 按风险权重排序持仓
	sortedPositions := dh.sortPositionsByRisk(positions)

	// 执行清算
	var liquidatedAmount float64
	for _, position := range sortedPositions {
		if liquidatedAmount >= requiredAmount {
			break
		}

		// 计算本次清算金额
		liquidationAmount := dh.calculateLiquidationAmount(position, requiredAmount-liquidatedAmount)

		// 执行清算
		err := dh.liquidatePosition(ctx, position, liquidationAmount)
		if err != nil {
			fmt.Printf("Failed to liquidate position %s: %v\n", position.ID, err)
			continue
		}

		liquidatedAmount += liquidationAmount

		// 记录清算
		dh.recordDefaultLiquidation(ctx, defaultEvent, position, liquidationAmount)
	}

	// 更新违约状态
	if liquidatedAmount >= requiredAmount {
		defaultEvent.Status = DefaultResolved
		defaultEvent.ResolvedAt = timePtrNow()
		defaultEvent.ResolutionType = "LIQUIDATED"
		defaultEvent.RecoveryAmount = liquidatedAmount
		defaultEvent.LossAmount = requiredAmount - liquidatedAmount
	} else {
		// 清算不足，启动追偿
		dh.initiateRecovery(ctx, defaultEvent, nil)
	}

	// 更新保证金催缴
	marginCall.Status = "LIQUIDATED"
	marginCall.OutstandingAmount = 0
	marginCall.UpdatedAt = time.Now()

	dh.marginRepo.UpdateMarginCall(ctx, marginCall)
}

// initiateRecovery 启动追偿
func (dh *DefaultHandler) initiateRecovery(ctx context.Context, defaultEvent *DefaultEvent, instruction *SettlementInstruction) {
	// 评估追偿可能性
	recoveryAssessment := dh.assessRecovery(ctx, defaultEvent)

	if recoveryAssessment.RecoveryProbability >= dh.config.RecoveryThreshold {
		// 启动追偿流程
		err := dh.recoveryService.InitiateRecovery(ctx, defaultEvent)
		if err != nil {
			fmt.Printf("Failed to initiate recovery for default %s: %v\n", defaultEvent.ID, err)

			// 追偿失败，升级到法律程序
			if defaultEvent.Amount >= dh.config.LegalThreshold {
				dh.initiateLegalAction(ctx, defaultEvent)
			} else {
				// 金额较小，核销
				dh.writeOffDefault(ctx, defaultEvent)
			}
		}
	} else if defaultEvent.Amount >= dh.config.LegalThreshold {
		// 直接启动法律程序
		dh.initiateLegalAction(ctx, defaultEvent)
	} else {
		// 核销
		dh.writeOffDefault(ctx, defaultEvent)
	}
}

// escalateDefault 升级违约
func (dh *DefaultHandler) escalateDefault(ctx context.Context, defaultEvent *DefaultEvent) {
	// 根据配置的升级级别处理
	for _, level := range dh.config.EscalationLevels {
		// 发送升级通知
		dh.sendEscalationNotification(ctx, defaultEvent, level)

		// 等待响应
		time.Sleep(dh.config.GracePeriod)

		// 检查是否已解决
		if dh.isResolved(ctx, defaultEvent.ID) {
			defaultEvent.Status = DefaultResolved
			defaultEvent.ResolvedAt = timePtrNow()
			defaultEvent.ResolutionType = fmt.Sprintf("RESOLVED_AT_%s", level)
			return
		}

		// 如果仍未解决，继续升级
	}

	// 所有升级级别都尝试过，启动追偿或法律程序
	if defaultEvent.Amount >= dh.config.LegalThreshold {
		dh.initiateLegalAction(ctx, defaultEvent)
	} else {
		dh.initiateRecovery(ctx, defaultEvent, nil)
	}
}

// initiateLegalAction 启动法律程序
func (dh *DefaultHandler) initiateLegalAction(ctx context.Context, defaultEvent *DefaultEvent) {
	// 启动法律程序
	err := dh.legalService.InitiateLegalAction(ctx, defaultEvent)
	if err != nil {
		fmt.Printf("Failed to initiate legal action for default %s: %v\n", defaultEvent.ID, err)

		// 法律程序失败，核销
		dh.writeOffDefault(ctx, defaultEvent)
		return
	}

	// 标记为法律程序中
	defaultEvent.Metadata["legal_action_initiated"] = true
	defaultEvent.Metadata["legal_action_date"] = time.Now()
	defaultEvent.UpdatedAt = time.Now()
}

// writeOffDefault 核销违约
func (dh *DefaultHandler) writeOffDefault(ctx context.Context, defaultEvent *DefaultEvent) {
	// 核销违约
	defaultEvent.Status = DefaultWrittenOff
	defaultEvent.WrittenOffAt = timePtrNow()
	defaultEvent.ResolutionType = "WRITTEN_OFF"
	defaultEvent.LossAmount = defaultEvent.Amount
	defaultEvent.UpdatedAt = time.Now()

	// 记录核销
	dh.recordWriteOff(ctx, defaultEvent)
}

// sendDefaultNotification 发送违约通知
func (dh *DefaultHandler) sendDefaultNotification(ctx context.Context, defaultEvent *DefaultEvent, gracePeriod int) {
	// 创建通知
	notification := &Notification{
		ID:       generateNotificationID(),
		MemberID: defaultEvent.MemberID,
		Type:     "DEFAULT_NOTIFICATION",
		Title:    fmt.Sprintf("Default Notice - Grace Period %d", gracePeriod),
		Message: fmt.Sprintf("You have been reported in default for %s. Amount: %.2f %s. Grace period %d of %d.",
			defaultEvent.DefaultType, defaultEvent.Amount, defaultEvent.Currency, gracePeriod, dh.config.MaxGracePeriods),
		Priority:  "HIGH",
		Channels:  []string{"EMAIL", "SMS", "IN_APP"},
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	// 发送通知
	sentAt := time.Now()
	notification.Status = "SENT"
	notification.SentAt = &sentAt

	dh.mu.Lock()
	dh.notificationLog[defaultEvent.ID] = append(dh.notificationLog[defaultEvent.ID], notification)
	dh.mu.Unlock()
}

// sendEscalationNotification 发送升级通知
func (dh *DefaultHandler) sendEscalationNotification(ctx context.Context, defaultEvent *DefaultEvent, level string) {
	// 创建升级通知
	notification := &Notification{
		ID:        generateNotificationID(),
		MemberID:  defaultEvent.MemberID,
		Type:      "ESCALATION_NOTIFICATION",
		Title:     fmt.Sprintf("Default Escalation - Level: %s", level),
		Message:   fmt.Sprintf("Your default has been escalated to %s. Immediate action required.", level),
		Priority:  "CRITICAL",
		Channels:  []string{"EMAIL", "SMS", "IN_APP"},
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	// 发送通知
	sentAt := time.Now()
	notification.Status = "SENT"
	notification.SentAt = &sentAt

	dh.mu.Lock()
	dh.notificationLog[defaultEvent.ID] = append(dh.notificationLog[defaultEvent.ID], notification)
	dh.mu.Unlock()
}

// sortPositionsByRisk 按风险排序持仓
func (dh *DefaultHandler) sortPositionsByRisk(positions []*Position) []*Position {
	// 简化实现：按市值排序
	sorted := make([]*Position, len(positions))
	copy(sorted, positions)

	// 按市值降序排序
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].MarketValue < sorted[j].MarketValue {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// calculateLiquidationAmount 计算清算金额
func (dh *DefaultHandler) calculateLiquidationAmount(position *Position, requiredAmount float64) float64 {
	// 计算最大可清算金额
	maxLiquidation := position.MarketValue * dh.config.LiquidationThreshold

	// 取较小值
	if requiredAmount < maxLiquidation {
		return requiredAmount
	}
	return maxLiquidation
}

// liquidatePosition 清算持仓
func (dh *DefaultHandler) liquidatePosition(ctx context.Context, position *Position, amount float64) error {
	if position == nil {
		return fmt.Errorf("position is nil")
	}
	if amount <= 0 {
		return fmt.Errorf("invalid liquidation amount: %.4f", amount)
	}
	if position.MarketValue <= 0 {
		return fmt.Errorf("position has no market value: %s", position.ID)
	}

	ratio := amount / position.MarketValue
	if ratio > 1 {
		ratio = 1
	}
	position.Quantity = position.Quantity * (1 - ratio)
	position.MarketValue -= amount
	if position.MarketValue < 0 {
		position.MarketValue = 0
	}

	return nil
}

// recordDefaultLiquidation 记录违约清算
func (dh *DefaultHandler) recordDefaultLiquidation(ctx context.Context, defaultEvent *DefaultEvent, position *Position, amount float64) {
	// 创建清算记录
	liquidation := &LiquidationRecord{
		ID:              generateLiquidationID(),
		MarginCallID:    defaultEvent.TriggerEventID,
		MemberID:        defaultEvent.MemberID,
		PositionID:      position.ID,
		Symbol:          position.Symbol,
		Amount:          amount,
		Currency:        defaultEvent.Currency,
		LiquidationType: "DEFAULT",
		Status:          "COMPLETED",
		ExecutedAt:      time.Now(),
		CreatedAt:       time.Now(),
	}

	// 保存清算记录
	dh.mu.Lock()
	dh.liquidationLog[defaultEvent.ID] = append(dh.liquidationLog[defaultEvent.ID], liquidation)
	dh.mu.Unlock()
}

// recordWriteOff 记录核销
func (dh *DefaultHandler) recordWriteOff(ctx context.Context, defaultEvent *DefaultEvent) {
	// 创建核销记录
	writeOff := &WriteOffRecord{
		ID:           generateWriteOffID(),
		DefaultID:    defaultEvent.ID,
		MemberID:     defaultEvent.MemberID,
		Amount:       defaultEvent.Amount,
		Currency:     defaultEvent.Currency,
		Reason:       "UNRECOVERABLE",
		Status:       "COMPLETED",
		WrittenOffAt: time.Now(),
		CreatedAt:    time.Now(),
	}

	// 保存核销记录
	dh.mu.Lock()
	dh.writeOffLog[defaultEvent.ID] = writeOff
	dh.mu.Unlock()
}

// assessRecovery 评估追偿
func (dh *DefaultHandler) assessRecovery(ctx context.Context, defaultEvent *DefaultEvent) *RecoveryAssessment {
	// 启发式评估：金额越大、违约类型越复杂，回收概率越低。
	probability := 0.7
	if defaultEvent.Amount > 1_000_000 {
		probability -= 0.25
	} else if defaultEvent.Amount > 100_000 {
		probability -= 0.15
	} else if defaultEvent.Amount > 10_000 {
		probability -= 0.08
	}

	switch defaultEvent.DefaultType {
	case DefaultOperational:
		probability += 0.05
	case DefaultCredit:
		probability -= 0.1
	case DefaultSettlement:
		probability -= 0.05
	}

	if probability < 0.05 {
		probability = 0.05
	}
	if probability > 0.95 {
		probability = 0.95
	}

	assessment := &RecoveryAssessment{
		DefaultID:               defaultEvent.ID,
		MemberID:                defaultEvent.MemberID,
		Amount:                  defaultEvent.Amount,
		RecoveryProbability:     probability,
		EstimatedRecoveryAmount: defaultEvent.Amount * probability,
		EstimatedRecoveryTime:   30 * 24 * time.Hour,
		RiskFactors:             []string{"CREDIT_HISTORY", "COLLATERAL", "LEGAL_ENVIRONMENT"},
		AssessmentDate:          time.Now(),
	}

	return assessment
}

// Data structures

type RecoveryAssessment struct {
	DefaultID               string        `json:"default_id"`
	MemberID                string        `json:"member_id"`
	Amount                  float64       `json:"amount"`
	RecoveryProbability     float64       `json:"recovery_probability"`
	EstimatedRecoveryAmount float64       `json:"estimated_recovery_amount"`
	EstimatedRecoveryTime   time.Duration `json:"estimated_recovery_time"`
	RiskFactors             []string      `json:"risk_factors"`
	AssessmentDate          time.Time     `json:"assessment_date"`
}

type WriteOffRecord struct {
	ID           string    `json:"id"`
	DefaultID    string    `json:"default_id"`
	MemberID     string    `json:"member_id"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	Reason       string    `json:"reason"`
	Status       string    `json:"status"`
	WrittenOffAt time.Time `json:"written_off_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// Repository interfaces

type DefaultRepository interface {
	SaveDefault(ctx context.Context, defaultEvent *DefaultEvent) error
	GetDefault(ctx context.Context, id string) (*DefaultEvent, error)
	GetDefaultsByMember(ctx context.Context, memberID string) ([]*DefaultEvent, error)
	GetActiveDefaults(ctx context.Context) ([]*DefaultEvent, error)
	UpdateDefault(ctx context.Context, defaultEvent *DefaultEvent) error
	DeleteDefault(ctx context.Context, id string) error
}

// Service interfaces

type LegalService interface {
	InitiateLegalAction(ctx context.Context, defaultEvent *DefaultEvent) error
	GetLegalStatus(ctx context.Context, caseID string) (string, error)
	SettleLegalCase(ctx context.Context, caseID string, settlementAmount float64) error
}

type RecoveryService interface {
	InitiateRecovery(ctx context.Context, defaultEvent *DefaultEvent) error
	GetRecoveryStatus(ctx context.Context, recoveryID string) (string, error)
	RecordRecovery(ctx context.Context, recoveryID string, recoveredAmount float64) error
}

// Helper functions

func generateDefaultID() string {
	return fmt.Sprintf("DEFAULT_%d", time.Now().UnixNano())
}

func generateDefaultNo() string {
	return fmt.Sprintf("DEF%d", time.Now().UnixNano())
}

func generateWriteOffID() string {
	return fmt.Sprintf("WRITEOFF_%d", time.Now().UnixNano())
}

func (dh *DefaultHandler) isResolved(ctx context.Context, defaultID string) bool {
	event, err := dh.defaultRepo.GetDefault(ctx, defaultID)
	if err != nil || event == nil {
		return false
	}
	return event.Status == DefaultResolved || event.Status == DefaultWrittenOff
}
