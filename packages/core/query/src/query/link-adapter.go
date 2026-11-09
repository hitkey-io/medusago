package query

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/services"
)

// LinkServiceAdapter exposes a link module service to Query (the method
// suffix does not apply to link modules).
type LinkServiceAdapter struct{ Service *services.LinkModuleService }

// List calls list.
func (a LinkServiceAdapter) List(ctx context.Context, _ string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	return a.Service.List(ctx, filters, config)
}

// ListAndCount calls listAndCount.
func (a LinkServiceAdapter) ListAndCount(ctx context.Context, _ string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	return a.Service.ListAndCount(ctx, filters, config)
}
