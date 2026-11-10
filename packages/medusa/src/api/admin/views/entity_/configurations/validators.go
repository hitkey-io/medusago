package configurations

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminGetViewConfigurationParams mirrors AdminGetViewConfigurationParams.
var AdminGetViewConfigurationParams = utils.CreateSelectParams()

// AdminGetActiveViewConfigurationParams mirrors
// AdminGetActiveViewConfigurationParams.
var AdminGetActiveViewConfigurationParams = utils.CreateSelectParams()

// AdminGetViewConfigurationsParamsFields mirrors
// AdminGetViewConfigurationsParamsFields.
var AdminGetViewConfigurationsParamsFields = zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("entity", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
	zod.F("user_id", zod.Union(zod.String(), zod.Array(zod.String()), zod.Null()).Optional()),
	zod.F("is_system_default", zod.Boolean().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetViewConfigurationsParams mirrors AdminGetViewConfigurationsParams.
var AdminGetViewConfigurationsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(20)}).
	Merge(AdminGetViewConfigurationsParamsFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetViewConfigurationsParamsFields))

func sortingSchema() *zod.Schema {
	return zod.Object(
		zod.F("id", zod.String()),
		zod.F("desc", zod.Boolean()),
	).Nullable().Optional()
}

// AdminCreateViewConfiguration mirrors AdminCreateViewConfiguration.
var AdminCreateViewConfiguration = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("is_system_default", zod.Boolean().Optional().Default(false)),
	zod.F("set_active", zod.Boolean().Optional().Default(false)),
	zod.F("configuration", zod.Object(
		zod.F("visible_columns", zod.Array(zod.String())),
		zod.F("column_order", zod.Array(zod.String())),
		zod.F("column_widths", zod.Record(zod.String(), zod.Number()).Optional()),
		zod.F("filters", zod.Record(zod.String(), zod.Any()).Optional()),
		zod.F("sorting", sortingSchema()),
		zod.F("search", zod.String().Optional()),
	)),
)

// AdminUpdateViewConfiguration mirrors AdminUpdateViewConfiguration.
var AdminUpdateViewConfiguration = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("is_system_default", zod.Boolean().Optional()),
	zod.F("set_active", zod.Boolean().Optional().Default(false)),
	zod.F("configuration", zod.Object(
		zod.F("visible_columns", zod.Array(zod.String()).Optional()),
		zod.F("column_order", zod.Array(zod.String()).Optional()),
		zod.F("column_widths", zod.Record(zod.String(), zod.Number()).Optional()),
		zod.F("filters", zod.Record(zod.String(), zod.Any()).Optional()),
		zod.F("sorting", sortingSchema()),
		zod.F("search", zod.String().Optional()),
	).Optional()),
)

// AdminSetActiveViewConfiguration mirrors AdminSetActiveViewConfiguration.
var AdminSetActiveViewConfiguration = zod.Object(
	zod.F("view_configuration_id", zod.Union(zod.String(), zod.Null())),
)
