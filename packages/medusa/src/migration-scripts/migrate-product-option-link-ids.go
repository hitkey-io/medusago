package migrationscripts

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

const optionLinkBatchSize = 1000

func (c Context) updateIDsInBatches(ctx context.Context, tx pgx.Tx, table, prefix string) error {
	pattern := prefix + "_%"
	var total int
	if err := tx.QueryRow(ctx, `select count(id)::int as count from "`+table+`" where id not like $1`, pattern).Scan(&total); err != nil {
		return err
	}
	if total == 0 {
		c.Logger.Info(fmt.Sprintf("No ids to rewrite in %s. Skipping.", table))
		return nil
	}
	c.Logger.Info(fmt.Sprintf("Rewriting %d ids in %s...", total, table))
	maxBatches := (total + optionLinkBatchSize - 1) / optionLinkBatchSize
	updated := 0
	for batch := 0; batch < maxBatches; batch++ {
		rows, err := tx.Query(ctx, `select id from "`+table+`" where id not like $1 order by id limit $2`, pattern, optionLinkBatchSize)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		values := make([]string, len(ids))
		args := make([]any, 0, len(ids)*2)
		for i, id := range ids {
			args = append(args, id, common.GenerateEntityID("", prefix))
			values[i] = fmt.Sprintf("($%d, $%d)", 2*i+1, 2*i+2)
		}
		if _, err := tx.Exec(ctx, `update "`+table+`" as t
        set id = v.new_id
        from (values `+strings.Join(values, ", ")+`) as v(old_id, new_id)
        where t.id = v.old_id`, args...); err != nil {
			return err
		}
		updated += len(ids)
		c.Logger.Info(fmt.Sprintf("Rewrote %d/%d ids in %s.", updated, total, table))
	}
	c.Logger.Info(fmt.Sprintf("Finished rewriting %d ids in %s.", updated, table))
	return nil
}

// migrateProductOptionLinkIDs mirrors migrate-product-option-link-ids.ts.
func (c Context) migrateProductOptionLinkIDs(ctx context.Context) error {
	if !c.isInstalled(utilsmodulessdk.PRODUCT) {
		return nil
	}
	c.Logger.Info("Starting migration of product option link ids...")
	err := pgx.BeginFunc(ctx, c.DB, func(tx pgx.Tx) error {
		if err := c.updateIDsInBatches(ctx, tx, "product_product_option", "prodopt"); err != nil {
			return err
		}
		return c.updateIDsInBatches(ctx, tx, "product_product_option_value", "prodoptval")
	})
	if err != nil {
		c.Logger.Error(err.Error())
		return err
	}
	c.Logger.Info("Finished migration of product option link ids.")
	return nil
}
