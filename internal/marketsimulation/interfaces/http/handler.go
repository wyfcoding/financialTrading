package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/application"
)

type Handler struct {
	app *application.MarketSimulationApplicationService
}

func NewHandler(r *gin.Engine, app *application.MarketSimulationApplicationService) *Handler {
	h := &Handler{app: app}
	v1 := r.Group("/api/v1/simulation")
	{
		v1.POST("", h.Create)
		v1.POST("/:id/start", h.Start)
		v1.POST("/:id/stop", h.Stop)
		v1.GET("", h.List)
	}
	return h
}

func (h *Handler) Create(c *gin.Context) {
	var cmd application.CreateSimulationCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dto, err := h.app.CreateSimulationConfig(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dto)
}

func (h *Handler) Start(c *gin.Context) {
	id := c.Param("id")
	if err := h.app.StartSimulation(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

func (h *Handler) Stop(c *gin.Context) {
	id := c.Param("id")
	if err := h.app.StopSimulation(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func (h *Handler) List(c *gin.Context) {
	dtos, err := h.app.ListSimulations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dtos)
}
