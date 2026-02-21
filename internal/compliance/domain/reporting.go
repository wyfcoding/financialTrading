// 生成摘要：
// - 从 taxreporting, tradereporting, regulatoryreporting 合并到 compliance 域。
// - 税务报告、交易报告、监管报告统一归入合规子域。
// - 关键实体：TaxReport, TradeReport, RegulatoryReport。
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// 合规报告模块错误定义。
var (
	ErrNoTradingHistory   = errors.New("no trading history found for tax year")
	ErrUnsupportedTaxRule = errors.New("currency/instrument rule not supported for tax")
)

// TaxYear 报税年度。
type TaxYear int

// CapitalGains 资本利得及相关抵扣信息。
type CapitalGains struct {
	ShortTermGains decimal.Decimal `json:"short_term_gains"`
	ShortTermLoss  decimal.Decimal `json:"short_term_loss"`
	LongTermGains  decimal.Decimal `json:"long_term_gains"`
	LongTermLoss   decimal.Decimal `json:"long_term_loss"`
	NetShortTerm   decimal.Decimal `json:"net_short_term"`
	NetLongTerm    decimal.Decimal `json:"net_long_term"`
	TotalNet       decimal.Decimal `json:"total_net"`
}

// TaxIncome 非交易直接利得（股息、利息）。
type TaxIncome struct {
	Dividends decimal.Decimal `json:"dividends"`
	Interest  decimal.Decimal `json:"interest"`
}

// TaxReport 单个用户的税务申报快照。
type TaxReport struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	TaxYear       TaxYear         `json:"tax_year"`
	Gains         CapitalGains    `json:"capital_gains"`
	Income        TaxIncome       `json:"income"`
	CarryoverLoss decimal.Decimal `json:"carryover_loss"`
	FileURI       string          `json:"file_uri"`
	GeneratedAt   time.Time       `json:"generated_at"`
}

// TradeMatch FIFO 买卖配对快照。
type TradeMatch struct {
	Asset      string          `json:"asset"`
	BuyDate    time.Time       `json:"buy_date"`
	SellDate   time.Time       `json:"sell_date"`
	DaysHeld   int             `json:"days_held"`
	Quantity   decimal.Decimal `json:"quantity"`
	CostBasis  decimal.Decimal `json:"cost_basis"`
	Proceeds   decimal.Decimal `json:"proceeds"`
	NetGain    decimal.Decimal `json:"net_gain"`
	IsWashSale bool            `json:"is_wash_sale"`
}

// AssessWashSale 评估是否触发洗售规则。
func (t *TradeMatch) AssessWashSale(replacements []time.Time) {
	for _, repDate := range replacements {
		diff := t.SellDate.Sub(repDate).Hours() / 24
		if diff >= -30 && diff <= 30 && t.NetGain.IsNegative() {
			t.IsWashSale = true
			break
		}
	}
}

// RegulatoryReportStatus 监管报告状态。
type RegulatoryReportStatus string

const (
	ReportPending   RegulatoryReportStatus = "PENDING"
	ReportGenerated RegulatoryReportStatus = "GENERATED"
	ReportSubmitted RegulatoryReportStatus = "SUBMITTED"
	ReportRejected  RegulatoryReportStatus = "REJECTED"
)

// RegulatoryReport 监管报告实体（MiFID2, EMIR 等）。
type RegulatoryReport struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	PeriodStart  time.Time              `json:"period_start"`
	PeriodEnd    time.Time              `json:"period_end"`
	Content      []byte                 `json:"content"`
	Status       RegulatoryReportStatus `json:"status"`
	SubmissionID string                 `json:"submission_id"`
	CreatedAt    time.Time              `json:"created_at"`
}

// TradeReport 交易报告实体。
type TradeReport struct {
	ID            string    `json:"id"`
	TradeID       string    `json:"trade_id"`
	Venue         string    `json:"venue"`
	Instrument    string    `json:"instrument"`
	Quantity      string    `json:"quantity"`
	Price         string    `json:"price"`
	Currency      string    `json:"currency"`
	OccurredAt    time.Time `json:"occurred_at"`
	ReportType    string    `json:"report_type"`
	Status        string    `json:"status"`
	SubmissionRef string    `json:"submission_ref"`
	Payload       []byte    `json:"payload"`
	CreatedAt     time.Time `json:"created_at"`
}

// TaxReportingService 税务报告服务接口。
type TaxReportingService interface {
	GenerateAnnualReport(ctx context.Context, userID string, year TaxYear) (*TaxReport, error)
	DownloadForm(ctx context.Context, reportID string) ([]byte, error)
}

// ReportGenerator 监管报告生成器接口。
type ReportGenerator interface {
	GenerateMiFID2TradeReport(trades []interface{}) (*RegulatoryReport, error)
}
