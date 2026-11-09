// Package services mirrors @medusajs/product/src/services.
package services

import (
	"context"
	"slices"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/product/src/models"
)

// ProductModuleService mirrors ProductModuleService: the generated
// MedusaService plus the overridden methods.
type ProductModuleService struct {
	*utilsmodulessdk.MedusaService
	db dal.Querier
}

// NewProductModuleService builds the service.
func NewProductModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*ProductModuleService, error) {
	return &ProductModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig), db: opts.DB}, nil
}

// ListBySuffix dispatches list<suffix> to the overridden methods.
func (s *ProductModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	switch suffix {
	case "Products":
		rows, _, err := s.listProducts(ctx, filters, config, false)
		return rows, err
	case "ProductVariants":
		rows, _, err := s.listProductVariants(ctx, filters, config, false)
		return rows, err
	case "ProductTags", "ProductTypes", "ProductCollections", "ProductCategories":
		filters = withoutContext(filters)
	}
	return s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
}

// ListAndCountBySuffix dispatches listAndCount<suffix>.
func (s *ProductModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	switch suffix {
	case "Products":
		return s.listProducts(ctx, filters, config, true)
	case "ProductVariants":
		return s.listProductVariants(ctx, filters, config, true)
	case "ProductTags", "ProductTypes", "ProductCollections", "ProductCategories":
		filters = withoutContext(filters)
	}
	return s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
}

// withoutContext mirrors extractPublishedProductsContext (the published
// products populateWhere is not supported yet).
func withoutContext(filters *dal.Record) *dal.Record {
	if filters == nil || !filters.Has("context") {
		return filters
	}
	out := dal.NewRecord()
	for _, k := range filters.Keys() {
		if k == "context" {
			continue
		}
		v, _ := filters.Get(k)
		out.Set(k, v)
	}
	return out
}

func addRelation(relations []string, r string) []string {
	if !slices.Contains(relations, r) {
		relations = append(relations, r)
	}
	return relations
}

// productFindConfig mirrors getProductFindConfig_: the order defaults to
// { id: "ASC" } and the images are ordered by rank.
func productFindConfig(config utilsmodulessdk.FindConfig) utilsmodulessdk.FindConfig {
	c := config
	if len(c.Order) == 0 {
		c.Order = []dal.OrderBy{{Field: "id", Direction: "ASC"}}
	} else {
		c.Order = append([]dal.OrderBy(nil), c.Order...)
	}
	if slices.Contains(c.Relations, "images") {
		c.Order = append(c.Order, dal.OrderBy{Field: "images.rank", Direction: "ASC"})
	}
	return c
}

// listProducts mirrors listProducts / listAndCountProducts.
func (s *ProductModuleService) listProducts(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig, count bool) ([]*dal.Record, int, error) {
	var relations []string
	for _, r := range config.Relations {
		if r != "product_options" && r != "product_options.values" && r != "product_options.product_option" {
			relations = append(relations, r)
		}
	}
	loadVariantImages := slices.Contains(relations, "variants.images")
	filterValues := slices.Contains(relations, "options.values")
	if loadVariantImages {
		relations = addRelation(addRelation(relations, "variants"), "images")
	}
	if filterValues {
		relations = addRelation(addRelation(relations, "options"), "options.values")
	}
	config.Relations = relations
	svc := s.Internal("Product")
	var rows []*dal.Record
	n := 0
	var err error
	if count {
		rows, n, err = svc.ListAndCount(ctx, filters, productFindConfig(config))
	} else {
		rows, err = svc.List(ctx, filters, productFindConfig(config))
	}
	if err != nil {
		return nil, 0, err
	}
	if loadVariantImages {
		if err := s.assignImagesToVariants(ctx, rows); err != nil {
			return nil, 0, err
		}
	}
	if filterValues {
		if err := s.filterOptionValues(ctx, rows); err != nil {
			return nil, 0, err
		}
	}
	return rows, n, nil
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func str(r *dal.Record, k string) string {
	v, _ := r.Get(k)
	s, _ := v.(string)
	return s
}

// assignImagesToVariants mirrors assignImagesToVariants: every variant gets
// the product's general images (assigned to no variant) plus the images
// assigned to it.
func (s *ProductModuleService) assignImagesToVariants(ctx context.Context, products []*dal.Record) error {
	var toProcess []*dal.Record
	var variantIDs []any
	for _, p := range products {
		vs, _ := p.Get("variants")
		imgs, hasImages := p.Get("images")
		if len(records(vs)) == 0 || !hasImages || imgs == nil {
			continue
		}
		toProcess = append(toProcess, p)
		for _, v := range records(vs) {
			variantIDs = append(variantIDs, str(v, "id"))
		}
	}
	if len(variantIDs) == 0 {
		return nil
	}
	f := dal.NewRecord()
	f.Set("variant_id", variantIDs)
	rels, err := s.Internal("ProductVariantProductImage").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"variant_id", "image_id"}})
	if err != nil {
		return err
	}
	byVariant := map[string]map[string]bool{}
	withVariants := map[string]bool{}
	for _, r := range rels {
		v, i := str(r, "variant_id"), str(r, "image_id")
		if byVariant[v] == nil {
			byVariant[v] = map[string]bool{}
		}
		byVariant[v][i] = true
		withVariants[i] = true
	}
	for _, p := range toProcess {
		imgs, _ := p.Get("images")
		var general, specific []*dal.Record
		for _, img := range records(imgs) {
			if withVariants[str(img, "id")] {
				specific = append(specific, img)
			} else {
				general = append(general, img)
			}
		}
		vs, _ := p.Get("variants")
		for _, v := range records(vs) {
			images := append([]*dal.Record{}, general...)
			if ids := byVariant[str(v, "id")]; len(ids) > 0 {
				for _, img := range specific {
					if ids[str(img, "id")] {
						images = append(images, img)
					}
				}
			}
			v.Set("images", images)
		}
	}
	return nil
}

// getOptionValueIDsByProductIDs mirrors
// ProductRepository.getOptionValueIdsByProductIds.
func (s *ProductModuleService) getOptionValueIDsByProductIDs(ctx context.Context, productIDs []any) (map[string]map[string]bool, error) {
	out := map[string]map[string]bool{}
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `select "ppo"."product_id", "ppov"."product_option_value_id" from "product_product_option_value" as "ppov" inner join "product_product_option" as "ppo" on "ppo"."id" = "ppov"."product_product_option_id" and "ppo"."deleted_at" is null where "ppo"."product_id" = any($1) and "ppov"."deleted_at" is null`, toStrings(productIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pid, vid string
		if err := rows.Scan(&pid, &vid); err != nil {
			return nil, err
		}
		if out[pid] == nil {
			out[pid] = map[string]bool{}
		}
		out[pid][vid] = true
	}
	return out, rows.Err()
}

func toStrings(v []any) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// filterOptionValues mirrors filterOptionValues: each option keeps only
// the values assigned to the product.
func (s *ProductModuleService) filterOptionValues(ctx context.Context, products []*dal.Record) error {
	var with []*dal.Record
	var ids []any
	for _, p := range products {
		opts, _ := p.Get("options")
		if len(records(opts)) == 0 {
			continue
		}
		with = append(with, p)
		if id := str(p, "id"); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	allowed, err := s.getOptionValueIDsByProductIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, p := range with {
		ok := allowed[str(p, "id")]
		opts, _ := p.Get("options")
		for _, o := range records(opts) {
			vals, _ := o.Get("values")
			kept := []*dal.Record{}
			for _, v := range records(vals) {
				if ok[str(v, "id")] {
					kept = append(kept, v)
				}
			}
			o.Set("values", kept)
		}
	}
	return nil
}

// listProductVariants mirrors listProductVariants /
// listAndCountProductVariants.
func (s *ProductModuleService) listProductVariants(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig, count bool) ([]*dal.Record, int, error) {
	loadImages := slices.Contains(config.Relations, "images")
	if loadImages {
		config.Relations = append(append([]string{}, config.Relations...), "product.images")
	}
	svc := s.Internal("ProductVariant")
	var rows []*dal.Record
	n := 0
	var err error
	if count {
		rows, n, err = svc.ListAndCount(ctx, filters, config)
	} else {
		rows, err = svc.List(ctx, filters, config)
	}
	if err != nil {
		return nil, 0, err
	}
	if loadImages {
		images, err := s.getVariantImages(ctx, rows)
		if err != nil {
			return nil, 0, err
		}
		for _, v := range rows {
			imgs := images[str(v, "id")]
			if imgs == nil {
				imgs = []*dal.Record{}
			}
			v.Set("images", imgs)
		}
	}
	return rows, n, nil
}

// getVariantImages mirrors getVariantImages.
func (s *ProductModuleService) getVariantImages(ctx context.Context, variants []*dal.Record) (map[string][]*dal.Record, error) {
	out := map[string][]*dal.Record{}
	if len(variants) == 0 {
		return out, nil
	}
	var productIDs []any
	seen := map[string]bool{}
	for _, v := range variants {
		if pid := str(v, "product_id"); pid != "" && !seen[pid] {
			seen[pid] = true
			productIDs = append(productIDs, pid)
		}
	}
	f := dal.NewRecord()
	f.Set("product_id", productIDs)
	images, err := s.Internal("ProductImage").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"variants"}})
	if err != nil {
		return nil, err
	}
	byProduct := map[string][]*dal.Record{}
	specific := map[string]map[string]bool{}
	for _, img := range images {
		pid := str(img, "product_id")
		byProduct[pid] = append(byProduct[pid], img)
		vs, _ := img.Get("variants")
		for _, v := range records(vs) {
			vid := str(v, "id")
			if specific[vid] == nil {
				specific[vid] = map[string]bool{}
			}
			specific[vid][str(img, "id")] = true
		}
	}
	for _, v := range variants {
		var list []*dal.Record
		for _, img := range byProduct[str(v, "product_id")] {
			vs, _ := img.Get("variants")
			if len(records(vs)) == 0 || specific[str(v, "id")][str(img, "id")] {
				list = append(list, img)
			}
		}
		out[str(v, "id")] = list
	}
	return out, nil
}
