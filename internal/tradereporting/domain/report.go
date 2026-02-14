package domain

import (
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrReportNotFound         = errors.New("report not found")
	ErrReportAlreadySent      = errors.New("report already sent")
	ErrReportValidationFailed = errors.New("report validation failed")
)

type ReportType string

const (
	ReportTypeTradeReport      ReportType = "TRADE_REPORT"
	ReportTypeOrderReport      ReportType = "ORDER_REPORT"
	ReportTypePositionReport   ReportType = "POSITION_REPORT"
	ReportTypeMarginReport     ReportType = "MARGIN_REPORT"
	ReportTypeShortSaleReport  ReportType = "SHORT_SALE_REPORT"
	ReportTypeLargeTradeReport ReportType = "LARGE_TRADE_REPORT"
	ReportTypeSuspiciousReport ReportType = "SUSPICIOUS_TRADE_REPORT"
)

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "PENDING"
	ReportStatusValidated ReportStatus = "VALIDATED"
	ReportStatusSubmitted ReportStatus = "SUBMITTED"
	ReportStatusAccepted  ReportStatus = "ACCEPTED"
	ReportStatusRejected  ReportStatus = "REJECTED"
	ReportStatusFailed    ReportStatus = "FAILED"
)

type RegulatoryAuthority string

const (
	RegulatorSEC   RegulatoryAuthority = "SEC"
	RegulatorFINRA RegulatoryAuthority = "FINRA"
	RegulatorCFTC  RegulatoryAuthority = "CFTC"
	RegulatorFCA   RegulatoryAuthority = "FCA"
	RegulatorESMA  RegulatoryAuthority = "ESMA"
	RegulatorCSRC  RegulatoryAuthority = "CSRC"
	RegulatorSFC   RegulatoryAuthority = "SFC"
)

type TradeReport struct {
	ID                 string              `json:"id"`
	ReportNo           string              `json:"report_no"`
	ReportType         ReportType          `json:"report_type"`
	Status             ReportStatus        `json:"status"`
	Regulator          RegulatoryAuthority `json:"regulator"`
	TradeID            string              `json:"trade_id"`
	OrderID            string              `json:"order_id"`
	Symbol             string              `json:"symbol"`
	ISIN               string              `json:"isin"`
	Side               string              `json:"side"`
	Quantity           decimal.Decimal     `json:"quantity"`
	Price              decimal.Decimal     `json:"price"`
	Amount             decimal.Decimal     `json:"amount"`
	Currency           string              `json:"currency"`
	TradeTime          time.Time           `json:"trade_time"`
	SettlementDate     time.Time           `json:"settlement_date"`
	BuyerID            string              `json:"buyer_id"`
	SellerID           string              `json:"seller_id"`
	BrokerID           string              `json:"broker_id"`
	Venue              string              `json:"venue"`
	ExecutionVenue     string              `json:"execution_venue"`
	TransactionType    string              `json:"transaction_type"`
	WaiverIndicator    string              `json:"waiver_indicator"`
	ShortSaleIndicator string              `json:"short_sale_indicator"`
	PriceMultiplier    decimal.Decimal     `json:"price_multiplier"`
	NotionalAmount     decimal.Decimal     `json:"notional_amount"`
	ClearingFlag       bool                `json:"clearing_flag"`
	ValidationResult   *ValidationResult   `json:"validation_result,omitempty"`
	SubmittedAt        *time.Time          `json:"submitted_at,omitempty"`
	AcknowledgedAt     *time.Time          `json:"acknowledged_at,omitempty"`
	RejectedAt         *time.Time          `json:"rejected_at,omitempty"`
	RejectionReason    string              `json:"rejection_reason,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type ValidationResult struct {
	IsValid     bool      `json:"is_valid"`
	Errors      []string  `json:"errors,omitempty"`
	Warnings    []string  `json:"warnings,omitempty"`
	ValidatedAt time.Time `json:"validated_at"`
}

type PositionReport struct {
	ID            string              `json:"id"`
	ReportNo      string              `json:"report_no"`
	ReportType    ReportType          `json:"report_type"`
	Status        ReportStatus        `json:"status"`
	Regulator     RegulatoryAuthority `json:"regulator"`
	ReportDate    time.Time           `json:"report_date"`
	ParticipantID string              `json:"participant_id"`
	Symbol        string              `json:"symbol"`
	ISIN          string              `json:"isin"`
	LongPosition  decimal.Decimal     `json:"long_position"`
	ShortPosition decimal.Decimal     `json:"short_position"`
	NetPosition   decimal.Decimal     `json:"net_position"`
	MarketValue   decimal.Decimal     `json:"market_value"`
	Currency      string              `json:"currency"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

type LargeTradeReport struct {
	ID           string              `json:"id"`
	ReportNo     string              `json:"report_no"`
	ReportType   ReportType          `json:"report_type"`
	Status       ReportStatus        `json:"status"`
	Regulator    RegulatoryAuthority `json:"regulator"`
	TradeID      string              `json:"trade_id"`
	Symbol       string              `json:"symbol"`
	Quantity     decimal.Decimal     `json:"quantity"`
	Amount       decimal.Decimal     `json:"amount"`
	Threshold    decimal.Decimal     `json:"threshold"`
	ThresholdPct decimal.Decimal     `json:"threshold_pct"`
	ReportReason string              `json:"report_reason"`
	CreatedAt    time.Time           `json:"created_at"`
}

type RegulatorySubmission struct {
	ID             string              `json:"id"`
	SubmissionNo   string              `json:"submission_no"`
	ReportType     ReportType          `json:"report_type"`
	Regulator      RegulatoryAuthority `json:"regulator"`
	Status         ReportStatus        `json:"status"`
	ReportCount    int                 `json:"report_count"`
	FileName       string              `json:"file_name"`
	FileContent    []byte              `json:"file_content,omitempty"`
	Checksum       string              `json:"checksum"`
	SubmittedAt    *time.Time          `json:"submitted_at,omitempty"`
	AcknowledgedAt *time.Time          `json:"acknowledged_at,omitempty"`
	ErrorMessage   string              `json:"error_message,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
}

type ReportingRule struct {
	ID              string              `json:"id"`
	RuleCode        string              `json:"rule_code"`
	Name            string              `json:"name"`
	Regulator       RegulatoryAuthority `json:"regulator"`
	ReportType      ReportType          `json:"report_type"`
	ThresholdAmount decimal.Decimal     `json:"threshold_amount"`
	ThresholdPct    decimal.Decimal     `json:"threshold_pct"`
	TimeLimit       int                 `json:"time_limit"`
	Enabled         bool                `json:"enabled"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type TradeReportingEngine struct {
	reports     map[string]*TradeReport
	rules       map[string]*ReportingRule
	submissions map[string]*RegulatorySubmission
	mu          sync.RWMutex
}

func NewTradeReportingEngine() *TradeReportingEngine {
	return &TradeReportingEngine{
		reports:     make(map[string]*TradeReport),
		rules:       make(map[string]*ReportingRule),
		submissions: make(map[string]*RegulatorySubmission),
	}
}

func (e *TradeReportingEngine) AddRule(rule *ReportingRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.RuleCode] = rule
}

func (e *TradeReportingEngine) CreateTradeReport(trade *TradeForReport) (*TradeReport, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	report := &TradeReport{
		ID:                 generateReportID(),
		ReportNo:           "TR" + time.Now().Format("20060102150405"),
		ReportType:         ReportTypeTradeReport,
		Status:             ReportStatusPending,
		Regulator:          trade.Regulator,
		TradeID:            trade.TradeID,
		OrderID:            trade.OrderID,
		Symbol:             trade.Symbol,
		ISIN:               trade.ISIN,
		Side:               trade.Side,
		Quantity:           trade.Quantity,
		Price:              trade.Price,
		Amount:             trade.Amount,
		Currency:           trade.Currency,
		TradeTime:          trade.TradeTime,
		SettlementDate:     trade.SettlementDate,
		BuyerID:            trade.BuyerID,
		SellerID:           trade.SellerID,
		BrokerID:           trade.BrokerID,
		Venue:              trade.Venue,
		ExecutionVenue:     trade.ExecutionVenue,
		TransactionType:    trade.TransactionType,
		ShortSaleIndicator: trade.ShortSaleIndicator,
		PriceMultiplier:    trade.PriceMultiplier,
		NotionalAmount:     trade.NotionalAmount,
		ClearingFlag:       trade.ClearingFlag,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	e.reports[report.ID] = report
	return report, nil
}

func (e *TradeReportingEngine) GetReport(reportID string) (*TradeReport, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	report, exists := e.reports[reportID]
	if !exists {
		return nil, ErrReportNotFound
	}
	return report, nil
}

func (e *TradeReportingEngine) ValidateReport(reportID string) (*ValidationResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, exists := e.reports[reportID]
	if !exists {
		return nil, ErrReportNotFound
	}

	result := &ValidationResult{
		IsValid:     true,
		Errors:      make([]string, 0),
		Warnings:    make([]string, 0),
		ValidatedAt: time.Now(),
	}

	if report.Symbol == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "Symbol is required")
	}

	if report.Quantity.IsZero() {
		result.IsValid = false
		result.Errors = append(result.Errors, "Quantity must be positive")
	}

	if report.Price.IsZero() {
		result.IsValid = false
		result.Errors = append(result.Errors, "Price must be positive")
	}

	if report.TradeTime.IsZero() {
		result.IsValid = false
		result.Errors = append(result.Errors, "Trade time is required")
	}

	report.ValidationResult = result
	if result.IsValid {
		report.Status = ReportStatusValidated
	}
	report.UpdatedAt = time.Now()

	return result, nil
}

func (e *TradeReportingEngine) SubmitReport(reportID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, exists := e.reports[reportID]
	if !exists {
		return ErrReportNotFound
	}

	if report.Status == ReportStatusSubmitted || report.Status == ReportStatusAccepted {
		return ErrReportAlreadySent
	}

	if report.ValidationResult == nil || !report.ValidationResult.IsValid {
		return ErrReportValidationFailed
	}

	now := time.Now()
	report.Status = ReportStatusSubmitted
	report.SubmittedAt = &now
	report.UpdatedAt = now

	return nil
}

func (e *TradeReportingEngine) AcknowledgeReport(reportID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, exists := e.reports[reportID]
	if !exists {
		return ErrReportNotFound
	}

	now := time.Now()
	report.Status = ReportStatusAccepted
	report.AcknowledgedAt = &now
	report.UpdatedAt = now

	return nil
}

func (e *TradeReportingEngine) RejectReport(reportID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, exists := e.reports[reportID]
	if !exists {
		return ErrReportNotFound
	}

	now := time.Now()
	report.Status = ReportStatusRejected
	report.RejectedAt = &now
	report.RejectionReason = reason
	report.UpdatedAt = now

	return nil
}

func (e *TradeReportingEngine) CheckReportingRequirements(trade *TradeForReport) []*ReportingRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	applicableRules := make([]*ReportingRule, 0)

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		if rule.ThresholdAmount.GreaterThan(decimal.Zero) {
			if trade.Amount.GreaterThanOrEqual(rule.ThresholdAmount) {
				applicableRules = append(applicableRules, rule)
			}
		} else {
			applicableRules = append(applicableRules, rule)
		}
	}

	return applicableRules
}

func (e *TradeReportingEngine) GetPendingReports() []*TradeReport {
	e.mu.RLock()
	defer e.mu.RUnlock()

	pending := make([]*TradeReport, 0)
	for _, report := range e.reports {
		if report.Status == ReportStatusPending || report.Status == ReportStatusValidated {
			pending = append(pending, report)
		}
	}
	return pending
}

func (e *TradeReportingEngine) GetReportsByDate(date time.Time) []*TradeReport {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*TradeReport, 0)
	dateStr := date.Format("2006-01-02")
	for _, report := range e.reports {
		if report.TradeTime.Format("2006-01-02") == dateStr {
			result = append(result, report)
		}
	}
	return result
}

func (e *TradeReportingEngine) CreateSubmission(reports []*TradeReport, regulator RegulatoryAuthority) (*RegulatorySubmission, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	submission := &RegulatorySubmission{
		ID:           generateSubmissionID(),
		SubmissionNo: "SUB" + time.Now().Format("20060102150405"),
		ReportType:   ReportTypeTradeReport,
		Regulator:    regulator,
		Status:       ReportStatusPending,
		ReportCount:  len(reports),
		CreatedAt:    time.Now(),
	}

	e.submissions[submission.ID] = submission
	return submission, nil
}

type TradeForReport struct {
	TradeID            string              `json:"trade_id"`
	OrderID            string              `json:"order_id"`
	Symbol             string              `json:"symbol"`
	ISIN               string              `json:"isin"`
	Side               string              `json:"side"`
	Quantity           decimal.Decimal     `json:"quantity"`
	Price              decimal.Decimal     `json:"price"`
	Amount             decimal.Decimal     `json:"amount"`
	Currency           string              `json:"currency"`
	TradeTime          time.Time           `json:"trade_time"`
	SettlementDate     time.Time           `json:"settlement_date"`
	BuyerID            string              `json:"buyer_id"`
	SellerID           string              `json:"seller_id"`
	BrokerID           string              `json:"broker_id"`
	Venue              string              `json:"venue"`
	ExecutionVenue     string              `json:"execution_venue"`
	TransactionType    string              `json:"transaction_type"`
	WaiverIndicator    string              `json:"waiver_indicator"`
	ShortSaleIndicator string              `json:"short_sale_indicator"`
	PriceMultiplier    decimal.Decimal     `json:"price_multiplier"`
	NotionalAmount     decimal.Decimal     `json:"notional_amount"`
	ClearingFlag       bool                `json:"clearing_flag"`
	Regulator          RegulatoryAuthority `json:"regulator"`
}

type TradeReportRepository interface {
	Save(report *TradeReport) error
	FindByID(reportID string) (*TradeReport, error)
	FindByReportNo(reportNo string) (*TradeReport, error)
	FindByTradeID(tradeID string) ([]*TradeReport, error)
	FindByStatus(status ReportStatus) ([]*TradeReport, error)
	FindByDate(date time.Time) ([]*TradeReport, error)
	Update(report *TradeReport) error
}

type PositionReportRepository interface {
	Save(report *PositionReport) error
	FindByID(reportID string) (*PositionReport, error)
	FindByParticipantID(participantID string, date time.Time) ([]*PositionReport, error)
	Update(report *PositionReport) error
}

type RegulatorySubmissionRepository interface {
	Save(submission *RegulatorySubmission) error
	FindByID(submissionID string) (*RegulatorySubmission, error)
	FindByStatus(status ReportStatus) ([]*RegulatorySubmission, error)
	Update(submission *RegulatorySubmission) error
}

type ReportingRuleRepository interface {
	Save(rule *ReportingRule) error
	FindByID(ruleID string) (*ReportingRule, error)
	FindByCode(ruleCode string) (*ReportingRule, error)
	FindByRegulator(regulator RegulatoryAuthority) ([]*ReportingRule, error)
	FindEnabled() ([]*ReportingRule, error)
	Update(rule *ReportingRule) error
}

func generateReportID() string {
	return "RPT" + time.Now().Format("20060102150405")
}

func generateSubmissionID() string {
	return "SUB" + time.Now().Format("20060102150405")
}
