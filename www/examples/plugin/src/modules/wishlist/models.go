package wishlist

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// Wishlist is the data model of the `wishlist` table.
var Wishlist = dml.Define("wishlist", dml.Schema{
	{Name: "id", Property: dml.Id().PrimaryKey()},
	{Name: "customer_id", Property: dml.Text()},
})
