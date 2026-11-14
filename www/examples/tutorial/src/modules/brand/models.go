package brand

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// region: brand
// Brand is the data model of the `brand` table.
var Brand = dml.Define("brand", dml.Schema{
	{Name: "id", Property: dml.Id().PrimaryKey()},
	{Name: "name", Property: dml.Text()},
})

// endregion: brand
