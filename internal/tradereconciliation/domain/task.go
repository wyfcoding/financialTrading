package domain

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type TaskStatus int8
type DiscrepancyStatus int8

const (
	TaskPending   TaskStatus = 1
	TaskRunning   TaskStatus = 2
	TaskCompleted TaskStatus = 3
	TaskFailed    TaskStatus = 4

	DiscrepancyOpen     DiscrepancyStatus = 1
	DiscrepancyResolved DiscrepancyStatus = 2
	DiscrepancyIgnored  DiscrepancyStatus = 3
)

func (s TaskStatus) String() string {
	switch s {
	case TaskPending:
		return "PENDING"
	case TaskRunning:
		return "RUNNING"
	case TaskCompleted:
		return "COMPLETED"
	case TaskFailed:
		return "FAILED"
	}
	return "UNKNOWN"
}

func (s DiscrepancyStatus) String() string {
	switch s {
	case DiscrepancyOpen:
		return "OPEN"
	case DiscrepancyResolved:
		return "RESOLVED"
	case DiscrepancyIgnored:
		return "IGNORED"
	}
	return "UNKNOWN"
}

// ReconciliationTask 对账任务
type ReconciliationTask struct {
	gorm.Model
	TaskID           string     `gorm:"column:task_id;type:varchar(32);unique_index;not null"`
	SourceA          string     `gorm:"column:source_a;type:varchar(50);not null"`
	SourceB          string     `gorm:"column:source_b;type:varchar(50);not null"`
	StartTime        time.Time  `gorm:"column:start_time;not null"`
	EndTime          time.Time  `gorm:"column:end_time;not null"`
	Status           TaskStatus `gorm:"column:status;type:tinyint;not null;default:1"`
	ProcessedCount   int32      `gorm:"column:processed_count;default:0"`
	DiscrepancyCount int32      `gorm:"column:discrepancy_count;default:0"`

	Discrepancies []Discrepancy `gorm:"foreignKey:TaskID;references:TaskID"`
}

// Discrepancy 差异记录
type Discrepancy struct {
	gorm.Model
	DiscrepancyID string            `gorm:"column:discrepancy_id;type:varchar(32);unique_index;not null"`
	TaskID        string            `gorm:"column:task_id;type:varchar(32);index;not null"`
	RecordID      string            `gorm:"column:record_id;type:varchar(64);index;not null"`
	Field         string            `gorm:"column:field;type:varchar(50);not null"`
	ValueA        string            `gorm:"column:value_a;type:varchar(255)"`
	ValueB        string            `gorm:"column:value_b;type:varchar(255)"`
	Status        DiscrepancyStatus `gorm:"column:status;type:tinyint;not null;default:1"`
	Resolution    string            `gorm:"column:resolution;type:varchar(50)"`
	Comment       string            `gorm:"column:comment;type:varchar(255)"`
}

func (ReconciliationTask) TableName() string { return "reconciliation_tasks" }
func (Discrepancy) TableName() string        { return "discrepancies" }

func NewTask(id, sourceA, sourceB string, start, end time.Time) *ReconciliationTask {
	return &ReconciliationTask{
		TaskID:    id,
		SourceA:   sourceA,
		SourceB:   sourceB,
		StartTime: start,
		EndTime:   end,
		Status:    TaskPending,
	}
}

func (t *ReconciliationTask) Start() {
	if t.Status == TaskPending {
		t.Status = TaskRunning
	}
}

func (t *ReconciliationTask) Complete() {
	t.Status = TaskCompleted
}

func (t *ReconciliationTask) Fail() {
	t.Status = TaskFailed
}

func (d *Discrepancy) Resolve(resolution, comment string) error {
	if d.Status != DiscrepancyOpen {
		return errors.New("discrepancy not open")
	}
	d.Status = DiscrepancyResolved
	d.Resolution = resolution
	d.Comment = comment
	return nil
}
