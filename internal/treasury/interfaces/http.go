// Package interfaces 资金服务接口层
package interfaces

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/treasury/application"
	"github.com/wyfcoding/financialtrading/internal/treasury/domain"
)

// HTTPHandler HTTP 接口处理器
type HTTPHandler struct {
	commandService *application.CommandService
	queryService   *application.QueryService
}

// NewHTTPHandler 创建 HTTP 处理器
func NewHTTPHandler(
	commandService *application.CommandService,
	queryService *application.QueryService,
) *HTTPHandler {
	return &HTTPHandler{
		commandService: commandService,
		queryService:   queryService,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(r *gin.RouterGroup) {
	treasury := r.Group("/treasury")
	{
		treasury.POST("/accounts", h.CreateAccount)
		treasury.GET("/accounts/:id/balance", h.GetBalance)
		treasury.GET("/accounts/:id/transactions", h.ListTransactions)

		treasury.POST("/deposit", h.Deposit)
		treasury.POST("/freeze", h.Freeze)
		treasury.POST("/unfreeze", h.Unfreeze)
		treasury.POST("/deduct", h.Deduct)
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
