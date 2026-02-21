//go:build clearing_experimental
// +build clearing_experimental

package domain

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MarginCallWorkflow 保证金催缴工作流
type MarginCallWorkflow struct {
	marginRepo          MarginRepository
	notificationService NotificationService
	escalationService   EscalationService
	paymentGateway      PaymentGateway
	mu                  sync.RWMutex
	config              *WorkflowConfig
	workflows           map[string]*WorkflowInstance
	liquidations        map[string][]*LiquidationRecord
}

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	InitialGracePeriod   time.Duration `json:"initial_grace_period"`
	EscalationLevels     int           `json:"escalation_levels"`
	EscalationInterval   time.Duration `json:"escalation_interval"`
	MaxEscalations       int           `json:"max_escalations"`
	AutoLiquidation      bool          `json:"auto_liquidation"`
	LiquidationThreshold float64       `json:"liquidation_threshold"`
	NotificationChannels []string      `json:"notification_channels"`
}

// WorkflowInstance 工作流实例
type WorkflowInstance struct {
	ID           string          `json:"id"`
	MarginCallID string          `json:"margin_call_id"`
	MemberID     string          `json:"member_id"`
	CurrentStep  string          `json:"current_step"`
	Status       string          `json:"status"` // ACTIVE, COMPLETED, FAILED
	Steps        []*WorkflowStep `json:"steps"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	StepName    string                 `json:"step_name"`
	Action      string                 `json:"action"`
	Status      string                 `json:"status"` // PENDING, IN_PROGRESS, COMPLETED, FAILED
	Deadline    time.Time              `json:"deadline"`
	CompletedAt *time.Time             `json:"completed_at"`
	Result      string                 `json:"result"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewMarginCallWorkflow 创建保证金催缴工作流
func NewMarginCallWorkflow(marginRepo MarginRepository,
	notificationService NotificationService,
	escalationService EscalationService,
	paymentGateway PaymentGateway) *MarginCallWorkflow {

	return &MarginCallWorkflow{
		marginRepo:          marginRepo,
		notificationService: notificationService,
		escalationService:   escalationService,
		paymentGateway:      paymentGateway,
		config: &WorkflowConfig{
			InitialGracePeriod:   24 * time.Hour,
			EscalationLevels:     3,
			EscalationInterval:   12 * time.Hour,
			MaxEscalations:       5,
			AutoLiquidation:      true,
			LiquidationThreshold: 0.5,
			NotificationChannels: []string{"EMAIL", "SMS", "IN_APP"},
		},
		workflows:    make(map[string]*WorkflowInstance),
		liquidations: make(map[string][]*LiquidationRecord),
	}
}

// StartWorkflow 启动工作流
func (mcw *MarginCallWorkflow) StartWorkflow(ctx context.Context, marginCall *MarginCall) error {
	// 创建工作流实例
	workflow := &WorkflowInstance{
		ID:           generateWorkflowID(),
		MarginCallID: marginCall.ID,
		MemberID:     marginCall.MemberID,
		CurrentStep:  "INITIAL_NOTIFICATION",
		Status:       "ACTIVE",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 创建工作流步骤
	workflow.Steps = mcw.createWorkflowSteps(marginCall)

	// 保存工作流
	mcw.mu.Lock()
	mcw.workflows[workflow.ID] = workflow
	mcw.mu.Unlock()

	// 执行当前步骤
	err := mcw.executeStep(ctx, workflow, workflow.CurrentStep)
	if err != nil {
		return fmt.Errorf("failed to execute initial step: %w", err)
	}

	return nil
}

// createWorkflowSteps 创建工作流步骤
func (mcw *MarginCallWorkflow) createWorkflowSteps(marginCall *MarginCall) []*WorkflowStep {
	steps := []*WorkflowStep{
		{
			StepName: "INITIAL_NOTIFICATION",
			Action:   "SEND_INITIAL_NOTIFICATION",
			Status:   "PENDING",
			Deadline: time.Now().Add(mcw.config.InitialGracePeriod),
			Metadata: map[string]interface{}{
				"call_type": marginCall.CallType,
				"amount":    marginCall.Amount,
				"currency":  marginCall.Currency,
				"due_date":  marginCall.DueDate,
			},
		},
		{
			StepName: "FOLLOW_UP",
			Action:   "SEND_FOLLOW_UP",
			Status:   "PENDING",
			Deadline: time.Now().Add(mcw.config.InitialGracePeriod + mcw.config.EscalationInterval),
			Metadata: map[string]interface{}{
				"escalation_level": 1,
			},
		},
		{
			StepName: "ESCALATION_1",
			Action:   "ESCALATE_TO_SUPERVISOR",
			Status:   "PENDING",
			Deadline: time.Now().Add(mcw.config.InitialGracePeriod + 2*mcw.config.EscalationInterval),
			Metadata: map[string]interface{}{
				"escalation_level": 2,
			},
		},
		{
			StepName: "ESCALATION_2",
			Action:   "ESCALATE_TO_MANAGEMENT",
			Status:   "PENDING",
			Deadline: time.Now().Add(mcw.config.InitialGracePeriod + 3*mcw.config.EscalationInterval),
			Metadata: map[string]interface{}{
				"escalation_level": 3,
			},
		},
		{
			StepName: "LIQUIDATION_WARNING",
			Action:   "SEND_LIQUIDATION_WARNING",
			Status:   "PENDING",
			Deadline: time.Now().Add(mcw.config.InitialGracePeriod + 4*mcw.config.EscalationInterval),
			Metadata: map[string]interface{}{
				"warning_level": "FINAL",
			},
		},
		{
			StepName: "AUTO_LIQUIDATION",
			Action:   "EXECUTE_LIQUIDATION",
			Status:   "PENDING",
			Deadline: time.Now().Add(mcw.config.InitialGracePeriod + 5*mcw.config.EscalationInterval),
			Metadata: map[string]interface{}{
				"liquidation_type": "AUTO",
			},
		},
	}

	return steps
}

// executeStep 执行步骤
func (mcw *MarginCallWorkflow) executeStep(ctx context.Context, workflow *WorkflowInstance, stepName string) error {
	// 查找步骤
	var step *WorkflowStep
	for _, s := range workflow.Steps {
		if s.StepName == stepName {
			step = s
			break
		}
	}

	if step == nil {
		return fmt.Errorf("step not found: %s", stepName)
	}

	// 更新步骤状态
	step.Status = "IN_PROGRESS"
	workflow.UpdatedAt = time.Now()

	// 执行步骤动作
	var err error
	switch step.Action {
	case "SEND_INITIAL_NOTIFICATION":
		err = mcw.sendInitialNotification(ctx, workflow, step)
	case "SEND_FOLLOW_UP":
		err = mcw.sendFollowUp(ctx, workflow, step)
	case "ESCALATE_TO_SUPERVISOR":
		err = mcw.escalateToSupervisor(ctx, workflow, step)
	case "ESCALATE_TO_MANAGEMENT":
		err = mcw.escalateToManagement(ctx, workflow, step)
	case "SEND_LIQUIDATION_WARNING":
		err = mcw.sendLiquidationWarning(ctx, workflow, step)
	case "EXECUTE_LIQUIDATION":
		err = mcw.executeLiquidation(ctx, workflow, step)
	default:
		err = fmt.Errorf("unknown action: %s", step.Action)
	}

	// 更新步骤结果
	if err != nil {
		step.Status = "FAILED"
		step.Result = err.Error()
	} else {
		step.Status = "COMPLETED"
		step.Result = "SUCCESS"
		completedAt := time.Now()
		step.CompletedAt = &completedAt
	}

	// 保存工作流状态
	mcw.saveWorkflowState(ctx, workflow)

	// 如果步骤失败，触发升级
	if step.Status == "FAILED" {
		mcw.handleStepFailure(ctx, workflow, step)
	}

	// 如果步骤成功，执行下一步
	if step.Status == "COMPLETED" {
		mcw.executeNextStep(ctx, workflow, step)
	}

	return err
}

// sendInitialNotification 发送初始通知
func (mcw *MarginCallWorkflow) sendInitialNotification(ctx context.Context, workflow *WorkflowInstance, step *WorkflowStep) error {
	// 获取保证金催缴
	marginCall, err := mcw.marginRepo.GetMarginCall(ctx, workflow.MarginCallID)
	if err != nil {
		return fmt.Errorf("failed to get margin call: %w", err)
	}

	// 创建通知
	notification := &Notification{
		ID:       generateNotificationID(),
		MemberID: marginCall.MemberID,
		Type:     "MARGIN_CALL",
		Title:    "Margin Call Notification",
		Message: fmt.Sprintf("You have a margin call of %.2f %s due by %s",
			marginCall.Amount, marginCall.Currency, marginCall.DueDate.Format("2006-01-02 15:04")),
		Priority:  "HIGH",
		Channels:  mcw.config.NotificationChannels,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	// 发送通知
	err = mcw.notificationService.SendNotification(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	// 记录通知发送
	step.Metadata["notification_sent"] = true
	step.Metadata["notification_id"] = notification.ID
	step.Metadata["sent_at"] = time.Now()

	return nil
}

// sendFollowUp 发送跟进通知
func (mcw *MarginCallWorkflow) sendFollowUp(ctx context.Context, workflow *WorkflowInstance, step *WorkflowStep) error {
	// 获取保证金催缴
	marginCall, err := mcw.marginRepo.GetMarginCall(ctx, workflow.MarginCallID)
	if err != nil {
		return fmt.Errorf("failed to get margin call: %w", err)
	}

	// 检查是否已支付
	if marginCall.Status == "COMPLETED" {
		step.Result = "ALREADY_PAID"
		return nil
	}

	// 创建跟进通知
	notification := &Notification{
		ID:       generateNotificationID(),
		MemberID: marginCall.MemberID,
		Type:     "MARGIN_CALL_FOLLOW_UP",
		Title:    "Margin Call Follow-up",
		Message: fmt.Sprintf("Reminder: Your margin call of %.2f %s is still outstanding. Please make payment immediately.",
			marginCall.Amount, marginCall.Currency),
		Priority:  "HIGH",
		Channels:  mcw.config.NotificationChannels,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	// 发送通知
	err = mcw.notificationService.SendNotification(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to send follow-up notification: %w", err)
	}

	// 记录跟进
	step.Metadata["follow_up_sent"] = true
	step.Metadata["notification_id"] = notification.ID
	step.Metadata["sent_at"] = time.Now()

	return nil
}

// escalateToSupervisor 升级到主管
func (mcw *MarginCallWorkflow) escalateToSupervisor(ctx context.Context, workflow *WorkflowInstance, step *WorkflowStep) error {
	// 获取保证金催缴
	marginCall, err := mcw.marginRepo.GetMarginCall(ctx, workflow.MarginCallID)
	if err != nil {
		return fmt.Errorf("failed to get margin call: %w", err)
	}

	// 升级到主管
	err = mcw.escalationService.EscalateToSupervisor(ctx, marginCall)
	if err != nil {
		return fmt.Errorf("failed to escalate to supervisor: %w", err)
	}

	// 记录升级
	step.Metadata["escalated_to_supervisor"] = true
	step.Metadata["escalated_at"] = time.Now()

	return nil
}

// escalateToManagement 升级到管理层
func (mcw *MarginCallWorkflow) escalateToManagement(ctx context.Context, workflow *WorkflowInstance, step *WorkflowStep) error {
	// 获取保证金催缴
	marginCall, err := mcw.marginRepo.GetMarginCall(ctx, workflow.MarginCallID)
	if err != nil {
		return fmt.Errorf("failed to get margin call: %w", err)
	}

	// 升级到管理层
	err = mcw.escalationService.EscalateToManagement(ctx, marginCall)
	if err != nil {
		return fmt.Errorf("failed to escalate to management: %w", err)
	}

	// 记录升级
	step.Metadata["escalated_to_management"] = true
	step.Metadata["escalated_at"] = time.Now()

	return nil
}

// sendLiquidationWarning 发送清算警告
func (mcw *MarginCallWorkflow) sendLiquidationWarning(ctx context.Context, workflow *WorkflowInstance, step *WorkflowStep) error {
	// 获取保证金催缴
	marginCall, err := mcw.marginRepo.GetMarginCall(ctx, workflow.MarginCallID)
	if err != nil {
		return fmt.Errorf("failed to get margin call: %w", err)
	}

	// 创建清算警告通知
	notification := &Notification{
		ID:       generateNotificationID(),
		MemberID: marginCall.MemberID,
		Type:     "LIQUIDATION_WARNING",
		Title:    "Final Warning: Pending Liquidation",
		Message: fmt.Sprintf("URGENT: Your margin call of %.2f %s is overdue. If not paid immediately, positions will be liquidated.",
			marginCall.Amount, marginCall.Currency),
		Priority:  "CRITICAL",
		Channels:  mcw.config.NotificationChannels,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	// 发送通知
	err = mcw.notificationService.SendNotification(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to send liquidation warning: %w", err)
	}

	// 记录警告
	step.Metadata["liquidation_warning_sent"] = true
	step.Metadata["notification_id"] = notification.ID
	step.Metadata["sent_at"] = time.Now()

	return nil
}

// executeLiquidation 执行清算
func (mcw *MarginCallWorkflow) executeLiquidation(ctx context.Context, workflow *WorkflowInstance, step *WorkflowStep) error {
	// 检查是否启用自动清算
	if !mcw.config.AutoLiquidation {
		step.Result = "AUTO_LIQUIDATION_DISABLED"
		return nil
	}

	// 获取保证金催缴
	marginCall, err := mcw.marginRepo.GetMarginCall(ctx, workflow.MarginCallID)
	if err != nil {
		return fmt.Errorf("failed to get margin call: %w", err)
	}

	// 检查是否已支付
	if marginCall.Status == "COMPLETED" {
		step.Result = "ALREADY_PAID"
		return nil
	}

	// 执行清算
	err = mcw.executeAutoLiquidation(ctx, marginCall)
	if err != nil {
		return fmt.Errorf("failed to execute auto liquidation: %w", err)
	}

	// 记录清算
	step.Metadata["liquidation_executed"] = true
	step.Metadata["executed_at"] = time.Now()

	return nil
}

// executeAutoLiquidation 执行自动清算
func (mcw *MarginCallWorkflow) executeAutoLiquidation(ctx context.Context, marginCall *MarginCall) error {
	// 获取会员持仓
	positions, err := mcw.marginRepo.GetMemberPositions(ctx, marginCall.MemberID)
	if err != nil {
		return fmt.Errorf("failed to get member positions: %w", err)
	}

	// 计算需要清算的金额
	requiredAmount := marginCall.OutstandingAmount

	// 按风险权重排序持仓
	sortedPositions := mcw.sortPositionsByRisk(positions)

	// 执行清算
	var liquidatedAmount float64
	for _, position := range sortedPositions {
		if liquidatedAmount >= requiredAmount {
			break
		}

		// 计算本次清算金额
		liquidationAmount := mcw.calculateLiquidationAmount(position, requiredAmount-liquidatedAmount)

		// 执行清算
		err := mcw.liquidatePosition(ctx, position, liquidationAmount)
		if err != nil {
			fmt.Printf("Failed to liquidate position %s: %v\n", position.ID, err)
			continue
		}

		liquidatedAmount += liquidationAmount

		// 记录清算
		mcw.recordLiquidation(ctx, marginCall, position, liquidationAmount)
	}

	// 更新保证金催缴状态
	if liquidatedAmount >= requiredAmount {
		marginCall.Status = "LIQUIDATED"
		marginCall.OutstandingAmount = 0
	} else {
		marginCall.OutstandingAmount -= liquidatedAmount
	}

	marginCall.UpdatedAt = time.Now()

	err = mcw.marginRepo.UpdateMarginCall(ctx, marginCall)
	if err != nil {
		return fmt.Errorf("failed to update margin call: %w", err)
	}

	return nil
}

// sortPositionsByRisk 按风险排序持仓
func (mcw *MarginCallWorkflow) sortPositionsByRisk(positions []*Position) []*Position {
	// 简化实现：按市值排序
	// 实际应该考虑风险权重、流动性等因素

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
func (mcw *MarginCallWorkflow) calculateLiquidationAmount(position *Position, requiredAmount float64) float64 {
	// 计算最大可清算金额
	maxLiquidation := position.MarketValue * mcw.config.LiquidationThreshold

	// 取较小值
	if requiredAmount < maxLiquidation {
		return requiredAmount
	}
	return maxLiquidation
}

// liquidatePosition 清算持仓
func (mcw *MarginCallWorkflow) liquidatePosition(ctx context.Context, position *Position, amount float64) error {
	if position == nil {
		return fmt.Errorf("position is nil")
	}
	if amount <= 0 {
		return fmt.Errorf("invalid liquidation amount: %.4f", amount)
	}
	if position.MarketValue <= 0 {
		return fmt.Errorf("position has no market value: %s", position.ID)
	}

	// 简化实现：按市值比例减少持仓数量与市值。
	liquidationRatio := amount / position.MarketValue
	if liquidationRatio > 1 {
		liquidationRatio = 1
	}
	position.Quantity = position.Quantity * (1 - liquidationRatio)
	position.MarketValue -= amount
	if position.MarketValue < 0 {
		position.MarketValue = 0
	}

	return nil
}

// recordLiquidation 记录清算
func (mcw *MarginCallWorkflow) recordLiquidation(ctx context.Context, marginCall *MarginCall, position *Position, amount float64) {
	// 创建清算记录
	liquidation := &LiquidationRecord{
		ID:              generateLiquidationID(),
		MarginCallID:    marginCall.ID,
		MemberID:        marginCall.MemberID,
		PositionID:      position.ID,
		Symbol:          position.Symbol,
		Amount:          amount,
		Currency:        marginCall.Currency,
		LiquidationType: "AUTO",
		Status:          "COMPLETED",
		ExecutedAt:      time.Now(),
		CreatedAt:       time.Now(),
	}

	// 保存清算记录
	mcw.mu.Lock()
	mcw.liquidations[marginCall.ID] = append(mcw.liquidations[marginCall.ID], liquidation)
	mcw.mu.Unlock()
}

// handleStepFailure 处理步骤失败
func (mcw *MarginCallWorkflow) handleStepFailure(ctx context.Context, workflow *WorkflowInstance, step *WorkflowStep) {
	// 记录失败
	step.Metadata["failure_handled"] = true
	step.Metadata["handled_at"] = time.Now()

	// 根据失败动作触发升级处理
	marginCall, err := mcw.marginRepo.GetMarginCall(ctx, workflow.MarginCallID)
	if err != nil {
		workflow.Status = "FAILED"
		workflow.UpdatedAt = time.Now()
		return
	}

	switch step.Action {
	case "SEND_INITIAL_NOTIFICATION", "SEND_FOLLOW_UP", "SEND_LIQUIDATION_WARNING":
		_ = mcw.escalationService.EscalateToSupervisor(ctx, marginCall)
	case "ESCALATE_TO_SUPERVISOR", "ESCALATE_TO_MANAGEMENT", "EXECUTE_LIQUIDATION":
		_ = mcw.escalationService.EscalateToManagement(ctx, marginCall)
	}

	workflow.Status = "FAILED"
	workflow.UpdatedAt = time.Now()
}

// executeNextStep 执行下一步
func (mcw *MarginCallWorkflow) executeNextStep(ctx context.Context, workflow *WorkflowInstance, step *WorkflowStep) {
	// 确定下一步
	nextStep := mcw.determineNextStep(workflow, step)
	if nextStep == "" {
		// 工作流完成
		workflow.Status = "COMPLETED"
		workflow.UpdatedAt = time.Now()
		return
	}

	// 更新当前步骤
	workflow.CurrentStep = nextStep
	workflow.UpdatedAt = time.Now()

	// 执行下一步
	go func() {
		err := mcw.executeStep(ctx, workflow, nextStep)
		if err != nil {
			fmt.Printf("Failed to execute step %s: %v\n", nextStep, err)
		}
	}()
}

// determineNextStep 确定下一步
func (mcw *MarginCallWorkflow) determineNextStep(workflow *WorkflowInstance, step *WorkflowStep) string {
	// 根据当前步骤确定下一步
	switch step.StepName {
	case "INITIAL_NOTIFICATION":
		return "FOLLOW_UP"
	case "FOLLOW_UP":
		return "ESCALATION_1"
	case "ESCALATION_1":
		return "ESCALATION_2"
	case "ESCALATION_2":
		return "LIQUIDATION_WARNING"
	case "LIQUIDATION_WARNING":
		return "AUTO_LIQUIDATION"
	case "AUTO_LIQUIDATION":
		return "" // 工作流结束
	default:
		return ""
	}
}

// saveWorkflowState 保存工作流状态
func (mcw *MarginCallWorkflow) saveWorkflowState(ctx context.Context, workflow *WorkflowInstance) {
	mcw.mu.Lock()
	mcw.workflows[workflow.ID] = workflow
	mcw.mu.Unlock()
}

// ProcessPayment 处理支付
func (mcw *MarginCallWorkflow) ProcessPayment(ctx context.Context, marginCallID string, paymentAmount float64, paymentMethod string) error {
	// 获取保证金催缴
	marginCall, err := mcw.marginRepo.GetMarginCall(ctx, marginCallID)
	if err != nil {
		return fmt.Errorf("failed to get margin call: %w", err)
	}

	// 处理支付
	err = mcw.paymentGateway.ProcessPayment(ctx, marginCall.MemberID, paymentAmount, marginCall.Currency, paymentMethod)
	if err != nil {
		return fmt.Errorf("payment processing failed: %w", err)
	}

	// 更新保证金催缴
	marginCall.PaidAmount += paymentAmount
	marginCall.OutstandingAmount -= paymentAmount

	if marginCall.OutstandingAmount <= 0 {
		marginCall.Status = "COMPLETED"
		marginCall.ResponseDate = timePtrNow()

		// 完成工作流
		mcw.mu.Lock()
		for _, workflow := range mcw.workflows {
			if workflow.MarginCallID == marginCallID {
				workflow.Status = "COMPLETED"
				workflow.UpdatedAt = time.Now()
			}
		}
		mcw.mu.Unlock()
	}

	marginCall.UpdatedAt = time.Now()

	err = mcw.marginRepo.UpdateMarginCall(ctx, marginCall)
	if err != nil {
		return fmt.Errorf("failed to update margin call: %w", err)
	}

	return nil
}

// Data structures

type Notification struct {
	ID        string     `json:"id"`
	MemberID  string     `json:"member_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Priority  string     `json:"priority"`
	Channels  []string   `json:"channels"`
	Status    string     `json:"status"`
	SentAt    *time.Time `json:"sent_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type LiquidationRecord struct {
	ID              string    `json:"id"`
	MarginCallID    string    `json:"margin_call_id"`
	MemberID        string    `json:"member_id"`
	PositionID      string    `json:"position_id"`
	Symbol          string    `json:"symbol"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	LiquidationType string    `json:"liquidation_type"`
	Status          string    `json:"status"`
	ExecutedAt      time.Time `json:"executed_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// Service interfaces

type NotificationService interface {
	SendNotification(ctx context.Context, notification *Notification) error
	GetNotificationStatus(ctx context.Context, notificationID string) (string, error)
}

type EscalationService interface {
	EscalateToSupervisor(ctx context.Context, marginCall *MarginCall) error
	EscalateToManagement(ctx context.Context, marginCall *MarginCall) error
	GetEscalationLevel(ctx context.Context, marginCallID string) (int, error)
}

type PaymentGateway interface {
	ProcessPayment(ctx context.Context, memberID string, amount float64, currency string, method string) error
	GetPaymentStatus(ctx context.Context, paymentID string) (string, error)
	RefundPayment(ctx context.Context, paymentID string) error
}

// Helper functions

func generateWorkflowID() string {
	return fmt.Sprintf("WORKFLOW_%d", time.Now().UnixNano())
}

func generateNotificationID() string {
	return fmt.Sprintf("NOTIFY_%d", time.Now().UnixNano())
}

func generateLiquidationID() string {
	return fmt.Sprintf("LIQUIDATE_%d", time.Now().UnixNano())
}
