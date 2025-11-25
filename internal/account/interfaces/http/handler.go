package http

import (
	"net/http"

	"github.com/wyfcoding/financialTrading/internal/account/application"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// AccountHandler HTTP 处理器
type AccountHandler struct {
	accountService *application.AccountApplicationService
}

// NewAccountHandler 创建 HTTP 处理器
func NewAccountHandler(accountService *application.AccountApplicationService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// RegisterRoutes 注册路由
func (h *AccountHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/accounts", h.CreateAccount)
		api.GET("/accounts/:id", h.GetAccount)
		api.POST("/accounts/:id/deposit", h.Deposit)
	}
}

// CreateAccount 创建账户
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req application.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.accountService.CreateAccount(c.Request.Context(), &req)
	if err != nil {
		logger.Error(c.Request.Context(), "Failed to create account", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, account)
}

// GetAccount 获取账户
func (h *AccountHandler) GetAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required"})
		return
	}

	account, err := h.accountService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		logger.Error(c.Request.Context(), "Failed to get account", "account_id", accountID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, account)
}

// DepositRequest 充值请求
type DepositRequest struct {
	Amount string `json:"amount" binding:"required"`
}

// Deposit 充值
func (h *AccountHandler) Deposit(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required"})
		return
	}

	var req DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}

	if err := h.accountService.Deposit(c.Request.Context(), accountID, amount); err != nil {
		logger.Error(c.Request.Context(), "Failed to deposit", "account_id", accountID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
