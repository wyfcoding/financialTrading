package http

import (
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/application"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/response"
)

// HTTP 处理器
// 负责处理与账户相关的 HTTP 请求
type AccountHandler struct {
	accountService *application.AccountService // 账户应用服务
}

// 创建 HTTP 处理器实例
// accountService: 注入的账户应用服务
func NewAccountHandler(accountService *application.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// 注册路由
func (h *AccountHandler) RegisterRoutes(router *gin.RouterGroup) {
	{
		api := router
		api.POST("/accounts", h.CreateAccount)
		api.GET("/accounts/:id", h.GetAccount)
		api.POST("/accounts/:id/deposit", h.Deposit)
		api.POST("/accounts/:id/freeze", h.Freeze)
		api.POST("/accounts/:id/unfreeze", h.Unfreeze)
	}
}

// BalanceActionRequest 资金操作请求
type BalanceActionRequest struct {
	Amount string `json:"amount" binding:"required"`
	Reason string `json:"reason"`
}

// Freeze 冻结余额
func (h *AccountHandler) Freeze(c *gin.Context) {
	accountID := c.Param("id")
	var req BalanceActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.accountService.FreezeBalance(c.Request.Context(), accountID, amount, req.Reason); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"status": "frozen"})
}

// Unfreeze 解冻余额
func (h *AccountHandler) Unfreeze(c *gin.Context) {
	accountID := c.Param("id")
	var req BalanceActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.accountService.UnfreezeBalance(c.Request.Context(), accountID, amount); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"status": "unfrozen"})
}

// CreateAccount 创建账户
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req application.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	account, err := h.accountService.CreateAccount(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to create account", "error", err)
		response.Error(c, err)
		return
	}

	response.Success(c, account)
}

// GetAccount 获取账户
func (h *AccountHandler) GetAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		response.Error(c, nil) // or custom error
		return
	}

	account, err := h.accountService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get account", "account_id", accountID, "error", err)
		response.Error(c, err)
		return
	}

	response.Success(c, account)
}

// DepositRequest 充值请求
type DepositRequest struct {
	Amount string `json:"amount" binding:"required"`
}

// Deposit 充值
func (h *AccountHandler) Deposit(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		response.Error(c, nil)
		return
	}

	var req DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.accountService.Deposit(c.Request.Context(), accountID, amount); err != nil {
		logging.Error(c.Request.Context(), "Failed to deposit", "account_id", accountID, "error", err)
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"status": "success"})
}
