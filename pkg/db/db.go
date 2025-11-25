// Package db 提供 GORM 初始化、连接池/重连策略、指标暴露、事务助手、分库分表辅助工具
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pkgLogger "github.com/fynnwu/FinancialTrading/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// Config 数据库配置
type Config struct {
	Driver             string
	DSN                string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetime    int
	LogEnabled         bool
	SlowQueryThreshold int
}

// DB 数据库实例包装
type DB struct {
	*gorm.DB
	config Config
}

// Init 初始化数据库连接
func Init(cfg Config) (*DB, error) {
	var dialector gorm.Dialector

	// 根据驱动类型选择方言
	switch cfg.Driver {
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	// 创建 GORM 日志记录器
	gormLogger := NewGormLogger(cfg.LogEnabled, time.Duration(cfg.SlowQueryThreshold)*time.Millisecond)

	// 打开数据库连接
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层 SQL 数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// 测试连接
	if err := sqlDB.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pkgLogger.Info(context.Background(), "Database connected successfully", "driver", cfg.Driver)

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

// Close 关闭数据库连接
func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// WithTx 在事务中执行函数，支持自动回滚和提交
func (d *DB) WithTx(ctx context.Context, fn func(*gorm.DB) error) error {
	tx := d.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

// WithTxIsolation 在指定隔离级别的事务中执行函数
func (d *DB) WithTxIsolation(ctx context.Context, isolation string, fn func(*gorm.DB) error) error {
	tx := d.DB.WithContext(ctx).Begin(&sql.TxOptions{
		Isolation: parseIsolation(isolation),
	})
	if tx.Error != nil {
		return tx.Error
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

// BatchInsert 批量插入数据，支持大批量数据
func (d *DB) BatchInsert(ctx context.Context, records interface{}, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000
	}
	return d.DB.WithContext(ctx).CreateInBatches(records, batchSize).Error
}

// BatchUpdate 批量更新数据
func (d *DB) BatchUpdate(ctx context.Context, model interface{}, updates map[string]interface{}, conditions map[string]interface{}) error {
	query := d.DB.WithContext(ctx)
	for key, value := range conditions {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}
	return query.Model(model).Updates(updates).Error
}

// UpsertWithConflict 插入或更新（冲突时更新）
func (d *DB) UpsertWithConflict(ctx context.Context, record interface{}, uniqueFields []string, updateFields []string) error {
	return d.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   convertStringsToColumns(uniqueFields),
		DoUpdates: clause.AssignmentColumns(updateFields),
	}).Create(record).Error
}

func convertStringsToColumns(names []string) []clause.Column {
	columns := make([]clause.Column, len(names))
	for i, name := range names {
		columns[i] = clause.Column{Name: name}
	}
	return columns
}

// parseIsolation 解析隔离级别字符串
func parseIsolation(isolation string) sql.IsolationLevel {
	switch isolation {
	case "READ_UNCOMMITTED":
		return sql.LevelReadUncommitted
	case "READ_COMMITTED":
		return sql.LevelReadCommitted
	case "REPEATABLE_READ":
		return sql.LevelRepeatableRead
	case "SERIALIZABLE":
		return sql.LevelSerializable
	default:
		return sql.LevelDefault
	}
}

// GormLogger GORM 日志记录器实现
type GormLogger struct {
	enabled            bool
	slowQueryThreshold time.Duration
}

// NewGormLogger 创建 GORM 日志记录器
func NewGormLogger(enabled bool, slowQueryThreshold time.Duration) *GormLogger {
	return &GormLogger{
		enabled:            enabled,
		slowQueryThreshold: slowQueryThreshold,
	}
}

// LogMode 设置日志模式
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

// Info 记录信息日志
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.enabled {
		pkgLogger.Info(ctx, msg, "data", data)
	}
}

// Warn 记录警告日志
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	pkgLogger.Warn(ctx, msg, "data", data)
}

// Error 记录错误日志
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	pkgLogger.Error(ctx, msg, "data", data)
}

// Trace 记录 SQL 执行日志
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if !l.enabled {
		return
	}

	elapsed := time.Since(begin)
	sqlStr, rows := fc()

	args := []interface{}{
		"duration", elapsed,
		"rows", rows,
		"sql", sqlStr,
	}

	if err != nil {
		args = append(args, "error", err)
		pkgLogger.Error(ctx, "SQL execution failed", args...)
	} else if elapsed > l.slowQueryThreshold {
		pkgLogger.Warn(ctx, "Slow query detected", args...)
	} else if l.enabled {
		pkgLogger.Debug(ctx, "SQL executed", args...)
	}
}
