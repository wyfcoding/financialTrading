// Package domain 合规服务仓储接口
package domain

import "context"

type KYCRepository interface {
	Save(ctx context.Context, kyc *KYCApplication) error
	GetByUserID(ctx context.Context, userID uint64) (*KYCApplication, error)
	GetByApplicationID(ctx context.Context, appID string) (*KYCApplication, error)
	GetPending(ctx context.Context, limit int) ([]*KYCApplication, error)
}

type AMLRepository interface {
	Save(ctx context.Context, record *AMLRecord) error
	GetLatestByUserID(ctx context.Context, userID uint64) (*AMLRecord, error)
}
