package domain

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrRuleNotFound      = errors.New("rule not found")
	ErrRuleAlreadyExists = errors.New("rule already exists")
	ErrInvalidRuleConfig = errors.New("invalid rule configuration")
)

type AMLRuleType string

const (
	AMLRuleTypeTransaction    AMLRuleType = "TRANSACTION"
	AMLRuleTypeBehavior       AMLRuleType = "BEHAVIOR"
	AMLRuleTypePattern        AMLRuleType = "PATTERN"
	AMLRuleTypeVelocity       AMLRuleType = "VELOCITY"
	AMLRuleTypeThreshold      AMLRuleType = "THRESHOLD"
	AMLRuleTypeSanction       AMLRuleType = "SANCTION"
	AMLRuleTypePEP            AMLRuleType = "PEP"
)

type AMLRuleStatus string

const (
	AMLRuleStatusDraft     AMLRuleStatus = "DRAFT"
	AMLRuleStatusActive    AMLRuleStatus = "ACTIVE"
	AMLRuleStatusInactive  AMLRuleStatus = "INACTIVE"
	AMLRuleStatusArchived  AMLRuleStatus = "ARCHIVED"
)

type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "LOW"
	AlertSeverityMedium   AlertSeverity = "MEDIUM"
	AlertSeverityHigh     AlertSeverity = "HIGH"
	AlertSeverityCritical AlertSeverity = "CRITICAL"
)

type AMLRuleDefinition struct {
	ID          string            `json:"id"`
	RuleCode    string            `json:"rule_code"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        AMLRuleType       `json:"type"`
	Status      AMLRuleStatus     `json:"status"`
	Priority    int               `json:"priority"`
	Conditions  []RuleCondition   `json:"conditions"`
	Actions     []RuleAction      `json:"actions"`
	Parameters  map[string]string `json:"parameters"`
	Version     int               `json:"version"`
	CreatedBy   string            `json:"created_by"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedBy   string            `json:"updated_by"`
	UpdatedAt   time.Time         `json:"updated_at"`
	EffectiveFrom *time.Time      `json:"effective_from,omitempty"`
	EffectiveTo   *time.Time      `json:"effective_to,omitempty"`
}

type RuleCondition struct {
	ID         string            `json:"id"`
	RuleID     string            `json:"rule_id"`
	Field      string            `json:"field"`
	Operator   string            `json:"operator"`
	Value      string            `json:"value"`
	Weight     float64           `json:"weight"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

type RuleAction struct {
	ID          string            `json:"id"`
	RuleID      string            `json:"rule_id"`
	ActionType  string            `json:"action_type"`
	Severity    AlertSeverity     `json:"severity"`
	Description string            `json:"description"`
	NotifyTo    []string          `json:"notify_to"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

type AMLAlertDetail struct {
	ID             string        `json:"id"`
	AlertNo        string        `json:"alert_no"`
	RuleID         string        `json:"rule_id"`
	RuleCode       string        `json:"rule_code"`
	RuleName       string        `json:"rule_name"`
	UserID         string        `json:"user_id"`
	TransactionID  string        `json:"transaction_id,omitempty"`
	AlertType      AMLRuleType   `json:"alert_type"`
	Severity       AlertSeverity `json:"severity"`
	Status         string        `json:"status"`
	RiskScore      float64       `json:"risk_score"`
	TriggerReason  string        `json:"trigger_reason"`
	Details        string        `json:"details"`
	Evidence       []Evidence    `json:"evidence,omitempty"`
	AssignedTo     string        `json:"assigned_to,omitempty"`
	AssignedAt     *time.Time    `json:"assigned_at,omitempty"`
	ReviewedBy     string        `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time    `json:"reviewed_at,omitempty"`
	ReviewNotes    string        `json:"review_notes,omitempty"`
	Disposition    string        `json:"disposition,omitempty"`
	DispositionAt  *time.Time    `json:"disposition_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type Evidence struct {
	ID          string                 `json:"id"`
	AlertID     string                 `json:"alert_id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Source      string                 `json:"source"`
	CreatedAt   time.Time              `json:"created_at"`
}

type SanctionList struct {
	ID           string    `json:"id"`
	ListName     string    `json:"list_name"`
	ListType     string    `json:"list_type"`
	Source       string    `json:"source"`
	LastUpdated  time.Time `json:"last_updated"`
	RecordCount  int       `json:"record_count"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type SanctionEntry struct {
	ID              string    `json:"id"`
	ListID          string    `json:"list_id"`
	EntryType       string    `json:"entry_type"`
	Name            string    `json:"name"`
	NameOriginal    string    `json:"name_original,omitempty"`
	NameVariants    []string  `json:"name_variants,omitempty"`
	DateOfBirth     string    `json:"date_of_birth,omitempty"`
	PlaceOfBirth    string    `json:"place_of_birth,omitempty"`
	Nationality     string    `json:"nationality,omitempty"`
	PassportNo      string    `json:"passport_no,omitempty"`
	IDNumber        string    `json:"id_number,omitempty"`
	Address         string    `json:"address,omitempty"`
	Programs        []string  `json:"programs,omitempty"`
	Aliases         []string  `json:"aliases,omitempty"`
	MatchStrength   float64   `json:"match_strength"`
	LastUpdated     time.Time `json:"last_updated"`
	CreatedAt       time.Time `json:"created_at"`
}

type ScreeningResult struct {
	ID               string            `json:"id"`
	ScreeningID      string            `json:"screening_id"`
	UserID           string            `json:"user_id"`
	ScreeningType    string            `json:"screening_type"`
	InputName        string            `json:"input_name"`
	InputDOB         string            `json:"input_dob,omitempty"`
	InputNationality string            `json:"input_nationality,omitempty"`
	InputIDNumber    string            `json:"input_id_number,omitempty"`
	Matches          []*SanctionMatch  `json:"matches,omitempty"`
	HasMatch         bool              `json:"has_match"`
	HighestScore     float64           `json:"highest_score"`
	Status           string            `json:"status"`
	ScreenedAt       time.Time         `json:"screened_at"`
	CreatedAt        time.Time         `json:"created_at"`
}

type SanctionMatch struct {
	ID              string    `json:"id"`
	ResultID        string    `json:"result_id"`
	EntryID         string    `json:"entry_id"`
	EntryName       string    `json:"entry_name"`
	ListName        string    `json:"list_name"`
	MatchScore      float64   `json:"match_score"`
	NameScore       float64   `json:"name_score"`
	DOBScore        float64   `json:"dob_score"`
	IDScore         float64   `json:"id_score"`
	MatchFields     []string  `json:"match_fields"`
	MatchType       string    `json:"match_type"`
	IsFalsePositive bool      `json:"is_false_positive"`
	ReviewedBy      string    `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes     string    `json:"review_notes,omitempty"`
}

type SARReportDetail struct {
	ID              string        `json:"id"`
	ReportNo        string        `json:"report_no"`
	AlertID         string        `json:"alert_id,omitempty"`
	UserID          string        `json:"user_id"`
	ReportType      string        `json:"report_type"`
	Status          string        `json:"status"`
	FilingDate      *time.Time    `json:"filing_date,omitempty"`
	SubjectInfo     SubjectInfo   `json:"subject_info"`
	SuspiciousActivity SuspiciousActivity `json:"suspicious_activity"`
	Narrative       string        `json:"narrative"`
	PreparedBy      string        `json:"prepared_by"`
	ReviewedBy      string        `json:"reviewed_by,omitempty"`
	ApprovedBy      string        `json:"approved_by,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type SubjectInfo struct {
	Name           string   `json:"name"`
	DOB            string   `json:"dob,omitempty"`
	IDType         string   `json:"id_type,omitempty"`
	IDNumber       string   `json:"id_number,omitempty"`
	Address        string   `json:"address,omitempty"`
	Occupation     string   `json:"occupation,omitempty"`
	AccountNumbers []string `json:"account_numbers,omitempty"`
}

type SuspiciousActivity struct {
	ActivityType     string    `json:"activity_type"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	TotalAmount      float64   `json:"total_amount"`
	Currency         string    `json:"currency"`
	TransactionCount int       `json:"transaction_count"`
	RedFlags         []string  `json:"red_flags"`
}

type CTRReportDetail struct {
	ID              string    `json:"id"`
	ReportNo        string    `json:"report_no"`
	UserID          string    `json:"user_id"`
	TransactionID   string    `json:"transaction_id"`
	TransactionType string    `json:"transaction_type"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	TransactionDate time.Time `json:"transaction_date"`
	Status          string    `json:"status"`
	FilingDate      *time.Time `json:"filing_date,omitempty"`
	PreparedBy      string    `json:"prepared_by"`
	CreatedAt       time.Time `json:"created_at"`
}

type AMLRuleEngine struct {
	rules         map[string]*AMLRuleDefinition
	ruleVersions  map[string][]*AMLRuleDefinition
	alertManager  *AMLAlertManagerEnhanced
	screeningSvc  *SanctionScreeningService
	mu            sync.RWMutex
}

type AMLAlertManagerEnhanced struct {
	alerts     map[string]*AMLAlertDetail
	alertQueue chan *AMLAlertDetail
	mu         sync.RWMutex
}

type SanctionScreeningService struct {
	lists    map[string]*SanctionList
	entries  map[string][]*SanctionEntry
	mu       sync.RWMutex
}

func NewAMLRuleEngine() *AMLRuleEngine {
	return &AMLRuleEngine{
		rules:        make(map[string]*AMLRuleDefinition),
		ruleVersions: make(map[string][]*AMLRuleDefinition),
		alertManager: NewAMLAlertManagerEnhanced(),
		screeningSvc: NewSanctionScreeningService(),
	}
}

func NewAMLAlertManagerEnhanced() *AMLAlertManagerEnhanced {
	return &AMLAlertManagerEnhanced{
		alerts:     make(map[string]*AMLAlertDetail),
		alertQueue: make(chan *AMLAlertDetail, 1000),
	}
}

func NewSanctionScreeningService() *SanctionScreeningService {
	return &SanctionScreeningService{
		lists:   make(map[string]*SanctionList),
		entries: make(map[string][]*SanctionEntry),
	}
}

func (e *AMLRuleEngine) AddRule(rule *AMLRuleDefinition) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.rules[rule.RuleCode]; exists && rule.Status == AMLRuleStatusActive {
		return ErrRuleAlreadyExists
	}

	rule.Version = 1
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	e.rules[rule.RuleCode] = rule
	e.ruleVersions[rule.RuleCode] = append(e.ruleVersions[rule.RuleCode], rule)

	return nil
}

func (e *AMLRuleEngine) UpdateRule(rule *AMLRuleDefinition) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	existing, exists := e.rules[rule.RuleCode]
	if !exists {
		return ErrRuleNotFound
	}

	rule.Version = existing.Version + 1
	rule.UpdatedAt = time.Now()
	e.rules[rule.RuleCode] = rule
	e.ruleVersions[rule.RuleCode] = append(e.ruleVersions[rule.RuleCode], rule)

	return nil
}

func (e *AMLRuleEngine) GetRule(ruleCode string) (*AMLRuleDefinition, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rule, exists := e.rules[ruleCode]
	if !exists {
		return nil, ErrRuleNotFound
	}
	return rule, nil
}

func (e *AMLRuleEngine) GetRuleVersions(ruleCode string) ([]*AMLRuleDefinition, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	versions, exists := e.ruleVersions[ruleCode]
	if !exists {
		return nil, ErrRuleNotFound
	}
	return versions, nil
}

func (e *AMLRuleEngine) EvaluateTransaction(ctx context.Context, txCtx *TransactionContext) (*EvaluationResult, error) {
	result := &EvaluationResult{
		TransactionID:  txCtx.TransactionID,
		UserID:         txCtx.UserID,
		EvaluatedAt:    time.Now(),
		Passed:         true,
		TriggeredRules: make([]*TriggeredRuleResult, 0),
	}

	e.mu.RLock()
	activeRules := make([]*AMLRuleDefinition, 0)
	for _, rule := range e.rules {
		if rule.Status == AMLRuleStatusActive {
			activeRules = append(activeRules, rule)
		}
	}
	e.mu.RUnlock()

	for _, rule := range activeRules {
		triggered, evidence := e.evaluateRule(rule, txCtx)
		if triggered {
			result.Passed = false
			result.TriggeredRules = append(result.TriggeredRules, &TriggeredRuleResult{
				RuleID:    rule.ID,
				RuleCode:  rule.RuleCode,
				RuleName:  rule.Name,
				Severity:  rule.Actions[0].Severity,
				Evidence:  evidence,
			})

			alert := e.createAlert(rule, txCtx, evidence)
			e.alertManager.AddAlert(alert)
		}
	}

	result.RiskScore = e.calculateRiskScore(result.TriggeredRules)
	return result, nil
}

func (e *AMLRuleEngine) evaluateRule(rule *AMLRuleDefinition, txCtx *TransactionContext) (bool, []Evidence) {
	evidence := make([]Evidence, 0)
	triggeredCount := 0

	for _, condition := range rule.Conditions {
		if e.evaluateCondition(condition, txCtx) {
			triggeredCount++
			evidence = append(evidence, Evidence{
				Type:        "CONDITION_MATCH",
				Description: condition.Field + " " + condition.Operator + " " + condition.Value,
				Data:        map[string]interface{}{"condition": condition},
				Source:      "RULE_ENGINE",
				CreatedAt:   time.Now(),
			})
		}
	}

	return triggeredCount == len(rule.Conditions), evidence
}

func (e *AMLRuleEngine) evaluateCondition(condition RuleCondition, txCtx *TransactionContext) bool {
	var value interface{}
	switch condition.Field {
	case "amount":
		value = txCtx.Amount
	case "currency":
		value = txCtx.Currency
	case "transaction_type":
		value = txCtx.TransactionType
	case "country":
		value = txCtx.Country
	case "user_risk_level":
		value = txCtx.UserRiskLevel
	case "daily_transaction_count":
		value = txCtx.DailyTransactionCount
	case "daily_transaction_amount":
		value = txCtx.DailyTransactionAmount
	}

	return e.compareValue(value, condition.Operator, condition.Value)
}

func (e *AMLRuleEngine) compareValue(value interface{}, operator, expected string) bool {
	return true
}

func (e *AMLRuleEngine) createAlert(rule *AMLRuleDefinition, txCtx *TransactionContext, evidence []Evidence) *AMLAlertDetail {
	now := time.Now()
	return &AMLAlertDetail{
		AlertNo:        generateAlertNo(),
		RuleID:         rule.ID,
		RuleCode:       rule.RuleCode,
		RuleName:       rule.Name,
		UserID:         txCtx.UserID,
		TransactionID:  txCtx.TransactionID,
		AlertType:      rule.Type,
		Severity:       rule.Actions[0].Severity,
		Status:         "NEW",
		TriggerReason:  rule.Description,
		Details:        "Transaction triggered AML rule",
		Evidence:       evidence,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (e *AMLRuleEngine) calculateRiskScore(triggeredRules []*TriggeredRuleResult) float64 {
	if len(triggeredRules) == 0 {
		return 0
	}

	var totalScore float64
	for _, tr := range triggeredRules {
		switch tr.Severity {
		case AlertSeverityCritical:
			totalScore += 100
		case AlertSeverityHigh:
			totalScore += 75
		case AlertSeverityMedium:
			totalScore += 50
		case AlertSeverityLow:
			totalScore += 25
		}
	}

	return totalScore / float64(len(triggeredRules))
}

type TransactionContext struct {
	TransactionID         string                 `json:"transaction_id"`
	UserID                string                 `json:"user_id"`
	Amount                float64                `json:"amount"`
	Currency              string                 `json:"currency"`
	TransactionType       string                 `json:"transaction_type"`
	Country               string                 `json:"country"`
	IPAddress             string                 `json:"ip_address"`
	DeviceID              string                 `json:"device_id"`
	UserRiskLevel         string                 `json:"user_risk_level"`
	DailyTransactionCount int                    `json:"daily_transaction_count"`
	DailyTransactionAmount float64               `json:"daily_transaction_amount"`
	TransactionHistory    []TransactionSummary   `json:"transaction_history"`
	Extra                 map[string]interface{} `json:"extra,omitempty"`
}

type TransactionSummary struct {
	TransactionID string    `json:"transaction_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Type          string    `json:"type"`
	Timestamp     time.Time `json:"timestamp"`
}

type EvaluationResult struct {
	TransactionID  string                 `json:"transaction_id"`
	UserID         string                 `json:"user_id"`
	Passed         bool                   `json:"passed"`
	RiskScore      float64                `json:"risk_score"`
	TriggeredRules []*TriggeredRuleResult `json:"triggered_rules,omitempty"`
	EvaluatedAt    time.Time              `json:"evaluated_at"`
}

type TriggeredRuleResult struct {
	RuleID    string        `json:"rule_id"`
	RuleCode  string        `json:"rule_code"`
	RuleName  string        `json:"rule_name"`
	Severity  AlertSeverity `json:"severity"`
	Evidence  []Evidence    `json:"evidence,omitempty"`
}

func (am *AMLAlertManagerEnhanced) AddAlert(alert *AMLAlertDetail) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.alerts[alert.ID] = alert
	select {
	case am.alertQueue <- alert:
	default:
	}
}

func (am *AMLAlertManagerEnhanced) GetAlert(alertID string) (*AMLAlertDetail, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	alert, exists := am.alerts[alertID]
	if !exists {
		return nil, errors.New("alert not found")
	}
	return alert, nil
}

func (am *AMLAlertManagerEnhanced) AssignAlert(alertID, assignee string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	alert, exists := am.alerts[alertID]
	if !exists {
		return errors.New("alert not found")
	}
	now := time.Now()
	alert.AssignedTo = assignee
	alert.AssignedAt = &now
	alert.Status = "OPEN"
	alert.UpdatedAt = now
	return nil
}

func (am *AMLAlertManagerEnhanced) ReviewAlert(alertID, reviewer, notes, disposition string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	alert, exists := am.alerts[alertID]
	if !exists {
		return errors.New("alert not found")
	}
	now := time.Now()
	alert.ReviewedBy = reviewer
	alert.ReviewedAt = &now
	alert.ReviewNotes = notes
	alert.Disposition = disposition
	alert.DispositionAt = &now
	alert.Status = "CLOSED"
	alert.UpdatedAt = now
	return nil
}

func (ss *SanctionScreeningService) AddList(list *SanctionList) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.lists[list.ID] = list
}

func (ss *SanctionScreeningService) AddEntry(listID string, entry *SanctionEntry) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.entries[listID] = append(ss.entries[listID], entry)
}

func (ss *SanctionScreeningService) Screen(ctx context.Context, name, dob, nationality, idNumber string) (*ScreeningResult, error) {
	result := &ScreeningResult{
		ScreeningID:        generateScreeningID(),
		InputName:          name,
		InputDOB:           dob,
		InputNationality:   nationality,
		InputIDNumber:      idNumber,
		Matches:            make([]*SanctionMatch, 0),
		ScreenedAt:         time.Now(),
		CreatedAt:          time.Now(),
	}

	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for listID, entries := range ss.entries {
		list := ss.lists[listID]
		for _, entry := range entries {
			score := ss.calculateMatchScore(name, dob, idNumber, entry)
			if score >= 0.7 {
				result.HasMatch = true
				if score > result.HighestScore {
					result.HighestScore = score
				}
				result.Matches = append(result.Matches, &SanctionMatch{
					EntryID:    entry.ID,
					EntryName:  entry.Name,
					ListName:   list.ListName,
					MatchScore: score,
					NameScore:  ss.calculateNameScore(name, entry),
					DOBScore:   ss.calculateDOBScore(dob, entry),
					IDScore:    ss.calculateIDScore(idNumber, entry),
				})
			}
		}
	}

	if result.HasMatch {
		result.Status = "MATCH_FOUND"
	} else {
		result.Status = "CLEAR"
	}

	return result, nil
}

func (ss *SanctionScreeningService) calculateMatchScore(name, dob, idNumber string, entry *SanctionEntry) float64 {
	nameScore := ss.calculateNameScore(name, entry)
	dobScore := ss.calculateDOBScore(dob, entry)
	idScore := ss.calculateIDScore(idNumber, entry)

	totalScore := nameScore*0.5 + dobScore*0.3 + idScore*0.2
	return totalScore
}

func (ss *SanctionScreeningService) calculateNameScore(name string, entry *SanctionEntry) float64 {
	if name == "" || entry.Name == "" {
		return 0
	}
	if name == entry.Name {
		return 1.0
	}
	for _, alias := range entry.Aliases {
		if name == alias {
			return 0.95
		}
	}
	for _, variant := range entry.NameVariants {
		if name == variant {
			return 0.9
		}
	}
	return 0.3
}

func (ss *SanctionScreeningService) calculateDOBScore(dob string, entry *SanctionEntry) float64 {
	if dob == "" || entry.DateOfBirth == "" {
		return 0
	}
	if dob == entry.DateOfBirth {
		return 1.0
	}
	return 0
}

func (ss *SanctionScreeningService) calculateIDScore(idNumber string, entry *SanctionEntry) float64 {
	if idNumber == "" {
		return 0
	}
	if idNumber == entry.PassportNo || idNumber == entry.IDNumber {
		return 1.0
	}
	return 0
}

func generateAlertNo() string {
	return "AML" + time.Now().Format("20060102150405")
}

func generateScreeningID() string {
	return "SCR" + time.Now().Format("20060102150405")
}

type AMLRuleDefinitionRepository interface {
	Create(rule *AMLRuleDefinition) error
	Update(rule *AMLRuleDefinition) error
	Delete(ruleID string) error
	FindByID(ruleID string) (*AMLRuleDefinition, error)
	FindByCode(ruleCode string) (*AMLRuleDefinition, error)
	FindActive() ([]*AMLRuleDefinition, error)
	FindByType(ruleType AMLRuleType) ([]*AMLRuleDefinition, error)
}

type AMLAlertDetailRepository interface {
	Create(alert *AMLAlertDetail) error
	Update(alert *AMLAlertDetail) error
	FindByID(alertID string) (*AMLAlertDetail, error)
	FindByAlertNo(alertNo string) (*AMLAlertDetail, error)
	FindByUserID(userID string, startTime, endTime *time.Time) ([]*AMLAlertDetail, error)
	FindByStatus(status string, limit int) ([]*AMLAlertDetail, error)
	List(startTime, endTime *time.Time, severity AlertSeverity, page, pageSize int) ([]*AMLAlertDetail, int64, error)
}

type SanctionListRepository interface {
	Create(list *SanctionList) error
	Update(list *SanctionList) error
	FindByID(listID string) (*SanctionList, error)
	FindByName(listName string) (*SanctionList, error)
	FindAll() ([]*SanctionList, error)
}

type SanctionEntryRepository interface {
	Create(entry *SanctionEntry) error
	BatchCreate(entries []*SanctionEntry) error
	FindByID(entryID string) (*SanctionEntry, error)
	FindByListID(listID string) ([]*SanctionEntry, error)
	SearchByName(name string, limit int) ([]*SanctionEntry, error)
}

type ScreeningResultRepository interface {
	Create(result *ScreeningResult) error
	FindByID(resultID string) (*ScreeningResult, error)
	FindByScreeningID(screeningID string) (*ScreeningResult, error)
	FindByUserID(userID string, startTime, endTime *time.Time) ([]*ScreeningResult, error)
}

type SARReportDetailRepository interface {
	Create(report *SARReportDetail) error
	Update(report *SARReportDetail) error
	FindByID(reportID string) (*SARReportDetail, error)
	FindByReportNo(reportNo string) (*SARReportDetail, error)
	FindByUserID(userID string) ([]*SARReportDetail, error)
	List(startTime, endTime *time.Time, status string, page, pageSize int) ([]*SARReportDetail, int64, error)
}

type CTRReportDetailRepository interface {
	Create(report *CTRReportDetail) error
	FindByID(reportID string) (*CTRReportDetail, error)
	FindByReportNo(reportNo string) (*CTRReportDetail, error)
	FindByTransactionID(transactionID string) (*CTRReportDetail, error)
	List(startTime, endTime *time.Time, status string, page, pageSize int) ([]*CTRReportDetail, int64, error)
}
