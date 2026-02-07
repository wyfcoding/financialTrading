package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/auth/application"
)

type Handler struct {
	cmd   *application.AuthCommandService
	query *application.AuthQueryService
}

func NewHandler(cmd *application.AuthCommandService, query *application.AuthQueryService) *Handler {
	return &Handler{cmd: cmd, query: query}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/v1/auth")
	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
}

func (h *Handler) Register(c *gin.Context) {
	var req struct{ Email, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.cmd.Register(c.Request.Context(), application.RegisterCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user_id": id})
}

func (h *Handler) Login(c *gin.Context) {
	var req struct{ Email, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, exp, err := h.cmd.Login(c.Request.Context(), application.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "type": "Bearer", "expires_at": exp})
}
