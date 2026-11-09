package migrationscripts

import (
	"context"
	"fmt"

	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// backfillRefundStatus mirrors backfill-refund-status.ts.
func (c Context) backfillRefundStatus(ctx context.Context) error {
	if !c.isInstalled(utilsmodulessdk.PAYMENT) {
		return nil
	}
	c.Logger.Info("Backfilling the status of existing refunds")
	updated := int64(0)
	for {
		tag, err := c.DB.Exec(ctx, `update refund set status = 'succeeded'
         where id in (
           select id from refund
           where status = 'pending' and deleted_at is null
           limit $1 for update skip locked
         )`, 50000)
		if err != nil {
			c.Logger.Error(err.Error())
			return err
		}
		if tag.RowsAffected() == 0 {
			break
		}
		updated += tag.RowsAffected()
	}
	c.Logger.Info(fmt.Sprintf("Marked %d refunds as succeeded", updated))
	return nil
}
