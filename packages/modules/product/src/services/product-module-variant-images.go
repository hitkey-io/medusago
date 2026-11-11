package services

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// AddImageToVariant mirrors addImageToVariant: pairs { image_id,
// variant_id } become variant images; their ids are returned.
func (s *ProductModuleService) AddImageToVariant(ctx context.Context, pairs []*dal.Record) ([]*dal.Record, error) {
	// no @EmitEvents upstream: the mutations emit no events
	ctx = dal.WithoutEvents(ctx)
	created, err := s.Internal("ProductVariantProductImage").Create(ctx, pairs)
	if err != nil {
		return nil, err
	}
	out := make([]*dal.Record, len(created))
	for i, c := range created {
		r := dal.NewRecord()
		id, _ := c.Get("id")
		r.Set("id", id)
		out[i] = r
	}
	return out, nil
}

// RemoveImageFromVariant mirrors removeImageFromVariant.
func (s *ProductModuleService) RemoveImageFromVariant(ctx context.Context, pairs []*dal.Record) error {
	// no @EmitEvents upstream: the mutations emit no events
	ctx = dal.WithoutEvents(ctx)
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		or := make([]any, len(pairs))
		for i, p := range pairs {
			or[i] = p
		}
		f := dal.NewRecord()
		f.Set("$or", or)
		rows, err := tx.Internal("ProductVariantProductImage").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		ids := make([]any, len(rows))
		for i, r := range rows {
			ids[i], _ = r.Get("id")
		}
		if len(ids) == 0 {
			return nil
		}
		_, err = tx.Internal("ProductVariantProductImage").Delete(ctx, ids)
		return err
	})
}
