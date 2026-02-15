// Package domain 监管报告领域模型
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ReportType string

const (
	ReportTypeMIFID2_Transaction ReportType = "MIFID2_TRANSACTION"
	ReportTypeEMIR_Trade         ReportType = "EMIR_TRADE"
	ReportTypeMAS_OTC            ReportType = "MAS_OTC"
)

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "PENDING"
	ReportStatusGenerated ReportStatus = "GENERATED"
	ReportStatusSubmitted ReportStatus = "SUBMITTED"
	ReportStatusAccepted  ReportStatus = "ACCEPTED"
	ReportStatusRejected  ReportStatus = "REJECTED"
)

// RegulatoryReport 监管报告聚合根
type RegulatoryReport struct {
	gorm.Model
	ReportID      string       `gorm:"column:report_id;type:varchar(64);uniqueIndex;not null"`
	Date          time.Time    `gorm:"column:date;index;not null"`
	Type          ReportType   `gorm:"column:type;type:varchar(32);not null"`
	Status        ReportStatus `gorm:"column:status;type:varchar(32);not null;default:'PENDING'"`
	RecordCount   int          `gorm:"column:record_count"`
	Content       string       `gorm:"column:content;type:longtext"` // XML/CSV content
	SubmissionID  string       `gorm:"column:submission_id"`         // Regulator ACK ID
	RejectReason  string       `gorm:"column:reject_reason"`
}

func (RegulatoryReport) TableName() string { return "reg_reports" }

type ReportRepository interface {
	Save(ctx context.Context, report *RegulatoryReport) error
	GetByID(ctx context.Context, id string) (*RegulatoryReport, error)
	GetByDateAndType(ctx context.Context, date time.Time, typ ReportType) (*RegulatoryReport, error)
}

// DataProvider 外部数据源接口 (从 Order/Trade 服务获取数据)
type DataProvider interface {
	FetchTrades(ctx context.Context, date time.Time) ([]any, error)
}
