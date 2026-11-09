// Package fulfillmentsets mirrors api/admin/fulfillment-sets.
package fulfillmentsets

import (
	"fmt"

	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// Defaults mirrors defaultAdminFulfillmentSetsFields.
var Defaults = []string{"id", "name", "type", "created_at", "updated_at", "deleted_at", "*service_zones", "*service_zones.geo_zones"}

// Query configs.
var (
	RetrieveConfig            = httputils.QueryConfig{Defaults: Defaults, Entity: "fulfillment_set"}
	RetrieveServiceZoneConfig = httputils.QueryConfig{Defaults: []string{"id", "name", "type", "created_at", "updated_at", "deleted_at", "*geo_zones"}, Entity: "fulfillment_set"}
)

func geoZoneBase(extra ...zod.Field) *zod.Schema {
	fields := []zod.Field{
		zod.F("country_code", zod.String()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	}
	return zod.Object(append(fields, extra...)...)
}

// Geo zone schemas mirror validators/geo-zone.ts.
var (
	GeoZoneCountrySchema  = geoZoneBase(zod.F("type", zod.Literal("country")))
	GeoZoneProvinceSchema = geoZoneBase(zod.F("type", zod.Literal("province")), zod.F("province_code", zod.String()))
	GeoZoneCitySchema     = geoZoneBase(zod.F("type", zod.Literal("city")), zod.F("province_code", zod.String()), zod.F("city", zod.String()))
	GeoZoneZipSchema      = geoZoneBase(zod.F("type", zod.Literal("zip")), zod.F("province_code", zod.String()), zod.F("city", zod.String()), zod.F("postal_expression", zod.Record(zod.String(), zod.Unknown())))
)

func withID(s *zod.Schema) *zod.Schema {
	return s.Merge(zod.Object(zod.F("id", zod.String().Optional())))
}

// AdminCreateFulfillmentSetServiceZonesSchema mirrors the create schema.
var AdminCreateFulfillmentSetServiceZonesSchema = zod.Object(
	zod.F("name", zod.String()),
	zod.F("geo_zones", zod.Array(zod.Union(GeoZoneCountrySchema, GeoZoneProvinceSchema, GeoZoneCitySchema, GeoZoneZipSchema)).Optional()),
).Strict()

// AdminUpdateFulfillmentSetServiceZonesSchema mirrors the update schema.
var AdminUpdateFulfillmentSetServiceZonesSchema = zod.Object(
	zod.F("name", zod.String().Nullish()),
	zod.F("geo_zones", zod.Array(zod.Union(withID(GeoZoneCountrySchema), withID(GeoZoneProvinceSchema), withID(GeoZoneCitySchema), withID(GeoZoneZipSchema))).Optional()),
).Strict()

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminFulfillmentSetsRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/fulfillment-sets/:id/service-zones", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateFulfillmentSetServiceZonesSchema)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"DELETE"}, Matcher: "/admin/fulfillment-sets/:id/service-zones/:zone_id", Middlewares: mw(httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/fulfillment-sets/:id/service-zones/:zone_id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateFulfillmentSetServiceZonesSchema)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/fulfillment-sets/:id/service-zones/:zone_id", Middlewares: mw(httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveServiceZoneConfig))},
}

// RetrieveSetWithZones mirrors retrieveFulfillmentSet(id, { relations:
// ["service_zones"] }) and checks the zone belongs to it.
func RetrieveSetWithZones(req *medusahttp.MedusaRequest, setID, zoneID string) (*dal.Record, error) {
	svc := fulfillmentsteps.Fulfillment(req.Scope)
	set, err := svc.Internal("FulfillmentSet").Retrieve(req.Ctx(), setID, utilsmodulessdk.FindConfig{Relations: []string{"service_zones"}})
	if err != nil {
		return nil, err
	}
	zones, _ := set.Get("service_zones")
	found := false
	var list []*dal.Record
	switch t := zones.(type) {
	case []any:
		for _, z := range t {
			if zr, ok := z.(*dal.Record); ok {
				list = append(list, zr)
			}
		}
	case []*dal.Record:
		list = t
	}
	for _, zr := range list {
		if id, _ := zr.Get("id"); fmt.Sprint(id) == zoneID {
			found = true
		}
	}
	if !found {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Service zone with id: %s not found on fulfillment set", zoneID))
	}
	return set, nil
}

// RefetchFulfillmentSet mirrors refetchFulfillmentSet.
func RefetchFulfillmentSet(req *medusahttp.MedusaRequest, id any) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "fulfillment_set", f, req.QueryConfig.Fields)
}
