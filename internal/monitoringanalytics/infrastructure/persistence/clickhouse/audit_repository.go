package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

type auditRepository struct {
	conn driver.Conn
}

func NewAuditRepository(conn driver.Conn) domain.ExecutionAuditRepository {
	return &auditRepository{conn: conn}
}

func (r *auditRepository) BatchSave(ctx context.Context, audits []*domain.ExecutionAudit) error {
	if len(audits) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, "INSERT INTO execution_audits (id, trade_id, order_id, user_id, symbol, side, price, quantity, fee, venue, algo_type, timestamp)")
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, a := range audits {
		err := batch.Append(
			a.ID,
			a.TradeID,
			a.OrderID,
			a.UserID,
			a.Symbol,
			a.Side,
			a.Price.InexactFloat64(),
			a.Quantity.InexactFloat64(),
			a.Fee.InexactFloat64(),
			a.Venue,
			a.AlgoType,
			a.Timestamp,
		)
		if err != nil {
			return fmt.Errorf("failed to append to batch: %w", err)
		}
	}

	return batch.Send()
}

func (r *auditRepository) Query(ctx context.Context, userID, symbol string, startTime, endTime int64) ([]*domain.ExecutionAudit, error) {
	query := `SELECT id, trade_id, order_id, user_id, symbol, side, price, quantity, fee, venue, algo_type, timestamp 
	          FROM execution_audits 
	          WHERE timestamp >= ? AND timestamp <= ?`
	args := []any{startTime, endTime}

	if userID != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}
	if symbol != "" {
		query += " AND symbol = ?"
		args = append(args, symbol)
	}

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.ExecutionAudit
	for rows.Next() {
		var a domain.ExecutionAudit
		var p, q, f float64
		if err := rows.Scan(&a.ID, &a.TradeID, &a.OrderID, &a.UserID, &a.Symbol, &a.Side, &p, &q, &f, &a.Venue, &a.AlgoType, &a.Timestamp); err != nil {
			return nil, err
		}
		a.Price = decimal.NewFromFloat(p)
		a.Quantity = decimal.NewFromFloat(q)
		a.Fee = decimal.NewFromFloat(f)
		results = append(results, &a)
	}
	return results, nil
}
