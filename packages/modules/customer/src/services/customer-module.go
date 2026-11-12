// Package services mirrors @medusajs/customer/src/services.
package services

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/customer/src/models"
)

// CustomerModuleService mirrors CustomerModuleService: the generated service
// plus the customer group membership methods.
type CustomerModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewCustomerModuleService builds the service.
func NewCustomerModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*CustomerModuleService, error) {
	return &CustomerModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

// AddCustomerToGroup mirrors addCustomerToGroup: the { customer_id,
// customer_group_id } pairs are created in a transaction; their ids are
// returned.
func (s *CustomerModuleService) AddCustomerToGroup(ctx context.Context, pairs []*dal.Record) ([]*dal.Record, error) {
	out := []*dal.Record{}
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		created, err := m.Internal("CustomerGroupCustomer").Create(ctx, pairs)
		if err != nil {
			return err
		}
		for _, c := range created {
			r := dal.NewRecord()
			id, _ := c.Get("id")
			r.Set("id", id)
			out = append(out, r)
		}
		return nil
	})
	return out, err
}

// RemoveCustomerFromGroup mirrors removeCustomerFromGroup: the memberships
// matching any of the pairs are deleted.
func (s *CustomerModuleService) RemoveCustomerFromGroup(ctx context.Context, pairs []*dal.Record) error {
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		or := make([]any, len(pairs))
		for i, p := range pairs {
			or[i] = p
		}
		f := dal.NewRecord()
		f.Set("$or", or)
		rows, err := m.Internal("CustomerGroupCustomer").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		ids := make([]any, 0, len(rows))
		for _, r := range rows {
			id, _ := r.Get("id")
			ids = append(ids, id)
		}
		_, err = m.Internal("CustomerGroupCustomer").Delete(ctx, ids)
		return err
	})
}
