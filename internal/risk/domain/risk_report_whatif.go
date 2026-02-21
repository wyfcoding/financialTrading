//go:build risk_experimental
// +build risk_experimental

package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// RiskReportType 风险报告类型
type RiskReportType string

const (
	ReportDaily      RiskReportType = "DAILY"
	ReportWeekly     RiskReportType = "WEEKLY"
	ReportMonthly    RiskReportType = "MONTHLY"
	ReportAdHoc      RiskReportType = "AD_HOC"
	ReportRegulatory RiskReportType = "REGULATORY"
)

// RiskReport 风险报告
type RiskReport struct {
	ID          string         `json:"id"`
	ReportNo    string         `json:"report_no"`
	ReportType  RiskReportType `json:"report_type"`
	ReportDate  time.Time      `json:"report_date"`
	PeriodStart time.Time      `json:"period_start"`
	PeriodEnd   time.Time      `json:"period_end"`
	GeneratedAt time.Time      `json:"generated_at"`

	// 报告内容
	ExecutiveSummary string            `json:"executive_summary"`
	KeyFindings      []*KeyFinding     `json:"key_findings"`
	RiskMetrics      *RiskMetrics      `json:"risk_metrics"`
	LimitBreaches    []*LimitBreach    `json:"limit_breaches"`
	RiskEvents       []*RiskEvent      `json:"risk_events"`
	Recommendations  []*Recommendation `json:"recommendations"`

	// 格式和分发
	Format         string     `json:"format"` // PDF, HTML, CSV, JSON
	Recipients     []string   `json:"recipients"`
	DeliveryStatus string     `json:"delivery_status"`
	DeliveryTime   *time.Time `json:"delivery_time"`

	// 元数据
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// KeyFinding 关键发现
type KeyFinding struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	Impact      float64 `json:"impact"`
	Confidence  float64 `json:"confidence"`
	Category    string  `json:"category"`
}

// RiskMetrics 风险指标汇总
type RiskMetrics struct {
	PortfolioVaR      float64                       `json:"portfolio_var"`
	PortfolioCVaR     float64                       `json:"portfolio_cvar"`
	MaxDrawdown       float64                       `json:"max_drawdown"`
	Volatility        float64                       `json:"volatility"`
	SharpeRatio       float64                       `json:"sharpe_ratio"`
	SortinoRatio      float64                       `json:"sortino_ratio"`
	Beta              float64                       `json:"beta"`
	CorrelationMatrix map[string]map[string]float64 `json:"correlation_matrix"`
	RiskContributions map[string]float64            `json:"risk_contributions"`
}

// Recommendation 建议
type Recommendation struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Action      string    `json:"action"`
	Owner       string    `json:"owner"`
	DueDate     time.Time `json:"due_date"`
	Status      string    `json:"status"`
}

// ReportGenerator 报告生成器
type ReportGenerator struct {
	riskRepo        RiskRepository
	reportRepo      RiskReportRepository
	templateRepo    ReportTemplateRepository
	deliveryService ReportDeliveryService
	mu              sync.RWMutex
	templates       map[RiskReportType]*ReportTemplate
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator(riskRepo RiskRepository, reportRepo RiskReportRepository,
	templateRepo ReportTemplateRepository, deliveryService ReportDeliveryService) *ReportGenerator {

	return &ReportGenerator{
		riskRepo:        riskRepo,
		reportRepo:      reportRepo,
		templateRepo:    templateRepo,
		deliveryService: deliveryService,
		templates:       make(map[RiskReportType]*ReportTemplate),
	}
}

// Initialize 初始化报告生成器
func (rg *ReportGenerator) Initialize(ctx context.Context) error {
	// 加载报告模板
	templates, err := rg.templateRepo.GetAllTemplates(ctx)
	if err != nil {
		return fmt.Errorf("failed to load report templates: %w", err)
	}

	rg.mu.Lock()
	for _, template := range templates {
		rg.templates[template.ReportType] = template
	}
	rg.mu.Unlock()

	return nil
}

// GenerateReport 生成报告
func (rg *ReportGenerator) GenerateReport(ctx context.Context, reportType RiskReportType,
	portfolioID string, periodStart, periodEnd time.Time) (*RiskReport, error) {

	// 获取报告模板
	template, err := rg.getTemplate(reportType)
	if err != nil {
		return nil, fmt.Errorf("failed to get report template: %w", err)
	}

	// 收集报告数据
	reportData, err := rg.collectReportData(ctx, portfolioID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to collect report data: %w", err)
	}

	// 生成报告内容
	report, err := rg.generateReportContent(template, reportData, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report content: %w", err)
	}

	// 保存报告
	err = rg.reportRepo.SaveReport(ctx, report)
	if err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	// 分发报告
	go rg.deliverReport(ctx, report)

	return report, nil
}

// collectReportData 收集报告数据
func (rg *ReportGenerator) collectReportData(ctx context.Context, portfolioID string,
	periodStart, periodEnd time.Time) (*ReportData, error) {

	data := &ReportData{
		PortfolioID: portfolioID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		CollectedAt: time.Now(),
	}

	// 收集风险评估
	assessments, err := rg.riskRepo.GetAssessmentsByPortfolio(ctx, portfolioID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk assessments: %w", err)
	}
	data.RiskAssessments = assessments

	// 收集风险事件
	events, err := rg.riskRepo.GetEventsByPortfolio(ctx, portfolioID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk events: %w", err)
	}
	data.RiskEvents = events

	// 收集限额突破
	data.LimitBreaches = rg.extractLimitBreaches(events)

	// 计算风险指标
	metrics, err := rg.calculateRiskMetrics(assessments)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate risk metrics: %w", err)
	}
	data.RiskMetrics = metrics

	// 识别关键发现
	data.KeyFindings = rg.identifyKeyFindings(assessments, events)

	// 生成建议
	data.Recommendations = rg.generateRecommendations(data.KeyFindings)

	return data, nil
}

// calculateRiskMetrics 计算风险指标
func (rg *ReportGenerator) calculateRiskMetrics(assessments []*RiskAssessment) (*RiskMetrics, error) {
	if len(assessments) == 0 {
		return &RiskMetrics{}, nil
	}

	metrics := &RiskMetrics{
		CorrelationMatrix: make(map[string]map[string]float64),
		RiskContributions: make(map[string]float64),
	}

	// 使用最新评估
	latest := assessments[len(assessments)-1]

	metrics.PortfolioVaR = latest.VaR
	metrics.PortfolioCVaR = latest.CVaR

	// 计算其他指标
	var varSeries []float64
	var totalVar float64
	var downsideSquares float64
	var maxVar float64
	for _, a := range assessments {
		varSeries = append(varSeries, a.VaR)
		totalVar += a.VaR
		if a.VaR > maxVar {
			maxVar = a.VaR
		}
		if a.VaR < 0 {
			downsideSquares += a.VaR * a.VaR
		}
	}

	meanVar := totalVar / float64(len(varSeries))
	var variance float64
	for _, v := range varSeries {
		variance += math.Pow(v-meanVar, 2)
	}
	variance /= float64(len(varSeries))
	stdDev := math.Sqrt(variance)
	metrics.Volatility = stdDev

	if stdDev > 0 {
		metrics.SharpeRatio = meanVar / stdDev
	}

	downsideDev := math.Sqrt(downsideSquares / float64(len(varSeries)))
	if downsideDev > 0 {
		metrics.SortinoRatio = meanVar / downsideDev
	}

	if maxVar > 0 {
		metrics.MaxDrawdown = (maxVar - latest.VaR) / maxVar
	}

	// 以最新评估的风险贡献和相关矩阵为基准
	metrics.CorrelationMatrix = latest.CorrelationMatrix
	metrics.RiskContributions = latest.RiskContributions

	if latest.ExpectedLoss != 0 {
		metrics.Beta = latest.VaR / latest.ExpectedLoss
	}

	return metrics, nil
}

// identifyKeyFindings 识别关键发现
func (rg *ReportGenerator) identifyKeyFindings(assessments []*RiskAssessment, events []*RiskEvent) []*KeyFinding {
	var findings []*KeyFinding

	// 分析风险评估趋势
	if len(assessments) >= 2 {
		first := assessments[0]
		last := assessments[len(assessments)-1]

		// 检查风险变化
		riskChange := last.VaR - first.VaR
		if math.Abs(riskChange) > first.VaR*0.1 { // 变化超过10%
			finding := &KeyFinding{
				ID:          generateFindingID(),
				Title:       "Significant Risk Change",
				Description: fmt.Sprintf("Portfolio VaR changed by %.2f%%", (riskChange/first.VaR)*100),
				Severity:    rg.determineSeverity(riskChange),
				Impact:      math.Abs(riskChange),
				Confidence:  0.8,
				Category:    "Risk Trend",
			}
			findings = append(findings, finding)
		}
	}

	// 分析风险事件
	for _, event := range events {
		if event.Severity == "HIGH" || event.Severity == "CRITICAL" {
			finding := &KeyFinding{
				ID:          generateFindingID(),
				Title:       fmt.Sprintf("High Severity Risk Event: %s", event.EventType),
				Description: event.Description,
				Severity:    event.Severity,
				Impact:      event.Impact,
				Confidence:  0.9,
				Category:    "Risk Event",
			}
			findings = append(findings, finding)
		}
	}

	// 检查限额突破
	for _, event := range events {
		if event.EventType != "LIMIT_BREACH" {
			continue
		}
		finding := &KeyFinding{
			ID:          generateFindingID(),
			Title:       fmt.Sprintf("Limit Breach: %s", event.Symbol),
			Description: event.Description,
			Severity:    event.Severity,
			Impact:      event.Impact,
			Confidence:  0.85,
			Category:    "Limit Breach",
		}
		findings = append(findings, finding)
	}

	return findings
}

// determineSeverity 确定严重程度
func (rg *ReportGenerator) determineSeverity(change float64) string {
	absChange := math.Abs(change)

	if absChange > 0.3 {
		return "HIGH"
	} else if absChange > 0.15 {
		return "MEDIUM"
	} else {
		return "LOW"
	}
}

func (rg *ReportGenerator) extractLimitBreaches(events []*RiskEvent) []*LimitBreach {
	var breaches []*LimitBreach
	for _, event := range events {
		if event == nil || event.EventType != "LIMIT_BREACH" {
			continue
		}

		breach := &LimitBreach{
			LimitID:     event.ID,
			LimitName:   event.Description,
			Metric:      "UNKNOWN",
			Value:       event.Impact,
			Limit:       0,
			Symbol:      event.Symbol,
			Severity:    event.Severity,
			Impact:      event.Impact,
			Probability: event.Probability,
			Metadata:    event.Metadata,
		}

		if breach.Metadata != nil {
			if metric, ok := breach.Metadata["metric"].(string); ok && metric != "" {
				breach.Metric = metric
			}
			if value, ok := breach.Metadata["value"].(float64); ok {
				breach.Value = value
			}
			if limit, ok := breach.Metadata["limit"].(float64); ok {
				breach.Limit = limit
			}
			if name, ok := breach.Metadata["limit_name"].(string); ok && name != "" {
				breach.LimitName = name
			}
		}

		breaches = append(breaches, breach)
	}
	return breaches
}

// generateRecommendations 生成建议
func (rg *ReportGenerator) generateRecommendations(findings []*KeyFinding) []*Recommendation {
	var recommendations []*Recommendation

	for _, finding := range findings {
		if finding.Severity == "HIGH" || finding.Severity == "CRITICAL" {
			recommendation := &Recommendation{
				ID:          generateRecommendationID(),
				Title:       fmt.Sprintf("Address %s", finding.Title),
				Description: fmt.Sprintf("Mitigate the risk identified: %s", finding.Description),
				Priority:    finding.Severity,
				Action:      rg.determineAction(finding),
				Owner:       "Risk Manager",
				DueDate:     time.Now().Add(7 * 24 * time.Hour),
				Status:      "PENDING",
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}

// determineAction 确定行动
func (rg *ReportGenerator) determineAction(finding *KeyFinding) string {
	switch finding.Category {
	case "Risk Trend":
		return "Review portfolio composition and risk limits"
	case "Risk Event":
		return "Investigate root cause and implement controls"
	case "Limit Breach":
		return "Adjust limits or reduce exposure"
	default:
		return "Monitor and review"
	}
}

// generateReportContent 生成报告内容
func (rg *ReportGenerator) generateReportContent(template *ReportTemplate, data *ReportData,
	periodStart, periodEnd time.Time) (*RiskReport, error) {

	report := &RiskReport{
		ID:             generateReportID(),
		ReportNo:       generateReportNo(),
		ReportType:     template.ReportType,
		ReportDate:     time.Now(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		GeneratedAt:    time.Now(),
		Format:         template.DefaultFormat,
		DeliveryStatus: "PENDING",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 生成执行摘要
	report.ExecutiveSummary = rg.generateExecutiveSummary(data)

	// 设置关键发现
	report.KeyFindings = data.KeyFindings

	// 设置风险指标
	report.RiskMetrics = data.RiskMetrics

	// 设置风险事件与限额突破
	report.RiskEvents = data.RiskEvents
	report.LimitBreaches = data.LimitBreaches

	// 设置建议
	report.Recommendations = data.Recommendations

	// 应用模板
	report = rg.applyTemplate(template, report)

	return report, nil
}

// generateExecutiveSummary 生成执行摘要
func (rg *ReportGenerator) generateExecutiveSummary(data *ReportData) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Risk Report for Portfolio %s\n\n", data.PortfolioID))
	summary.WriteString(fmt.Sprintf("Period: %s to %s\n\n",
		data.PeriodStart.Format("2006-01-02"), data.PeriodEnd.Format("2006-01-02")))

	if len(data.KeyFindings) > 0 {
		summary.WriteString("Key Findings:\n")
		for i, finding := range data.KeyFindings {
			if i < 3 { // 只显示最重要的3个发现
				summary.WriteString(fmt.Sprintf("- %s (%s severity)\n", finding.Title, finding.Severity))
			}
		}
		summary.WriteString("\n")
	}

	if data.RiskMetrics != nil {
		summary.WriteString(fmt.Sprintf("Portfolio VaR: $%.2f\n", data.RiskMetrics.PortfolioVaR))
		summary.WriteString(fmt.Sprintf("Portfolio CVaR: $%.2f\n", data.RiskMetrics.PortfolioCVaR))
	}

	summary.WriteString("\nRecommendations:\n")
	for i, rec := range data.Recommendations {
		if i < 3 { // 只显示最重要的3个建议
			summary.WriteString(fmt.Sprintf("- %s (Priority: %s)\n", rec.Title, rec.Priority))
		}
	}

	return summary.String()
}

// applyTemplate 应用模板
func (rg *ReportGenerator) applyTemplate(template *ReportTemplate, report *RiskReport) *RiskReport {
	if template == nil || report == nil {
		return report
	}

	if report.Metadata == nil {
		report.Metadata = make(map[string]interface{})
	}
	report.Metadata["template_name"] = template.TemplateName
	report.Metadata["template_id"] = template.ID

	if template.TemplateText != "" {
		text := template.TemplateText
		text = strings.ReplaceAll(text, "{{report_no}}", report.ReportNo)
		text = strings.ReplaceAll(text, "{{report_type}}", string(report.ReportType))
		var portfolioVaR, portfolioCVaR float64
		if report.RiskMetrics != nil {
			portfolioVaR = report.RiskMetrics.PortfolioVaR
			portfolioCVaR = report.RiskMetrics.PortfolioCVaR
		}
		text = strings.ReplaceAll(text, "{{portfolio_var}}", fmt.Sprintf("%.2f", portfolioVaR))
		text = strings.ReplaceAll(text, "{{portfolio_cvar}}", fmt.Sprintf("%.2f", portfolioCVaR))
		report.ExecutiveSummary = text + "\n\n" + report.ExecutiveSummary
	}

	return report
}

// deliverReport 分发报告
func (rg *ReportGenerator) deliverReport(ctx context.Context, report *RiskReport) {
	// 设置收件人
	recipients := rg.determineRecipients(report.ReportType)
	report.Recipients = recipients

	// 分发报告
	err := rg.deliveryService.DeliverReport(ctx, report)
	if err != nil {
		report.DeliveryStatus = "FAILED"
	} else {
		report.DeliveryStatus = "DELIVERED"
		deliveryTime := time.Now()
		report.DeliveryTime = &deliveryTime
	}

	// 更新报告状态
	report.UpdatedAt = time.Now()
	rg.reportRepo.UpdateReport(ctx, report)
}

// determineRecipients 确定收件人
func (rg *ReportGenerator) determineRecipients(reportType RiskReportType) []string {
	switch reportType {
	case ReportDaily:
		return []string{"risk.manager@example.com", "trading.desk@example.com"}
	case ReportWeekly:
		return []string{"risk.manager@example.com", "cio@example.com", "compliance@example.com"}
	case ReportMonthly:
		return []string{"risk.manager@example.com", "cio@example.com", "ceo@example.com", "board@example.com"}
	case ReportRegulatory:
		return []string{"regulatory.reporting@example.com", "compliance@example.com"}
	default:
		return []string{"risk.manager@example.com"}
	}
}

// getTemplate 获取模板
func (rg *ReportGenerator) getTemplate(reportType RiskReportType) (*ReportTemplate, error) {
	rg.mu.RLock()
	template, exists := rg.templates[reportType]
	rg.mu.RUnlock()

	if !exists {
		// 使用默认模板
		template = &ReportTemplate{
			ReportType:    reportType,
			DefaultFormat: "PDF",
			TemplateText:  "Default template for " + string(reportType),
		}
	}

	return template, nil
}

// WhatIfAnalysis What-if分析
type WhatIfAnalysis struct {
	ID              string                 `json:"id"`
	AnalysisName    string                 `json:"analysis_name"`
	PortfolioID     string                 `json:"portfolio_id"`
	BaseScenario    *RiskAssessment        `json:"base_scenario"`
	WhatIfScenarios []*WhatIfScenario      `json:"what_if_scenarios"`
	Results         []*ScenarioResult      `json:"results"`
	GeneratedAt     time.Time              `json:"generated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// WhatIfScenario What-if场景
type WhatIfScenario struct {
	ID           string                 `json:"id"`
	ScenarioName string                 `json:"scenario_name"`
	Description  string                 `json:"description"`
	Changes      []*PortfolioChange     `json:"changes"`
	Assumptions  map[string]interface{} `json:"assumptions"`
}

// PortfolioChange 投资组合变化
type PortfolioChange struct {
	Symbol     string  `json:"symbol"`
	ChangeType string  `json:"change_type"` // ADD, REMOVE, MODIFY
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
}

// ScenarioResult 场景结果
type ScenarioResult struct {
	ScenarioID              string             `json:"scenario_id"`
	ScenarioName            string             `json:"scenario_name"`
	VaRChange               float64            `json:"var_change"`
	CVaRChange              float64            `json:"cvar_change"`
	RiskContributionChanges map[string]float64 `json:"risk_contribution_changes"`
	ImpactAnalysis          string             `json:"impact_analysis"`
}

// WhatIfAnalyzer What-if分析器
type WhatIfAnalyzer struct {
	riskManager   *RiskManager
	portfolioRepo PortfolioRepository
	mu            sync.RWMutex
}

// NewWhatIfAnalyzer 创建What-if分析器
func NewWhatIfAnalyzer(riskManager *RiskManager, portfolioRepo PortfolioRepository) *WhatIfAnalyzer {
	return &WhatIfAnalyzer{
		riskManager:   riskManager,
		portfolioRepo: portfolioRepo,
	}
}

// Analyze 分析What-if场景
func (wia *WhatIfAnalyzer) Analyze(ctx context.Context, portfolioID string,
	scenarios []*WhatIfScenario) (*WhatIfAnalysis, error) {

	// 获取基础投资组合
	basePortfolio, err := wia.portfolioRepo.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base portfolio: %w", err)
	}

	// 评估基础场景
	baseAssessment, err := wia.riskManager.AssessPortfolioRisk(ctx, portfolioID,
		RiskMarket, MetricVaR, ModelHistoricalSimulation)
	if err != nil {
		return nil, fmt.Errorf("failed to assess base scenario: %w", err)
	}

	analysis := &WhatIfAnalysis{
		ID:              generateAnalysisID(),
		AnalysisName:    fmt.Sprintf("What-if Analysis for %s", portfolioID),
		PortfolioID:     portfolioID,
		BaseScenario:    baseAssessment,
		WhatIfScenarios: scenarios,
		GeneratedAt:     time.Now(),
	}

	// 分析每个场景
	for _, scenario := range scenarios {
		result, err := wia.analyzeScenario(ctx, basePortfolio, baseAssessment, scenario)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze scenario %s: %w", scenario.ScenarioName, err)
		}

		analysis.Results = append(analysis.Results, result)
	}

	return analysis, nil
}

// analyzeScenario 分析单个场景
func (wia *WhatIfAnalyzer) analyzeScenario(ctx context.Context, basePortfolio *Portfolio,
	baseAssessment *RiskAssessment, scenario *WhatIfScenario) (*ScenarioResult, error) {

	// 创建修改后的投资组合
	modifiedPortfolio := wia.applyChanges(basePortfolio, scenario.Changes)

	// 评估修改后的投资组合
	modifiedAssessment, err := wia.riskManager.AssessPortfolioRisk(ctx, modifiedPortfolio.ID,
		RiskMarket, MetricVaR, ModelHistoricalSimulation)
	if err != nil {
		return nil, fmt.Errorf("failed to assess modified portfolio: %w", err)
	}

	// 计算变化
	result := &ScenarioResult{
		ScenarioID:              scenario.ID,
		ScenarioName:            scenario.ScenarioName,
		VaRChange:               modifiedAssessment.VaR - baseAssessment.VaR,
		CVaRChange:              modifiedAssessment.CVaR - baseAssessment.CVaR,
		RiskContributionChanges: make(map[string]float64),
	}

	// 计算风险贡献变化
	result.RiskContributionChanges = wia.calculateRiskContributionChanges(
		baseAssessment, modifiedAssessment)

	// 生成影响分析
	result.ImpactAnalysis = wia.generateImpactAnalysis(result, scenario)

	return result, nil
}

// applyChanges 应用变化
func (wia *WhatIfAnalyzer) applyChanges(basePortfolio *Portfolio, changes []*PortfolioChange) *Portfolio {
	// 深拷贝基础投资组合
	modifiedPortfolio := &Portfolio{
		ID:        fmt.Sprintf("%s_MODIFIED_%d", basePortfolio.ID, time.Now().UnixNano()),
		Name:      fmt.Sprintf("%s (Modified)", basePortfolio.Name),
		Symbols:   make([]string, len(basePortfolio.Symbols)),
		Positions: make([]*Position, len(basePortfolio.Positions)),
	}

	copy(modifiedPortfolio.Symbols, basePortfolio.Symbols)

	for i, pos := range basePortfolio.Positions {
		modifiedPortfolio.Positions[i] = &Position{
			Symbol:      pos.Symbol,
			Quantity:    pos.Quantity,
			Price:       pos.Price,
			MarketValue: pos.MarketValue,
		}
	}

	// 应用变化
	for _, change := range changes {
		wia.applyChange(modifiedPortfolio, change)
	}

	return modifiedPortfolio
}

// applyChange 应用单个变化
func (wia *WhatIfAnalyzer) applyChange(portfolio *Portfolio, change *PortfolioChange) {
	switch change.ChangeType {
	case "ADD":
		// 添加新持仓
		position := &Position{
			Symbol:      change.Symbol,
			Quantity:    change.Quantity,
			Price:       change.Price,
			MarketValue: change.Quantity * change.Price,
		}
		portfolio.Positions = append(portfolio.Positions, position)
		portfolio.Symbols = append(portfolio.Symbols, change.Symbol)

	case "REMOVE":
		// 移除持仓
		var newPositions []*Position
		var newSymbols []string

		for i, pos := range portfolio.Positions {
			if pos.Symbol != change.Symbol {
				newPositions = append(newPositions, portfolio.Positions[i])
				newSymbols = append(newSymbols, portfolio.Symbols[i])
			}
		}

		portfolio.Positions = newPositions
		portfolio.Symbols = newSymbols

	case "MODIFY":
		// 修改持仓
		for _, pos := range portfolio.Positions {
			if pos.Symbol == change.Symbol {
				pos.Quantity = change.Quantity
				pos.Price = change.Price
				pos.MarketValue = change.Quantity * change.Price
				break
			}
		}
	}
}

// calculateRiskContributionChanges 计算风险贡献变化
func (wia *WhatIfAnalyzer) calculateRiskContributionChanges(base, modified *RiskAssessment) map[string]float64 {
	changes := make(map[string]float64)

	// 比较风险贡献
	for symbol, baseContribution := range base.RiskContributions {
		if modifiedContribution, exists := modified.RiskContributions[symbol]; exists {
			changes[symbol] = modifiedContribution - baseContribution
		} else {
			changes[symbol] = -baseContribution // 完全移除
		}
	}

	// 检查新增的贡献
	for symbol, modifiedContribution := range modified.RiskContributions {
		if _, exists := base.RiskContributions[symbol]; !exists {
			changes[symbol] = modifiedContribution // 新增
		}
	}

	return changes
}

// generateImpactAnalysis 生成影响分析
func (wia *WhatIfAnalyzer) generateImpactAnalysis(result *ScenarioResult, scenario *WhatIfScenario) string {
	var analysis strings.Builder

	analysis.WriteString(fmt.Sprintf("Scenario: %s\n", scenario.ScenarioName))
	analysis.WriteString(fmt.Sprintf("Description: %s\n\n", scenario.Description))

	analysis.WriteString("Risk Impact:\n")
	analysis.WriteString(fmt.Sprintf("  VaR Change: $%.2f (%.2f%%)\n",
		result.VaRChange, (result.VaRChange/result.VaRChange)*100))
	analysis.WriteString(fmt.Sprintf("  CVaR Change: $%.2f (%.2f%%)\n\n",
		result.CVaRChange, (result.CVaRChange/result.CVaRChange)*100))

	if len(result.RiskContributionChanges) > 0 {
		analysis.WriteString("Top Risk Contribution Changes:\n")
		// 找出变化最大的3个
		topChanges := wia.getTopChanges(result.RiskContributionChanges, 3)
		for symbol, change := range topChanges {
			analysis.WriteString(fmt.Sprintf("  %s: %.2f%%\n", symbol, change*100))
		}
	}

	// 生成建议
	analysis.WriteString("\nRecommendation: ")
	if result.VaRChange > 0 {
		analysis.WriteString("Consider reducing exposure or implementing hedges.")
	} else {
		analysis.WriteString("Risk reduction achieved. Consider maintaining this configuration.")
	}

	return analysis.String()
}

// getTopChanges 获取最大的变化
func (wia *WhatIfAnalyzer) getTopChanges(changes map[string]float64, n int) map[string]float64 {
	// 简化实现，实际应该排序
	topChanges := make(map[string]float64)

	count := 0
	for symbol, change := range changes {
		if count >= n {
			break
		}
		topChanges[symbol] = change
		count++
	}

	return topChanges
}

// Helper functions

func generateReportID() string {
	return fmt.Sprintf("REPORT_%d", time.Now().UnixNano())
}

func generateReportNo() string {
	return fmt.Sprintf("RPT%d", time.Now().UnixNano())
}

func generateFindingID() string {
	return fmt.Sprintf("FINDING_%d", time.Now().UnixNano())
}

func generateRecommendationID() string {
	return fmt.Sprintf("REC_%d", time.Now().UnixNano())
}

func generateAnalysisID() string {
	return fmt.Sprintf("ANALYSIS_%d", time.Now().UnixNano())
}

// Data structures

type ReportData struct {
	PortfolioID     string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	CollectedAt     time.Time
	RiskAssessments []*RiskAssessment
	RiskEvents      []*RiskEvent
	LimitBreaches   []*LimitBreach
	RiskMetrics     *RiskMetrics
	KeyFindings     []*KeyFinding
	Recommendations []*Recommendation
}

type ReportTemplate struct {
	ID            string         `json:"id"`
	ReportType    RiskReportType `json:"report_type"`
	TemplateName  string         `json:"template_name"`
	TemplateText  string         `json:"template_text"`
	DefaultFormat string         `json:"default_format"`
	Variables     []string       `json:"variables"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// Repository interfaces

type RiskReportRepository interface {
	SaveReport(ctx context.Context, report *RiskReport) error
	GetReport(ctx context.Context, id string) (*RiskReport, error)
	GetReportsByType(ctx context.Context, reportType RiskReportType,
		startDate, endDate time.Time) ([]*RiskReport, error)
	UpdateReport(ctx context.Context, report *RiskReport) error
	DeleteReport(ctx context.Context, id string) error
}

type ReportTemplateRepository interface {
	SaveTemplate(ctx context.Context, template *ReportTemplate) error
	GetTemplate(ctx context.Context, reportType RiskReportType) (*ReportTemplate, error)
	GetAllTemplates(ctx context.Context) ([]*ReportTemplate, error)
	UpdateTemplate(ctx context.Context, template *ReportTemplate) error
	DeleteTemplate(ctx context.Context, id string) error
}

type ReportDeliveryService interface {
	DeliverReport(ctx context.Context, report *RiskReport) error
	GetDeliveryStatus(ctx context.Context, reportID string) (string, error)
}

type PortfolioRepository interface {
	GetPortfolio(ctx context.Context, portfolioID string) (*Portfolio, error)
	SavePortfolio(ctx context.Context, portfolio *Portfolio) error
	UpdatePortfolio(ctx context.Context, portfolio *Portfolio) error
}
