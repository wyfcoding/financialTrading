// Package interfaces 资金服务接口层
package interfaces

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/treasury/application"
	"github.com/wyfcoding/financialtrading/internal/treasury/domain"
)

// HTTPHandler HTTP 接口处理器
type HTTPHandler struct {
	commandService  *application.CommandService
	queryService    *application.QueryService
	treasuryService *application.TreasuryService
}

// NewHTTPHandler 创建 HTTP 处理器
func NewHTTPHandler(
	commandService *application.CommandService,
	queryService *application.QueryService,
	treasuryService *application.TreasuryService,
) *HTTPHandler {
	return &HTTPHandler{
		commandService:  commandService,
		queryService:    queryService,
		treasuryService: treasuryService,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(r *gin.RouterGroup) {
	treasury := r.Group("/treasury")
	{
		// 账户基础操作
		treasury.POST("/accounts", h.CreateAccount)
		treasury.GET("/accounts/:id/balance", h.GetBalance)
		treasury.GET("/accounts/:id/transactions", h.ListTransactions)

		treasury.POST("/deposit", h.Deposit)
		treasury.POST("/freeze", h.Freeze)
		treasury.POST("/unfreeze", h.Unfreeze)
		treasury.POST("/deduct", h.Deduct)
		treasury.POST("/transfer", h.Transfer)

		// 资金池管理
		treasury.POST("/pools", h.CreateCashPool)
		treasury.GET("/pools/:pool_id/analysis", h.AnalyzeLiquidityGap)
		treasury.POST("/pools/:pool_id/monitor", h.MonitorLiquidity)

		// 调拨指令
		treasury.POST("/transfers/initiate", h.InitiateTransfer)
		treasury.POST("/transfers/:id/approve", h.ApproveTransfer)
	}
}

// CreateAccountRequest 创建账户请求
type CreateAccountRequest struct {
	OwnerID  uint64 `json:"owner_id" binding:"required"`
	Type     int8   `json:"type" binding:"required"`
	Currency int8   `json:"currency" binding:"required"`
}

// CreateAccount 创建账户
func (h *HTTPHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.CreateAccountCommand{
		OwnerID:  req.OwnerID,
		Type:     domain.AccountType(req.Type),
		Currency: domain.Currency(req.Currency),
	}

	accountID, err := h.commandService.CreateAccount(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"account_id": accountID})
}

// DepositRequest 充值请求
type DepositRequest struct {
	AccountID uint64 `json:"account_id" binding:"required"`
	Amount    int64  `json:"amount" binding:"required"`
	RefID     string `json:"ref_id"`
	Source    string `json:"source"`
}

// Deposit 充值
func (h *HTTPHandler) Deposit(c *gin.Context) {
	var req DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.DepositCommand{
		AccountID: req.AccountID,
		Amount:    req.Amount,
		RefID:     req.RefID,
		Source:    req.Source,
	}

	txID, err := h.commandService.Deposit(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction_id": txID})
}

// FreezeRequest 冻结请求
type FreezeRequest struct {
	AccountID uint64 `json:"account_id" binding:"required"`
	Amount    int64  `json:"amount" binding:"required"`
	RefID     string `json:"ref_id"`
	Reason    string `json:"reason"`
}

// Freeze 冻结
func (h *HTTPHandler) Freeze(c *gin.Context) {
	var req FreezeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.FreezeCommand{
		AccountID: req.AccountID,
		Amount:    req.Amount,
		RefID:     req.RefID,
		Reason:    req.Reason,
	}

	txID, err := h.commandService.Freeze(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction_id": txID})
}

// UnfreezeRequest 解冻请求
type UnfreezeRequest struct {
	AccountID uint64 `json:"account_id" binding:"required"`
	Amount    int64  `json:"amount" binding:"required"`
	RefID     string `json:"ref_id"`
	Reason    string `json:"reason"`
}

// Unfreeze 解冻
func (h *HTTPHandler) Unfreeze(c *gin.Context) {
	var req UnfreezeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.UnfreezeCommand{
		AccountID: req.AccountID,
		Amount:    req.Amount,
		RefID:     req.RefID,
		Reason:    req.Reason,
	}

	txID, err := h.commandService.Unfreeze(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction_id": txID})
}

// DeductRequest 扣减请求
type DeductRequest struct {
	AccountID     uint64 `json:"account_id" binding:"required"`
	Amount        int64  `json:"amount" binding:"required"`
	RefID         string `json:"ref_id"`
	Reason        string `json:"reason"`
	UnfreezeFirst bool   `json:"unfreeze_first"`
}

// Deduct 扣减
func (h *HTTPHandler) Deduct(c *gin.Context) {
	var req DeductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.DeductCommand{
		AccountID:     req.AccountID,
		Amount:        req.Amount,
		RefID:         req.RefID,
		Reason:        req.Reason,
		UnfreezeFirst: req.UnfreezeFirst,
	}

	txID, err := h.commandService.Deduct(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction_id": txID})
}

// TransferRequest 转账请求
type TransferRequest struct {
	FromAccountID uint64 `json:"from_account_id" binding:"required"`
	ToAccountID   uint64 `json:"to_account_id" binding:"required"`
	Amount        int64  `json:"amount" binding:"required"`
	RefID         string `json:"ref_id"`
	Remark        string `json:"remark"`
}

// Transfer 转账
func (h *HTTPHandler) Transfer(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.TransferCommand{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
		RefID:         req.RefID,
		Remark:        req.Remark,
	}

	refID, err := h.commandService.Transfer(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction_ref": refID})
}

// GetBalance 获取余额
func (h *HTTPHandler) GetBalance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	account, err := h.queryService.GetBalance(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, account)
}

// ListTransactions 获取流水
func (h *HTTPHandler) ListTransactions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var txType *domain.TransactionType
	if tStr := c.Query("type"); tStr != "" {
		if t, err := strconv.Atoi(tStr); err == nil {
			tt := domain.TransactionType(t)
			txType = &tt
		}
	}

	txs, total, err := h.queryService.ListTransactions(c.Request.Context(), id, txType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txs, "total": total})
}

// New Treasury Handlers

// CreateCashPoolRequest 创建资金池请求
type CreateCashPoolRequest struct {
	Name      string          `json:"name" binding:"required"`
	Currency  string          `json:"currency" binding:"required"`
	MinTarget decimal.Decimal `json:"min_target"`
	MaxTarget decimal.Decimal `json:"max_target"`
}

func (h *HTTPHandler) CreateCashPool(c *gin.Context) {
	var req CreateCashPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pool, err := h.treasuryService.CreateCashPool(c.Request.Context(), req.Name, req.Currency, req.MinTarget, req.MaxTarget)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"pool_id": pool.ID})
}

// AnalyzeLiquidityGap 分析流动性缺口
func (h *HTTPHandler) AnalyzeLiquidityGap(c *gin.Context) {
	poolID, err := strconv.ParseUint(c.Param("pool_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pool id"})
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	gaps, err := h.treasuryService.AnalyzeLiquidityGap(c.Request.Context(), poolID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"gaps": gaps})
}

// MonitorLiquidity 监控流动性
func (h *HTTPHandler) MonitorLiquidity(c *gin.Context) {
	poolID, err := strconv.ParseUint(c.Param("pool_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pool id"})
		return
	}

	err = h.treasuryService.MonitorLiquidity(c.Request.Context(), poolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// InitiateTransferRequest 发起调拨请求
type InitiateTransferRequest struct {
	FromAccountID uint64          `json:"from_account_id" binding:"required"`
	ToAccountID   uint64          `json:"to_account_id" binding:"required"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	Currency      string          `json:"currency" binding:"required"`
	Purpose       string          `json:"purpose" binding:"required"`
}

func (h *HTTPHandler) InitiateTransfer(c *gin.Context) {
	var req InitiateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.treasuryService.InitiateTransfer(c.Request.Context(), req.FromAccountID, req.ToAccountID, req.Amount, req.Currency, req.Purpose)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"instruction_id": id})
}

// ApproveTransferRequest 审批请求
type ApproveTransferRequest struct {
	ApproverID string `json:"approver_id" binding:"required"`
}

func (h *HTTPHandler) ApproveTransfer(c *gin.Context) {
	id := c.Param("id")
	var req ApproveTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.treasuryService.ApproveTransfer(c.Request.Context(), id, req.ApproverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
