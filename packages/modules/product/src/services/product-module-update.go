package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// UpdateProducts mirrors updateProducts(idOrSelector, data).
func (s *ProductModuleService) UpdateProducts(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		input, err := t.normalizeSelectorUpdate(ctx, "Product", idOrSelector, data)
		if err != nil {
			return err
		}
		products, err := t.updateProducts(ctx, input)
		if err != nil {
			return err
		}
		if err := t.filterOptionValues(ctx, products); err != nil {
			return err
		}
		out = products
		return nil
	})
	return out, err
}

// UpsertProducts mirrors upsertProducts.
func (s *ProductModuleService) UpsertProducts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var forCreate, forUpdate []*dal.Record
	for _, p := range data {
		if id, _ := p.Get("id"); id != nil && id != "" {
			forUpdate = append(forUpdate, p)
		} else {
			forCreate = append(forCreate, p)
		}
	}
	var created, updated []*dal.Record
	var err error
	if len(forCreate) > 0 {
		if created, err = s.CreateProducts(ctx, forCreate); err != nil {
			return nil, err
		}
	}
	if len(forUpdate) > 0 {
		err = s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
			t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
			var uerr error
			updated, uerr = t.updateProducts(ctx, forUpdate)
			return uerr
		})
		if err != nil {
			return nil, err
		}
	}
	return append(created, updated...), nil
}

// productScalarKeys are the update keys assigned to the product row itself.
var productRelationKeys = map[string]bool{
	"options": true, "variants": true, "tags": true, "categories": true, "images": true,
	"option_ids": true, "type": true, "collection": true,
}

// updateProducts mirrors updateProducts_ and ProductRepository.deepUpdate:
// option links (option_ids), scalar fields, variants (updated, created, and
// the loaded ones missing from the input deleted: orphanRemoval), tags,
// categories and images. Assigning product.options directly is not
// supported.
func (s *ProductModuleService) updateProducts(ctx context.Context, input []*dal.Record) ([]*dal.Record, error) {
	data := make([]*dal.Record, len(input))
	for i, p := range input {
		if p.Has("options") {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Updating product options through the product is not supported yet")
		}
		data[i] = clone(p)
	}
	var allOptionIDs []string
	for _, p := range data {
		ids, _ := p.Get("option_ids")
		for _, id := range list(ids) {
			if sid := fmt.Sprint(id); id != nil && sid != "" {
				allOptionIDs = append(allOptionIDs, sid)
			}
		}
	}
	var expected map[string]map[string]bool
	var unlinkPairs []ProductOptionProductPair
	if len(allOptionIDs) > 0 || anyHas(data, "option_ids") {
		f := dal.NewRecord()
		f.Set("id", productIDsOfRecords(data))
		originals, err := s.Internal("Product").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"options", "options.values", "tags"}})
		if err != nil {
			return nil, err
		}
		if len(allOptionIDs) > 0 {
			of := dal.NewRecord()
			of.Set("id", toAnyStrings(allOptionIDs))
			existingOptions, err := s.Internal("ProductOption").List(ctx, of, utilsmodulessdk.FindConfig{})
			if err != nil {
				return nil, err
			}
			if len(existingOptions) != len(allOptionIDs) {
				found := map[string]bool{}
				for _, o := range existingOptions {
					found[strOf(o, "id")] = true
				}
				var missing []string
				for _, id := range allOptionIDs {
					if !found[id] {
						missing = append(missing, id)
					}
				}
				if len(missing) > 0 {
					return nil, invalid("Some product options were not found: [%s]", strings.Join(missing, ", "))
				}
			}
		}
		var linkPairs []optionPair
		linkPairs, unlinkPairs, expected = computeOptionLinkChanges(data, originals)
		for _, p := range data {
			p.Delete("option_ids")
		}
		if len(linkPairs) > 0 {
			if err := s.addProductOptionToProduct(ctx, linkPairs); err != nil {
				return nil, err
			}
		}
		dal.FlushEvents(s.db)
	}
	normalized := make([]*dal.Record, len(data))
	for i, p := range data {
		normalized[i] = normalizeUpdateProductInput(p)
		if h := strOf(normalized[i], "handle"); h != "" && !IsValidHandle(h) {
			return nil, invalid("Invalid product handle '%s'. It must contain URL safe characters", h)
		}
	}

	idList := make([]string, len(normalized))
	for i, p := range normalized {
		idList[i] = strOf(p, "id")
	}
	relations := map[string]bool{}
	for _, p := range normalized {
		for _, k := range []string{"tags", "categories", "images", "collection", "type"} {
			if p.Has(k) {
				relations[k] = true
			}
		}
	}
	var relList []string
	for _, k := range []string{"tags", "categories", "images", "collection", "type"} {
		if relations[k] {
			relList = append(relList, k)
		}
	}
	existing, err := s.findByIDsWithSplitPopulate(ctx, idList, relList)
	if err != nil {
		return nil, err
	}
	byID := map[string]*dal.Record{}
	for _, p := range existing {
		byID[strOf(p, "id")] = p
	}
	var missing []string
	for _, id := range idList {
		if byID[id] == nil {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Unable to update the products with ids: %s", strings.Join(missing, ", ")))
	}

	var scalarUpdates []any
	for _, p := range normalized {
		id, _ := p.Get("id")
		row := dal.NewRecord()
		row.Set("id", id)
		for _, k := range p.Keys() {
			if productRelationKeys[k] {
				continue
			}
			v, _ := p.Get(k)
			row.Set(k, v)
		}
		scalarUpdates = append(scalarUpdates, row)

		if tags, ok := p.Get("tags"); ok {
			if err := s.Internal("Product").ReplaceManyToMany(ctx, id, "tags", productIDsOf(tags)); err != nil {
				return nil, err
			}
		}
		if cats, ok := p.Get("categories"); ok {
			if err := s.Internal("Product").ReplaceManyToMany(ctx, id, "categories", productIDsOf(cats)); err != nil {
				return nil, err
			}
		}
		if images, ok := p.Get("images"); ok {
			if err := s.replaceImages(ctx, id, records(images), byID[fmt.Sprint(id)]); err != nil {
				return nil, err
			}
		}
	}
	if _, err := s.Internal("Product").Update(ctx, scalarUpdates); err != nil {
		return nil, err
	}
	variantOrder := map[string][]string{}
	if anyHas(normalized, "variants") {
		valueIDsByProduct, err := s.getOptionValueIDsByProductIDs(ctx, toAnyStrings(idList))
		if err != nil {
			return nil, err
		}
		for _, p := range normalized {
			vs, ok := p.Get("variants")
			if !ok || vs == nil {
				continue
			}
			order, err := s.deepUpdateVariants(ctx, strOf(p, "id"), records(vs), expected[strOf(p, "id")], valueIDsByProduct[strOf(p, "id")])
			if err != nil {
				return nil, err
			}
			variantOrder[strOf(p, "id")] = order
		}
	}
	if len(unlinkPairs) > 0 {
		validated := map[string]bool{}
		for pid := range expected {
			validated[pid] = true
		}
		if _, err := s.removeProductOptionFromProduct(ctx, unlinkPairs, validated); err != nil {
			return nil, err
		}
	}
	out, err := s.findByIDsWithSplitPopulate(ctx, idList, append([]string{"options", "options.values", "variants", "images", "tags"}, relList...))
	if err != nil {
		return nil, err
	}
	// the assigned variants collection keeps the input order
	for _, p := range out {
		order, ok := variantOrder[strOf(p, "id")]
		if !ok {
			continue
		}
		cur, _ := p.Get("variants")
		byID := map[string]any{}
		for _, v := range list(cur) {
			byID[strOf(recordOf(v), "id")] = v
		}
		ordered := make([]any, 0, len(order))
		for _, id := range order {
			if v, ok := byID[id]; ok {
				ordered = append(ordered, v)
			}
		}
		p.Set("variants", ordered)
	}
	return out, nil
}

// replaceImages mirrors assigning images (with rank = index) to the
// product: existing images are updated, new ones created and the removed
// ones deleted (orphanRemoval).
func (s *ProductModuleService) replaceImages(ctx context.Context, productID any, images []*dal.Record, product *dal.Record) error {
	current := map[string]bool{}
	if product != nil {
		cur, _ := product.Get("images")
		for _, img := range records(cur) {
			current[strOf(img, "id")] = true
		}
	}
	kept := map[string]bool{}
	var updates []any
	var creates []*dal.Record
	for i, img := range images {
		id := strOf(img, "id")
		if id != "" && current[id] {
			kept[id] = true
			u := clone(img)
			u.Set("rank", float64(i))
			updates = append(updates, u)
			continue
		}
		c := clone(img)
		c.Delete("id")
		c.Set("rank", float64(i))
		c.Set("product_id", productID)
		creates = append(creates, c)
	}
	var removed []any
	for id := range current {
		if !kept[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		if _, err := s.Internal("ProductImage").Delete(ctx, removed); err != nil {
			return err
		}
	}
	if len(updates) > 0 {
		if _, err := s.Internal("ProductImage").Update(ctx, updates); err != nil {
			return err
		}
	}
	if len(creates) > 0 {
		if _, err := s.Internal("ProductImage").Create(ctx, creates); err != nil {
			return err
		}
	}
	return nil
}

// UpdateProductVariants mirrors updateProductVariants(idOrSelector, data).
func (s *ProductModuleService) UpdateProductVariants(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var input []*dal.Record
		if id, ok := idOrSelector.(string); ok {
			input = []*dal.Record{withID(id, data)}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			rows, err := t.Internal("ProductVariant").List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, r := range rows {
				id, _ := r.Get("id")
				input = append(input, withID(id, data))
			}
		}
		var err error
		out, err = t.updateVariants(ctx, input)
		return err
	})
	return out, err
}

// UpsertProductVariants mirrors upsertProductVariants.
func (s *ProductModuleService) UpsertProductVariants(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	// @EmitEvents runs inside the transaction upstream (before the flush):
	// the entity mutations emit no events
	ctx = dal.WithoutEvents(ctx)
	var forCreate, forUpdate []*dal.Record
	for _, v := range data {
		if id, _ := v.Get("id"); id != nil && id != "" {
			forUpdate = append(forUpdate, v)
		} else {
			forCreate = append(forCreate, v)
		}
	}
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		if len(forCreate) > 0 {
			created, err := t.createVariants(ctx, forCreate)
			if err != nil {
				return err
			}
			out = append(out, created...)
		}
		if len(forUpdate) > 0 {
			updated, err := t.updateVariants(ctx, forUpdate)
			if err != nil {
				return err
			}
			out = append(out, updated...)
		}
		return nil
	})
	return out, err
}

// updateVariants mirrors updateVariants_.
func (s *ProductModuleService) updateVariants(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	hasOptions := false
	ids := make([]any, len(data))
	for i, d := range data {
		ids[i], _ = d.Get("id")
		if o, ok := d.Get("options"); ok && isPresentValue(o) {
			hasOptions = true
		}
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	variants, err := s.Internal("ProductVariant").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(variants) != len(data) {
		found := map[string]bool{}
		for _, v := range variants {
			found[strOf(v, "id")] = true
		}
		var missing []string
		for _, id := range ids {
			if !found[fmt.Sprint(id)] {
				missing = append(missing, fmt.Sprint(id))
			}
		}
		return nil, invalid("Cannot update non-existing variants with ids: %s", strings.Join(missing, ", "))
	}
	withProduct := make([]*dal.Record, 0, len(variants))
	for _, v := range variants {
		var update *dal.Record
		for _, d := range data {
			if fmt.Sprint(func() any { x, _ := d.Get("id"); return x }()) == strOf(v, "id") {
				update = d
				break
			}
		}
		r := dal.NewRecord()
		for _, k := range update.Keys() {
			if k == "options" || k == "id" {
				continue
			}
			x, _ := update.Get(k)
			r.Set(k, x)
		}
		if o, ok := update.Get("options"); ok && isPresentValue(o) {
			r.Set("options", o)
		}
		r.Set("id", strOf(v, "id"))
		pid, _ := v.Get("product_id")
		r.Set("product_id", pid)
		withProduct = append(withProduct, r)
	}
	if hasOptions {
		var pids []any
		seen := map[string]bool{}
		for _, v := range variants {
			pid := strOf(v, "product_id")
			if !seen[pid] {
				seen[pid] = true
				pids = append(pids, pid)
			}
		}
		pf := dal.NewRecord()
		pf.Set("product_id", pids)
		all, err := s.Internal("ProductVariant").List(ctx, pf, utilsmodulessdk.FindConfig{Relations: []string{"options"}})
		if err != nil {
			return nil, err
		}
		optionsByProduct, valueIDsByProduct, err := s.loadOptionsAndValuesByProductID(ctx, pids)
		if err != nil {
			return nil, err
		}
		if withProduct, err = s.assignOptionsToVariants(withProduct, all, optionsByProduct, valueIDsByProduct); err != nil {
			return nil, err
		}
	}
	var updates []any
	for _, v := range withProduct {
		u := clone(v)
		u.Delete("options")
		updates = append(updates, u)
	}
	res, err := s.Internal("ProductVariant").Update(ctx, updates)
	if err != nil {
		return nil, err
	}
	if hasOptions {
		for _, v := range withProduct {
			o, ok := v.Get("options")
			if !ok {
				continue
			}
			if err := s.Internal("ProductVariant").ReplaceManyToMany(ctx, strOf(v, "id"), "options", productIDsOf(o)); err != nil {
				return nil, err
			}
		}
	}
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}, nil
	}
	return records(res), nil
}

// isPresentValue mirrors isPresent: not null/undefined and not an empty
// object, array or string.
func isPresentValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return t != nil && t.Len() > 0
	}
	return true
}

func anyHas(list []*dal.Record, key string) bool {
	for _, r := range list {
		if r.Has(key) {
			return true
		}
	}
	return false
}

func productIDsOfRecords(list []*dal.Record) []any {
	out := make([]any, len(list))
	for i, r := range list {
		out[i], _ = r.Get("id")
	}
	return out
}

func toAnyStrings(list []string) []any {
	out := make([]any, len(list))
	for i, s := range list {
		out[i] = s
	}
	return out
}

// computeOptionLinkChanges mirrors computeOptionLinkChanges.
func computeOptionLinkChanges(data []*dal.Record, originals []*dal.Record) ([]optionPair, []ProductOptionProductPair, map[string]map[string]bool) {
	var link []optionPair
	var unlink []ProductOptionProductPair
	expected := map[string]map[string]bool{}
	for _, p := range data {
		raw, ok := p.Get("option_ids")
		if !ok || raw == nil {
			continue
		}
		pid := strOf(p, "id")
		var newIDs []string
		newSet := map[string]bool{}
		for _, id := range list(raw) {
			sid := fmt.Sprint(id)
			if !newSet[sid] {
				newSet[sid] = true
				newIDs = append(newIDs, sid)
			}
		}
		if p.Has("variants") {
			expected[pid] = newSet
		}
		var existingIDs []string
		existingSet := map[string]bool{}
		for _, o := range originals {
			if strOf(o, "id") != pid {
				continue
			}
			opts, _ := o.Get("options")
			for _, opt := range records(opts) {
				id := strOf(opt, "id")
				if !existingSet[id] {
					existingSet[id] = true
					existingIDs = append(existingIDs, id)
				}
			}
		}
		for _, id := range newIDs {
			if !existingSet[id] {
				link = append(link, optionPair{productID: pid, optionID: id})
			}
		}
		for _, id := range existingIDs {
			if !newSet[id] {
				unlink = append(unlink, ProductOptionProductPair{ProductID: pid, ProductOptionID: id})
			}
		}
	}
	return link, unlink, expected
}

// deepUpdateVariants mirrors the variants part of deepUpdate: validate the
// variants' options against the product (restricted to the expected
// options when option_ids were given), map { title: value } to option value
// ids, then assign the collection (orphanRemoval deletes the variants
// missing from the input).
func (s *ProductModuleService) deepUpdateVariants(ctx context.Context, productID string, variants []*dal.Record, expected map[string]bool, allowed map[string]bool) ([]string, error) {
	optionsByProduct, _, err := s.loadOptionsAndValuesByProductID(ctx, []any{productID})
	if err != nil {
		return nil, err
	}
	productOptions := optionsByProduct[productID]
	forValidation := productOptions
	if expected != nil {
		forValidation = nil
		for _, o := range productOptions {
			if expected[strOf(o, "id")] {
				forValidation = append(forValidation, o)
			}
		}
	}
	input := make([]*dal.Record, len(variants))
	for i, v := range variants {
		c := clone(v)
		if o, ok := c.Get("options"); ok && !isPresentValue(o) {
			c.Delete("options")
		}
		c.Set("product_id", productID)
		input[i] = c
	}
	validated, err := s.assignOptionsToVariants(input, nil, map[string][]*dal.Record{productID: forValidation}, map[string]map[string]bool{productID: allowed})
	if err != nil {
		return nil, err
	}
	if err := checkIfVariantsHaveUniqueOptionsCombinations(validated); err != nil {
		return nil, err
	}
	// options { title: value } -> option value ids among all the product's options
	for _, v := range input {
		vo := recordOf(func() any { x, _ := v.Get("options"); return x }())
		if vo == nil {
			continue
		}
		var ids []any
		for _, key := range vo.Keys() {
			val, _ := vo.Get(key)
			var option *dal.Record
			for _, o := range productOptions {
				if strOf(o, "title") == key {
					option = o
					break
				}
			}
			var values []any
			if option != nil {
				vals, _ := option.Get("values")
				values = list(vals)
			}
			ov, err := resolveAllowedOptionValues(key, val, values, allowed)
			if err != nil {
				return nil, err
			}
			r := dal.NewRecord()
			r.Set("id", strOf(ov, "id"))
			ids = append(ids, r)
		}
		v.Set("options", ids)
	}

	f := dal.NewRecord()
	f.Set("product_id", productID)
	existing, err := s.Internal("ProductVariant").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	existingIDs := map[string]bool{}
	for _, e := range existing {
		existingIDs[strOf(e, "id")] = true
	}
	kept := map[string]bool{}
	var creates []*dal.Record
	var updates []any
	type optionReplace struct {
		id  string
		ids []any
	}
	var replaces []optionReplace
	for _, v := range input {
		id := strOf(v, "id")
		if id == "" {
			creates = append(creates, v)
			continue
		}
		kept[id] = true
		u := clone(v)
		if o, ok := u.Get("options"); ok {
			replaces = append(replaces, optionReplace{id: id, ids: productIDsOf(o)})
			u.Delete("options")
		}
		updates = append(updates, u)
	}
	var orphans []any
	for _, e := range existing {
		if id := strOf(e, "id"); !kept[id] {
			orphans = append(orphans, id)
		}
	}
	if len(orphans) > 0 {
		if _, err := s.Internal("ProductVariant").Delete(ctx, orphans); err != nil {
			return nil, err
		}
	}
	createdIDs := map[*dal.Record]string{}
	if len(creates) > 0 {
		created, err := s.Internal("ProductVariant").Create(ctx, creates)
		if err != nil {
			return nil, err
		}
		for i, c := range created {
			createdIDs[creates[i]] = strOf(c, "id")
		}
	}
	if len(updates) > 0 {
		if _, err := s.Internal("ProductVariant").Update(ctx, updates); err != nil {
			return nil, err
		}
	}
	for _, r := range replaces {
		if err := s.Internal("ProductVariant").ReplaceManyToMany(ctx, r.id, "options", r.ids); err != nil {
			return nil, err
		}
	}
	_ = existingIDs
	order := make([]string, len(input))
	for i, v := range input {
		if id := strOf(v, "id"); id != "" {
			order[i] = id
		} else {
			order[i] = createdIDs[v]
		}
	}
	return order, nil
}

// checkIfVariantsHaveUniqueOptionsCombinations mirrors
// checkIfVariantsHaveUniqueOptionsCombinations.
func checkIfVariantsHaveUniqueOptionsCombinations(variants []*dal.Record) error {
	for i, v := range variants {
		vo, ok := v.Get("options")
		if !ok || vo == nil {
			continue
		}
		for j := i + 1; j < len(variants); j++ {
			c := variants[j]
			co, _ := c.Get("options")
			have := map[string]bool{}
			for _, x := range list(co) {
				have[strOf(recordOf(x), "id")] = true
			}
			all := true
			for _, x := range list(vo) {
				if !have[strOf(recordOf(x), "id")] {
					all = false
					break
				}
			}
			if all {
				return invalid("Variant \"%s\" has same combination of option values as \"%s\".", strOf(v, "title"), strOf(c, "title"))
			}
		}
	}
	return nil
}
