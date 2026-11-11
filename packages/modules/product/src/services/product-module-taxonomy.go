package services

import (
	"context"
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// This file holds the tag, type and collection methods of
// ProductModuleService (product-module-service.ts).

var (
	kebabCamel = regexp.MustCompile(`([a-z])([A-Z])`)
	kebabSpace = regexp.MustCompile(`[\s_]+`)
)

// KebabCase mirrors kebabCase of @medusajs/utils.
func KebabCase(s string) string {
	s = kebabCamel.ReplaceAllString(s, "$1-$2")
	s = kebabSpace.ReplaceAllString(s, "-")
	return strings.ToLower(s)
}

func toAnyRecords(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i] = r
	}
	return out
}

// withID returns { id, ...data }.
func withID(id any, data *dal.Record) *dal.Record {
	r := dal.NewRecord()
	r.Set("id", id)
	if data != nil {
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			r.Set(k, v)
		}
	}
	return r
}

// normalizeSelectorUpdate mirrors the idOrSelector handling of the
// update<X> methods: a retrieve for an id, a list for a selector.
func (s *ProductModuleService) normalizeSelectorUpdate(ctx context.Context, class string, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	svc := s.Internal(class)
	if id, ok := idOrSelector.(string); ok {
		if _, err := svc.Retrieve(ctx, id, utilsmodulessdk.FindConfig{}); err != nil {
			return nil, err
		}
		return []*dal.Record{withID(id, data)}, nil
	}
	sel, _ := idOrSelector.(*dal.Record)
	rows, err := svc.List(ctx, sel, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	out := make([]*dal.Record, len(rows))
	for i, r := range rows {
		id, _ := r.Get("id")
		out[i] = withID(id, data)
	}
	return out, nil
}

func (s *ProductModuleService) updateMany(ctx context.Context, class string, input []*dal.Record) ([]*dal.Record, error) {
	if len(input) == 0 {
		return []*dal.Record{}, nil
	}
	res, err := s.Internal(class).Update(ctx, toAnyRecords(input))
	if err != nil {
		return nil, err
	}
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}, nil
	}
	return records(res), nil
}

func (s *ProductModuleService) upsertSimple(ctx context.Context, class string, input []*dal.Record) ([]*dal.Record, error) {
	var forCreate, forUpdate []*dal.Record
	for _, r := range input {
		if id, _ := r.Get("id"); id != nil && id != "" {
			forUpdate = append(forUpdate, r)
		} else {
			forCreate = append(forCreate, r)
		}
	}
	var created, updated []*dal.Record
	var err error
	if len(forCreate) > 0 {
		if created, err = s.Internal(class).Create(ctx, forCreate); err != nil {
			return nil, err
		}
	}
	if len(forUpdate) > 0 {
		if updated, err = s.updateMany(ctx, class, forUpdate); err != nil {
			return nil, err
		}
	}
	return append(created, updated...), nil
}

// CreateProductTags mirrors createProductTags.
func (s *ProductModuleService) CreateProductTags(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("ProductTag").Create(ctx, data)
}

// UpsertProductTags mirrors upsertProductTags.
func (s *ProductModuleService) UpsertProductTags(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.upsertSimple(ctx, "ProductTag", data)
}

// UpdateProductTags mirrors updateProductTags(idOrSelector, data).
func (s *ProductModuleService) UpdateProductTags(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	input, err := s.normalizeSelectorUpdate(ctx, "ProductTag", idOrSelector, data)
	if err != nil {
		return nil, err
	}
	return s.updateMany(ctx, "ProductTag", input)
}

// CreateProductTypes mirrors createProductTypes.
func (s *ProductModuleService) CreateProductTypes(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("ProductType").Create(ctx, data)
}

// UpsertProductTypes mirrors upsertProductTypes.
func (s *ProductModuleService) UpsertProductTypes(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.upsertSimple(ctx, "ProductType", data)
}

// UpdateProductTypes mirrors updateProductTypes(idOrSelector, data).
func (s *ProductModuleService) UpdateProductTypes(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	input, err := s.normalizeSelectorUpdate(ctx, "ProductType", idOrSelector, data)
	if err != nil {
		return nil, err
	}
	return s.updateMany(ctx, "ProductType", input)
}

// normalizeUpdateProductCollectionInput mirrors the static helper:
// product_ids become products: [{ id }].
func normalizeUpdateProductCollectionInput(c *dal.Record) *dal.Record {
	out := clone(c)
	if ids, ok := out.Get("product_ids"); ok {
		if l, isList := ids.([]any); isList {
			products := make([]any, len(l))
			for i, id := range l {
				p := dal.NewRecord()
				p.Set("id", id)
				products[i] = p
			}
			out.Set("products", products)
			out.Delete("product_ids")
		}
	}
	return out
}

func normalizeCreateProductCollectionInput(c *dal.Record) *dal.Record {
	out := normalizeUpdateProductCollectionInput(c)
	handle, _ := out.Get("handle")
	title, _ := out.Get("title")
	if h, _ := handle.(string); h == "" {
		if t, _ := title.(string); t != "" {
			out.Set("handle", KebabCase(t))
		}
	}
	return out
}

// productIDsOf returns the ids of a products: [{ id }] input.
func productIDsOf(v any) []any {
	var ids []any
	for _, p := range records(v) {
		id, _ := p.Get("id")
		ids = append(ids, id)
	}
	return ids
}

// CreateProductCollections mirrors createProductCollections
// (upsertWithReplace with the products relation: the given products get
// the collection).
func (s *ProductModuleService) CreateProductCollections(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var err error
		out, err = t.createCollections(ctx, data)
		return err
	})
	return out, err
}

func (s *ProductModuleService) createCollections(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	input := make([]*dal.Record, len(data))
	products := make([]any, len(data))
	for i, c := range data {
		n := normalizeCreateProductCollectionInput(c)
		products[i], _ = n.Get("products")
		n.Delete("products")
		input[i] = n
	}
	created, err := s.Internal("ProductCollection").Create(ctx, input)
	if err != nil {
		return nil, err
	}
	for i, c := range created {
		ids := productIDsOf(products[i])
		if len(ids) > 0 {
			id, _ := c.Get("id")
			if err := s.setProductsCollection(ctx, ids, id); err != nil {
				return nil, err
			}
		}
	}
	return created, nil
}

func (s *ProductModuleService) setProductsCollection(ctx context.Context, ids []any, collectionID any) error {
	in := dal.NewRecord()
	in.Set("$in", ids)
	sel := dal.NewRecord()
	sel.Set("id", in)
	data := dal.NewRecord()
	data.Set("collection_id", collectionID)
	u := dal.NewRecord()
	u.Set("selector", sel)
	u.Set("data", data)
	_, err := s.Internal("Product").Update(ctx, []any{u})
	return err
}

// UpsertProductCollections mirrors upsertProductCollections.
func (s *ProductModuleService) UpsertProductCollections(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var forCreate, forUpdate []*dal.Record
		for _, r := range data {
			if id, _ := r.Get("id"); id != nil && id != "" {
				forUpdate = append(forUpdate, r)
			} else {
				forCreate = append(forCreate, r)
			}
		}
		var created, updated []*dal.Record
		var err error
		if len(forCreate) > 0 {
			if created, err = t.createCollections(ctx, forCreate); err != nil {
				return err
			}
		}
		if len(forUpdate) > 0 {
			if updated, err = t.updateCollections(ctx, forUpdate); err != nil {
				return err
			}
		}
		out = append(created, updated...)
		return nil
	})
	return out, err
}

// UpdateProductCollections mirrors updateProductCollections(idOrSelector,
// data).
func (s *ProductModuleService) UpdateProductCollections(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		input, err := t.normalizeSelectorUpdate(ctx, "ProductCollection", idOrSelector, data)
		if err != nil {
			return err
		}
		out, err = t.updateCollections(ctx, input)
		return err
	})
	return out, err
}

// updateCollections mirrors updateCollections_: products given replace the
// collection's products.
func (s *ProductModuleService) updateCollections(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	normalized := make([]*dal.Record, len(data))
	plain := make([]*dal.Record, len(data))
	for i, c := range data {
		normalized[i] = normalizeUpdateProductCollectionInput(c)
		p := clone(normalized[i])
		p.Delete("products")
		plain[i] = p
	}
	updated, err := s.updateMany(ctx, "ProductCollection", plain)
	if err != nil {
		return nil, err
	}
	var toUpdate []any
	for _, c := range updated {
		id, _ := c.Get("id")
		var input *dal.Record
		for _, n := range normalized {
			if nid, _ := n.Get("id"); nid == id {
				input = n
				break
			}
		}
		if input == nil {
			continue
		}
		products, defined := input.Get("products")
		if !defined || products == nil {
			continue
		}
		ids := productIDsOf(products)
		if ids == nil {
			ids = []any{}
		}
		nin := dal.NewRecord()
		nin.Set("$nin", ids)
		dis := dal.NewRecord()
		dis.Set("collection_id", id)
		dis.Set("id", nin)
		disData := dal.NewRecord()
		disData.Set("collection_id", nil)
		u := dal.NewRecord()
		u.Set("selector", dis)
		u.Set("data", disData)
		toUpdate = append(toUpdate, u)
		if len(ids) > 0 {
			in := dal.NewRecord()
			in.Set("$in", ids)
			as := dal.NewRecord()
			as.Set("id", in)
			asData := dal.NewRecord()
			asData.Set("collection_id", id)
			u2 := dal.NewRecord()
			u2.Set("selector", as)
			u2.Set("data", asData)
			toUpdate = append(toUpdate, u2)
		}
	}
	if len(toUpdate) > 0 {
		if _, err := s.Internal("Product").Update(ctx, toUpdate); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// SoftDeleteProductOptions mirrors softDeleteProductOptions: options still
// associated with live products cannot be deleted; their product pivot rows
// are soft deleted first.
func (s *ProductModuleService) SoftDeleteProductOptions(ctx context.Context, ids []any) (*dal.Record, error) {
	// no @EmitEvents upstream: the mutations emit no events
	ctx = dal.WithoutEvents(ctx)
	if len(ids) == 0 {
		return dal.NewRecord(), nil
	}
	var out *dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		blocking, err := dal.QueryRows(ctx, t.db, `select "ppo"."product_option_id" from "product_product_option" as "ppo"
inner join "product" as "p" on "p"."id" = "ppo"."product_id"
inner join "product_option" as "po" on "po"."id" = "ppo"."product_option_id"
where "ppo"."product_option_id" = any($1) and "ppo"."deleted_at" is null and "p"."deleted_at" is null and "po"."deleted_at" is null limit 1`, toStrings(ids))
		if err != nil {
			return err
		}
		if len(blocking) > 0 {
			return invalid("Cannot delete product options that are associated with products.")
		}
		f := dal.NewRecord()
		f.Set("product_option_id", ids)
		ppos, err := t.Internal("ProductProductOption").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return err
		}
		if len(ppos) > 0 {
			ppoIDs := make([]any, len(ppos))
			for i, r := range ppos {
				ppoIDs[i], _ = r.Get("id")
			}
			vf := dal.NewRecord()
			vf.Set("product_product_option_id", ppoIDs)
			ppovs, err := t.Internal("ProductProductOptionValue").List(ctx, vf, utilsmodulessdk.FindConfig{Select: []string{"id"}})
			if err != nil {
				return err
			}
			if len(ppovs) > 0 {
				vids := make([]any, len(ppovs))
				for i, r := range ppovs {
					vids[i], _ = r.Get("id")
				}
				if _, err := t.Internal("ProductProductOptionValue").SoftDelete(ctx, vids); err != nil {
					return err
				}
			}
			if _, err := t.Internal("ProductProductOption").SoftDelete(ctx, ppoIDs); err != nil {
				return err
			}
		}
		res, err := t.Call(ctx, "softDeleteProductOptions", []any{ids})
		if err != nil {
			return err
		}
		out, _ = res.(*dal.Record)
		return nil
	})
	return out, err
}
