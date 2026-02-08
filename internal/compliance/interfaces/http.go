// Package interfaces 合规服务接口层
package interfaces

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/compliance/application"
	"github.com/wyfcoding/financialtrading/internal/compliance/domain"
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
	compliance := r.Group("/compliance")
	{
		compliance.POST("/kyc/submit", h.SubmitKYC)
		compliance.GET("/kyc/status", h.GetKYCStatus)
		compliance.POST("/kyc/review", h.ReviewKYC) // 内部或管理员接口
		compliance.POST("/aml/check", h.CheckAML)
	}
}

// SubmitKYCRequest 提交KYC请求
type SubmitKYCRequest struct {
	UserID         uint64 `json:"user_id" binding:"required"`
	Level          int8   `json:"level" binding:"required"`
	FirstName      string `json:"first_name" binding:"required"`
	LastName       string `json:"last_name" binding:"required"`
	IDNumber       string `json:"id_number" binding:"required"`
	DateOfBirth    string `json:"date_of_birth" binding:"required"`
	Country        string `json:"country" binding:"required"`
	IDCardFrontURL string `json:"id_card_front_url" binding:"required"`
	IDCardBackURL  string `json:"id_card_back_url" binding:"required"`
	FacePhotoURL   string `json:"face_photo_url" binding:"required"`
}

// SubmitKYC 提交KYC
func (h *HTTPHandler) SubmitKYC(c *gin.Context) {
	var req SubmitKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.SubmitKYCCommand{
		UserID:         req.UserID,
		Level:          domain.KYCLevel(req.Level),
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		IDNumber:       req.IDNumber,
		DateOfBirth:    req.DateOfBirth,
		Country:        req.Country,
		IDCardFrontURL: req.IDCardFrontURL,
		IDCardBackURL:  req.IDCardBackURL,
		FacePhotoURL:   req.FacePhotoURL,
	}

	appID, err := h.commandService.SubmitKYC(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"application_id": appID})
}

// GetKYCStatus 获取KYC状态
func (h *HTTPHandler) GetKYCStatus(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Query("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	status, err := h.queryService.GetKYCStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ReviewKYCRequest 审核KYC请求
type ReviewKYCRequest struct {
	ApplicationID string `json:"application_id" binding:"required"`
	Approved      bool   `json:"approved"`
	RejectReason  string `json:"reject_reason"`
	ReviewerID    string `json:"reviewer_id" binding:"required"`
}

// ReviewKYC 审核KYC
func (h *HTTPHandler) ReviewKYC(c *gin.Context) {
	var req ReviewKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.ReviewKYCCommand{
		ApplicationID: req.ApplicationID,
		Approved:      req.Approved,
		RejectReason:  req.RejectReason,
		ReviewerID:    req.ReviewerID,
	}

	if err := h.commandService.ReviewKYC(c.Request.Context(), cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reviewed"})
}

// CheckAMLRequest AML检查请求
type CheckAMLRequest struct {
	UserID  uint64 `json:"user_id" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Country string `json:"country" binding:"required"`
}

// CheckAML AML检查
func (h *HTTPHandler) CheckAML(c *gin.Context) {
	var req CheckAMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.CheckAMLCommand{
		UserID:  req.UserID,
		Name:    req.Name,
		Country: req.Country,
	}

	result, err := h.commandService.CheckAML(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
