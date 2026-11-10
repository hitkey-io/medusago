package propertylabels

import (
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// DefaultPropertyLabelFields mirrors defaultPropertyLabelFields.
var DefaultPropertyLabelFields = []string{
	"id",
	"entity",
	"property",
	"label",
	"description",
	"created_at",
	"updated_at",
}

// Query configs.
var (
	RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultPropertyLabelFields, Entity: "property_label"}
	ListTransformQueryConfig     = httputils.QueryConfig{Defaults: DefaultPropertyLabelFields, IsList: true, Entity: "property_label"}
)
