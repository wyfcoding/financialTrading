package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/sor/domain"
	"gorm.io/gorm"
)

type SORPlanModel struct {
	gorm.Model
	PlanID          string    `gorm:"column:plan_id;type:varchar(64);uniqueIndex;not null"`
	ParentOrderID   string    `gorm:"column:parent_order_id;type:varchar(64);index"`
	Symbol          string    `gorm:"column:symbol;type:varchar(32);not null;index"`
	Side            int       `gorm:"column:side;not null"`
	TotalQuantity   int64     `gorm:"column:total_quantity;not null"`
	Strategy        int       `gorm:"column:strategy;not null"`
	RoutesJSON      string    `gorm:"column:routes;type:text"`
	AveragePrice    float64   `gorm:"column:average_price"`
	TotalFee        float64   `gorm:"column:total_fee"`
	ExpectedCost    float64   `gorm:"column:expected_cost"`
	MarketImpact    float64   `gorm:"column:market_impact"`
	ConfidenceScore float64   `gorm:"column:confidence_score"`
	GeneratedAt     time.Time `gorm:"column:generated_at"`
	ExpiresAt       time.Time `gorm:"column:expires_at"`
	Status          int       `gorm:"column:status;not null;default:0"`
}

func (SORPlanModel) TableName() string {
	return "sor_plans"
}

type ExecutionReportModel struct {
	gorm.Model
	ReportID    string    `gorm:"column:report_id;type:varchar(64);uniqueIndex;not null"`
	PlanID      string    `gorm:"column:plan_id;type:varchar(64);index;not null"`
	RouteID     string    `gorm:"column:route_id;type:varchar(64);index"`
	VenueID     string    `gorm:"column:venue_id;type:varchar(64)"`
	FilledQty   int64     `gorm:"column:filled_qty"`
	FilledPrice float64   `gorm:"column:filled_price"`
	Fee         float64   `gorm:"column:fee"`
	Slippage    float64   `gorm:"column:slippage"`
	LatencyMs   int64     `gorm:"column:latency_ms"`
	ExecutedAt  time.Time `gorm:"column:executed_at"`
}

func (ExecutionReportModel) TableName() string {
	return "sor_execution_reports"
}

type VenueModel struct {
	gorm.Model
	VenueID      string  `gorm:"column:venue_id;type:varchar(64);uniqueIndex;not null"`
	Name         string  `gorm:"column:name;type:varchar(128);not null"`
	VenueType    int     `gorm:"column:venue_type;not null"`
	Region       string  `gorm:"column:region;type:varchar(32)"`
	Currency     string  `gorm:"column:currency;type:varchar(3)"`
	LatencyMs    int     `gorm:"column:latency_ms"`
	FeeRate      float64 `gorm:"column:fee_rate"`
	IsActive     bool    `gorm:"column:is_active;default:true"`
	IsDarkPool   bool    `gorm:"column:is_dark_pool;default:false"`
	MinOrderSize int64   `gorm:"column:min_order_size"`
	MaxOrderSize int64   `gorm:"column:max_order_size"`
}

func (VenueModel) TableName() string {
	return "sor_venues"
}

type SORPlanRepository struct {
	db *gorm.DB
}

func NewSORPlanRepository(db *gorm.DB) *SORPlanRepository {
	return &SORPlanRepository{db: db}
}

func (r *SORPlanRepository) Save(ctx context.Context, plan *domain.SORPlan) error {
	model := r.toModel(plan)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *SORPlanRepository) Get(ctx context.Context, planID string) (*domain.SORPlan, error) {
	var model SORPlanModel
	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

func (r *SORPlanRepository) GetByParentOrder(ctx context.Context, parentOrderID string) (*domain.SORPlan, error) {
	var model SORPlanModel
	if err := r.db.WithContext(ctx).Where("parent_order_id = ?", parentOrderID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

func (r *SORPlanRepository) Update(ctx context.Context, plan *domain.SORPlan) error {
	model := r.toModel(plan)
	return r.db.WithContext(ctx).Where("plan_id = ?", plan.PlanID).Updates(model).Error
}

func (r *SORPlanRepository) toModel(plan *domain.SORPlan) *SORPlanModel {
	routesJSON, _ := json.Marshal(plan.Routes)
	return &SORPlanModel{
		PlanID:          plan.PlanID,
		ParentOrderID:   plan.ParentOrderID,
		Symbol:          plan.Symbol,
		Side:            int(plan.Side),
		TotalQuantity:   plan.TotalQuantity,
		Strategy:        int(plan.Strategy),
		RoutesJSON:      string(routesJSON),
		AveragePrice:    plan.AveragePrice,
		TotalFee:        plan.TotalFee,
		ExpectedCost:    plan.ExpectedCost,
		MarketImpact:    plan.MarketImpact,
		ConfidenceScore: plan.ConfidenceScore,
		GeneratedAt:     plan.GeneratedAt,
		ExpiresAt:       plan.ExpiresAt,
		Status:          int(plan.Status),
	}
}

func (r *SORPlanRepository) toDomain(model *SORPlanModel) *domain.SORPlan {
	var routes []domain.OrderRoute
	_ = json.Unmarshal([]byte(model.RoutesJSON), &routes)
	return &domain.SORPlan{
		PlanID:          model.PlanID,
		ParentOrderID:   model.ParentOrderID,
		Symbol:          model.Symbol,
		Side:            domain.OrderSide(model.Side),
		TotalQuantity:   model.TotalQuantity,
		Strategy:        domain.RoutingStrategy(model.Strategy),
		Routes:          routes,
		AveragePrice:    model.AveragePrice,
		TotalFee:        model.TotalFee,
		ExpectedCost:    model.ExpectedCost,
		MarketImpact:    model.MarketImpact,
		ConfidenceScore: model.ConfidenceScore,
		GeneratedAt:     model.GeneratedAt,
		ExpiresAt:       model.ExpiresAt,
		Status:          domain.PlanStatus(model.Status),
	}
}

type ExecutionRepository struct {
	db *gorm.DB
}

func NewExecutionRepository(db *gorm.DB) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

func (r *ExecutionRepository) Save(ctx context.Context, report *domain.ExecutionReport) error {
	model := &ExecutionReportModel{
		ReportID:    report.ReportID,
		PlanID:      report.PlanID,
		RouteID:     report.RouteID,
		VenueID:     report.VenueID,
		FilledQty:   report.FilledQty,
		FilledPrice: report.FilledPrice,
		Fee:         report.Fee,
		Slippage:    report.Slippage,
		LatencyMs:   report.Latency.Milliseconds(),
		ExecutedAt:  report.ExecutedAt,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *ExecutionRepository) GetByPlanID(ctx context.Context, planID string) ([]*domain.ExecutionReport, error) {
	var models []*ExecutionReportModel
	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Find(&models).Error; err != nil {
		return nil, err
	}
	
	reports := make([]*domain.ExecutionReport, len(models))
	for i, m := range models {
		reports[i] = &domain.ExecutionReport{
			ReportID:    m.ReportID,
			PlanID:      m.PlanID,
			RouteID:     m.RouteID,
			VenueID:     m.VenueID,
			FilledQty:   m.FilledQty,
			FilledPrice: m.FilledPrice,
			Fee:         m.Fee,
			Slippage:    m.Slippage,
			Latency:     time.Duration(m.LatencyMs) * time.Millisecond,
			ExecutedAt:  m.ExecutedAt,
		}
	}
	return reports, nil
}

func (r *ExecutionRepository) GetByRouteID(ctx context.Context, routeID string) (*domain.ExecutionReport, error) {
	var model ExecutionReportModel
	if err := r.db.WithContext(ctx).Where("route_id = ?", routeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.ExecutionReport{
		ReportID:    model.ReportID,
		PlanID:      model.PlanID,
		RouteID:     model.RouteID,
		VenueID:     model.VenueID,
		FilledQty:   model.FilledQty,
		FilledPrice: model.FilledPrice,
		Fee:         model.Fee,
		Slippage:    model.Slippage,
		Latency:     time.Duration(model.LatencyMs) * time.Millisecond,
		ExecutedAt:  model.ExecutedAt,
	}, nil
}

type VenueRepository struct {
	db *gorm.DB
}

func NewVenueRepository(db *gorm.DB) *VenueRepository {
	return &VenueRepository{db: db}
}

func (r *VenueRepository) GetAll(ctx context.Context) ([]*domain.Venue, error) {
	var models []*VenueModel
	if err := r.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	return r.toDomains(models), nil
}

func (r *VenueRepository) GetByID(ctx context.Context, venueID string) (*domain.Venue, error) {
	var model VenueModel
	if err := r.db.WithContext(ctx).Where("venue_id = ?", venueID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

func (r *VenueRepository) GetActive(ctx context.Context) ([]*domain.Venue, error) {
	var models []*VenueModel
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&models).Error; err != nil {
		return nil, err
	}
	return r.toDomains(models), nil
}

func (r *VenueRepository) toDomain(model *VenueModel) *domain.Venue {
	return &domain.Venue{
		ID:           model.VenueID,
		Name:         model.Name,
		Type:         domain.VenueType(model.VenueType),
		Region:       model.Region,
		Currency:     model.Currency,
		Latency:      time.Duration(model.LatencyMs) * time.Millisecond,
		FeeRate:      model.FeeRate,
		IsActive:     model.IsActive,
		IsDarkPool:   model.IsDarkPool,
		MinOrderSize: model.MinOrderSize,
		MaxOrderSize: model.MaxOrderSize,
	}
}

func (r *VenueRepository) toDomains(models []*VenueModel) []*domain.Venue {
	venues := make([]*domain.Venue, len(models))
	for i, m := range models {
		venues[i] = r.toDomain(m)
	}
	return venues
}
