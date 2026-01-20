package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	"github.com/wyfcoding/financialtrading/internal/account/application"
)

type AccountHandler struct {
	appService   *application.AccountService
	queryService *application.AccountQueryService
}

func NewAccountHandler(
	appService *application.AccountService,
	queryService *application.AccountQueryService,
) *AccountHandler {
	return &AccountHandler{
		appService:   appService,
		queryService: queryService,
	}
}

func (h *AccountHandler) RegisterRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1/account")
	{
		v1.POST("", h.CreateAccount)
		v1.GET("/:id", h.GetAccount)
		v1.POST("/deposit", h.Deposit)
	}
}

// CreateAccount HTTP Handler
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req accountv1.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.CreateAccountCommand{
		UserID:      req.UserId,
		AccountType: req.AccountType,
		Currency:    req.Currency,
	}

	dto, err := h.appService.CreateAccount(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

// GetAccount HTTP Handler
func (h *AccountHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	dto, err := h.queryService.GetAccount(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

// Deposit HTTP Handler
func (h *AccountHandler) Deposit(c *gin.Context) {
	var req accountv1.DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	amount, _ := decimal.NewFromString(req.Amount)
	cmd := application.DepositCommand{
		AccountID: req.AccountId,
		Amount:    amount,
	}

	if err := h.appService.Deposit(c.Request.Context(), cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
