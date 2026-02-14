package domain

import (
	"context"
	"time"
)

type AMLRepository interface {
	SaveAlert(ctx context.Context, alert *AMLAlert) error
	GetAlert(ctx context.Context, id string) (*AMLAlert, error)
	UpdateAlert(ctx context.Context, alert *AMLAlert) error
	ListAlerts(ctx context.Context, status string, page, pageSize int) ([]*AMLAlert, error)

	SaveRiskScore(ctx context.Context, score *UserRiskScore) error
	GetRiskScore(ctx context.Context, userID string) (*UserRiskScore, error)
	
	GetUserRiskProfile(ctx context.Context, userID string) (*UserRiskProfile, error)
	SaveUserRiskProfile(ctx context.Context, profile *UserRiskProfile) error

	SaveRule(ctx context.Context, rule *AMLRule) error
	GetRule(ctx context.Context, ruleID string) (*AMLRule, error)
	UpdateRule(ctx context.Context, rule *AMLRule) error
	DeleteRule(ctx context.Context, ruleID string) error
	ListRules(ctx context.Context, ruleType string, isActive bool, page, pageSize int) ([]*AMLRule, error)
	ListActiveRules(ctx context.Context) ([]*AMLRule, error)

	GetWatchlist(ctx context.Context) ([]*WatchlistEntry, error)
	SaveWatchlistEntry(ctx context.Context, entry *WatchlistEntry) error
	DeleteWatchlistEntry(ctx context.Context, entryID string) error

	SaveAlertHistory(ctx context.Context, history *AlertHistory) error

	SaveSARReport(ctx context.Context, report *SARReport) error
	GetSARReport(ctx context.Context, reportID string) (*SARReport, error)
	ListSARReports(ctx context.Context, status string, page, pageSize int) ([]*SARReport, error)
}

type AMLAlert struct {
	AlertID     string    `gorm:"column:alert_id;type:varchar(64);uniqueIndex;not null"`
	UserID      string    `gorm:"column:user_id;type:varchar(64);index;not null"`
	Type        string    `gorm:"column:type;type:varchar(50);not null"`
	Status      string    `gorm:"column:status;type:varchar(32);not null;default:'NEW'"`
	RiskLevel   string    `gorm:"column:risk_level;type:varchar(20)"`
	Title       string    `gorm:"column:title;type:varchar(255)"`
	Description string    `gorm:"column:description;type:text"`
	AssignedTo  string    `gorm:"column:assigned_to;type:varchar(64)"`
	ResolvedAt  *time.Time `gorm:"column:resolved_at"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (AMLAlert) TableName() string { return "aml_alerts" }

type AlertHistory struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	AlertID     string    `gorm:"column:alert_id;index;not null"`
	Action      string    `gorm:"column:action;type:varchar(50);not null"`
	PerformedBy string    `gorm:"column:performed_by;type:varchar(64)"`
	Notes       string    `gorm:"column:notes;type:text"`
	OldStatus   string    `gorm:"column:old_status;type:varchar(32)"`
	NewStatus   string    `gorm:"column:new_status;type:varchar(32)"`
	Timestamp   time.Time `gorm:"column:timestamp"`
}

func (AlertHistory) TableName() string { return "aml_alert_history" }

type UserRiskScore struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    string    `gorm:"column:user_id;uniqueIndex;not null"`
	Score     float64   `gorm:"column:score;not null"`
	RiskLevel string    `gorm:"column:risk_level;type:varchar(20)"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (UserRiskScore) TableName() string { return "aml_user_risk_scores" }

type UserRiskProfile struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID           string    `gorm:"column:user_id;uniqueIndex;not null"`
	Score            float64   `gorm:"column:score"`
	RiskLevel        string    `gorm:"column:risk_level;type:varchar(20)"`
	KycLevel         string    `gorm:"column:kyc_level;type:varchar(10)"`
	IsPep            bool      `gorm:"column:is_pep;default:false"`
	IsSanctioned     bool      `gorm:"column:is_sanctioned;default:false"`
	HighRiskCountries string   `gorm:"column:high_risk_countries;type:text"`
	AssociatedEntities string  `gorm:"column:associated_entities;type:text"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (UserRiskProfile) TableName() string { return "aml_user_risk_profiles" }

type AMLRule struct {
	RuleID           string    `gorm:"column:rule_id;type:varchar(64);uniqueIndex;not null"`
	Name             string    `gorm:"column:name;type:varchar(255);not null"`
	Description      string    `gorm:"column:description;type:text"`
	Type             string    `gorm:"column:type;type:varchar(32);not null"`
	IsActive         bool      `gorm:"column:is_active;default:true"`
	Priority         int       `gorm:"column:priority;default:0"`
	Condition        string    `gorm:"column:condition;type:text"`
	Actions          string    `gorm:"column:actions;type:text"`
	DefaultRiskLevel string   `gorm:"column:default_risk_level;type:varchar(20)"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
	CreatedBy        string    `gorm:"column:created_by;type:varchar(64)"`
}

func (AMLRule) TableName() string { return "aml_rules" }

type WatchlistEntry struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	EntryID     string    `gorm:"column:entry_id;type:varchar(64);uniqueIndex;not null"`
	Name        string    `gorm:"column:name;type:varchar(255);not null;index"`
	EntityType  string    `gorm:"column:entity_type;type:varchar(32)"`
	Country     string    `gorm:"column:country;type:varchar(10)"`
	ListType    string    `gorm:"column:list_type;type:varchar(20)"`
	Reason      string    `gorm:"column:reason;type:text"`
	AddedAt     time.Time `gorm:"column:added_at"`
	AddedBy     string    `gorm:"column:added_by;type:varchar(64)"`
	ExpiresAt   *time.Time `gorm:"column:expires_at"`
	Metadata    string    `gorm:"column:metadata;type:text"`
}

func (WatchlistEntry) TableName() string { return "aml_watchlist" }

type SARReport struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ReportID        string    `gorm:"column:report_id;type:varchar(64);uniqueIndex;not null"`
	UserID          string    `gorm:"column:user_id;type:varchar(64);index;not null"`
	AlertIDs        string    `gorm:"column:alert_ids;type:text"`
	Narrative       string    `gorm:"column:narrative;type:text"`
	FilingStatus    string    `gorm:"column:filing_status;type:varchar(32)"`
	RegulatoryBody  string    `gorm:"column:regulatory_body;type:varchar(64)"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	FiledAt         *time.Time `gorm:"column:filed_at"`
	FiledBy         string    `gorm:"column:filed_by;type:varchar(64)"`
}

func (SARReport) TableName() string { return "aml_sar_reports" }

type CTRReport struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ReportID        string    `gorm:"column:report_id;type:varchar(64);uniqueIndex;not null"`
	UserID          string    `gorm:"column:user_id;type:varchar(64);not null"`
	TotalAmount     string    `gorm:"column:total_amount;type:varchar(64)"`
	Currency        string    `gorm:"column:currency;type:varchar(10)"`
	TransactionCount int      `gorm:"column:transaction_count"`
	PeriodStart     time.Time `gorm:"column:period_start"`
	PeriodEnd       time.Time `gorm:"column:period_end"`
	FilingStatus    string    `gorm:"column:filing_status;type:varchar(32)"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	FiledAt         *time.Time `gorm:"column:filed_at"`
}

func (CTRReport) TableName() string { return "aml_ctr_reports" }
