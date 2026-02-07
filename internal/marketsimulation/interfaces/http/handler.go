package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/application"
	"github.com/wyfcoding/pkg/response"
)

type Handler struct {
	cmd   *application.MarketSimulationCommandService
	query *application.MarketSimulationQueryService
}

func NewHandler(cmd *application.MarketSimulationCommandService, query *application.MarketSimulationQueryService) *Handler {
	return &Handler{cmd: cmd, query: query}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	v1 := router.Group("/api/v1/simulation")
	{
		v1.POST("", h.Create)
		v1.GET("/:id", h.Get)
		v1.POST("/:id/start", h.Start)
		v1.POST("/:id/stop", h.Stop)
		v1.GET("", h.List)
	}
}

func (h *Handler) Create(c *gin.Context) {
	var cmd application.CreateSimulationCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request data", err.Error())
		return
	}

	dto, err := h.cmd.CreateSimulation(c.Request.Context(), cmd)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	dto, err := h.query.GetSimulation(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	if dto == nil {
		response.ErrorWithStatus(c, http.StatusNotFound, "simulation not found", "")
		return
	}
	response.Success(c, dto)
}

func (h *Handler) Start(c *gin.Context) {
	id := c.Param("id")
	if err := h.cmd.StartSimulation(c.Request.Context(), application.StartSimulationCommand{ScenarioID: id}); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"status": "started"})
}

func (h *Handler) Stop(c *gin.Context) {
	id := c.Param("id")
	if err := h.cmd.StopSimulation(c.Request.Context(), application.StopSimulationCommand{ScenarioID: id}); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"status": "stopped"})
}

func (h *Handler) List(c *gin.Context) {
	dtos, err := h.query.ListSimulations(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, dtos)
}
