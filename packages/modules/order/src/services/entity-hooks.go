package services

import "github.com/hitkey-io/medusago/packages/core/utils/src/dal"

// The @OnInit hooks registered on the module models at the top of
// order-module-service.ts.
func init() {
	dal.RegisterOnInitWithRefs("OrderChangeAction", func(data dal.Row, ref func(string) dal.Row) {
		change := ref("order_change")
		if change != nil {
			for _, k := range []string{"version", "order_id", "claim_id", "exchange_id"} {
				if data[k] == nil {
					data[k] = change[k]
				}
			}
		}
		if data["claim_id"] == nil && data["exchange_id"] == nil && change != nil {
			data["return_id"] = change["return_id"]
		}
	})
}
