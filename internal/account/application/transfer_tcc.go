// 变更说明：实现金融级转账的 TCC 事务。
// Try 阶段：Account 冻结余额，Ledger 预检查科目。
// Confirm 阶段：Account 扣减冻结，Ledger 正式记账。
// Cancel 阶段：Account 释放冻结，Ledger 标记失败。
package application

import (
	"context"
	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/wyfcoding/pkg/dtm"
	"google.golang.org/protobuf/proto"
)

type FinancialTransferTCC struct {
	dtmServer string
}

func (t *FinancialTransferTCC) ExecuteTransfer(ctx context.Context, gid string, payload proto.Message) error {
	tcc := dtm.NewTcc(t.dtmServer, gid)

	return tcc.Execute(ctx, func(tccInstance *dtmgrpc.TccGrpc) error {
		// 分支 1：账户余额更新
		if err := dtm.CallBranch(tccInstance, payload, "account.svc/TryTransfer", "account.svc/ConfirmTransfer", "account.svc/CancelTransfer"); err != nil {
			return err
		}
		// 分支 2：复式记账凭证更新
		if err := dtm.CallBranch(tccInstance, payload, "ledger.svc/TryPost", "ledger.svc/ConfirmPost", "ledger.svc/CancelPost"); err != nil {
			return err
		}
		return nil
	})
}
