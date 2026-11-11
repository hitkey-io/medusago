package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/oklog/ulid/v2"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

func invalid(format string, args ...any) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(format, args...))
}

func clone(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

func list(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out
	}
	return nil
}

func recordOf(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

var (
	handleStrip  = regexp.MustCompile(`[^\p{L}\p{N}\s_-]`)
	handleSpaces = regexp.MustCompile(`[\s_]+`)
	handleDashes = regexp.MustCompile(`-+`)
	validHandle  = regexp.MustCompile(`^[\p{Ll}\p{Lo}\p{Lm}\p{N}]+(?:-[\p{Ll}\p{Lo}\p{Lm}\p{N}]+)*$`)
	upperASCII   = regexp.MustCompile(`[A-Z]`)
)

// ToHandle mirrors toHandle.
func ToHandle(value string) string {
	h := strings.ToLower(value)
	h = strings.ReplaceAll(h, "ß", "ss")
	h = handleStrip.ReplaceAllString(h, "")
	h = handleSpaces.ReplaceAllString(h, "-")
	h = handleDashes.ReplaceAllString(h, "-")
	h = strings.Trim(h, "-")
	if h == "" {
		h = "product-" + strings.ToLower(ulid.Make().String()[20:26])
	}
	return h
}

// IsValidHandle mirrors isValidHandle.
func IsValidHandle(value string) bool {
	if value == "" || upperASCII.MatchString(value) {
		return false
	}
	return validHandle.MatchString(value)
}

// normalizeUpdateProductInput mirrors normalizeUpdateProductInput.
func normalizeUpdateProductInput(p *dal.Record) *dal.Record {
	d := clone(p)
	if gc, _ := d.Get("is_giftcard"); gc == true {
		d.Set("discountable", false)
	}
	if ids, ok := d.Get("tag_ids"); ok && ids != nil {
		var tags []any
		for _, id := range list(ids) {
			r := dal.NewRecord()
			r.Set("id", id)
			tags = append(tags, r)
		}
		d.Delete("tag_ids")
		d.Set("tags", orEmpty(tags))
	}
	if ids, ok := d.Get("category_ids"); ok && ids != nil {
		var cats []any
		for _, id := range list(ids) {
			r := dal.NewRecord()
			r.Set("id", id)
			cats = append(cats, r)
		}
		d.Delete("category_ids")
		d.Set("categories", orEmpty(cats))
	}
	return d
}

func orEmpty(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}

// normalizeCreateProductInput mirrors normalizeCreateProductInput.
func normalizeCreateProductInput(p *dal.Record) *dal.Record {
	d := normalizeUpdateProductInput(p)
	if opts, ok := d.Get("options"); ok && len(list(opts)) > 0 {
		var out []any
		for _, o := range list(opts) {
			opt := recordOf(o)
			n := dal.NewRecord()
			title, _ := opt.Get("title")
			n.Set("title", title)
			if vals, ok := opt.Get("values"); ok && vals != nil {
				var vs []any
				for _, v := range list(vals) {
					r := dal.NewRecord()
					r.Set("value", v)
					vs = append(vs, r)
				}
				n.Set("values", orEmpty(vs))
			}
			excl, ok := opt.Get("is_exclusive")
			if !ok || excl == nil {
				excl = true
			}
			n.Set("is_exclusive", excl)
			if id, _ := opt.Get("id"); id != nil && id != "" {
				n.Set("id", id)
			}
			if vids, _ := opt.Get("value_ids"); vids != nil {
				n.Set("value_ids", vids)
			}
			out = append(out, n)
		}
		d.Set("options", out)
	}
	if h, _ := d.Get("handle"); h == nil || h == "" {
		if t, _ := d.Get("title"); t != nil && t != "" {
			d.Set("handle", ToHandle(fmt.Sprint(t)))
		}
	}
	if s, _ := d.Get("status"); s == nil || s == "" {
		d.Set("status", "draft")
	}
	imgs, _ := d.Get("images")
	images := list(imgs)
	if t, _ := d.Get("thumbnail"); (t == nil || t == "") && len(images) > 0 {
		url, _ := recordOf(images[0]).Get("url")
		d.Set("thumbnail", url)
	}
	if len(images) > 0 {
		out := make([]any, len(images))
		for i, img := range images {
			r := recordOf(img)
			if rank, _ := r.Get("rank"); rank != nil {
				out[i] = r
				continue
			}
			c := clone(r)
			c.Set("rank", float64(i))
			out[i] = c
		}
		d.Set("images", out)
	}
	return d
}

func strOf(r *dal.Record, k string) string {
	v, _ := r.Get(k)
	s, _ := v.(string)
	return s
}

// validateProductCreatePayload mirrors validateProductCreatePayload.
func validateProductCreatePayload(p *dal.Record) error {
	if h := strOf(p, "handle"); h != "" && !IsValidHandle(h) {
		return invalid("Invalid product handle '%s'. It must contain URL safe characters", h)
	}
	title := strOf(p, "title")
	if title == "" {
		return invalid("Product title is required")
	}
	opts, _ := p.Get("options")
	options := list(opts)
	seen := map[string]bool{}
	var dup []string
	for _, o := range options {
		if id := strOf(recordOf(o), "id"); id != "" {
			if seen[id] {
				dup = append(dup, id)
			}
			seen[id] = true
		}
	}
	if len(dup) > 0 {
		return invalid(`Product "%s" has duplicate option assignments: [%s]`, title, strings.Join(dup, ", "))
	}
	var missing []string
	if len(options) > 0 {
		vs, _ := p.Get("variants")
		for _, v := range list(vs) {
			vr := recordOf(v)
			vo := recordOf(func() any { x, _ := vr.Get("options"); return x }())
			for _, o := range options {
				t := strOf(recordOf(o), "title")
				val, _ := vo.Get(t)
				if vo == nil || val == nil || val == "" {
					missing = append(missing, strOf(vr, "title"))
				}
			}
		}
	}
	if len(missing) > 0 {
		return invalid(`Product "%s" has variants with missing options: [%s]`, title, strings.Join(missing, ", "))
	}
	return nil
}

// resolveAllowedOptionValues mirrors resolveAllowedOptionValues.
func resolveAllowedOptionValues(optionTitle string, value any, optionValues []any, allowed map[string]bool) (*dal.Record, error) {
	valueName := func(v *dal.Record) any {
		x, _ := v.Get("value")
		if r, ok := x.(*dal.Record); ok {
			y, _ := r.Get("value")
			return y
		}
		return x
	}
	idOf := func(v *dal.Record) string {
		x, _ := v.Get("value")
		if r, ok := x.(*dal.Record); ok {
			if id := strOf(r, "id"); id != "" {
				return id
			}
		}
		return strOf(v, "id")
	}
	for _, o := range optionValues {
		v := recordOf(o)
		if v == nil {
			continue
		}
		if allowed != nil {
			if id := idOf(v); id != "" && !allowed[id] {
				continue
			}
		}
		if valueName(v) == value {
			return v, nil
		}
	}
	return nil, invalid("Option value %v does not exist for option %s", value, optionTitle)
}

// CreateProducts mirrors createProducts (the variants are created by
// createProductVariants).
func (s *ProductModuleService) CreateProducts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		products, err := t.createProducts(ctx, data)
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

func (s *ProductModuleService) createProducts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	// options referenced by id
	var existingOptionIDs []any
	for _, p := range data {
		opts, _ := p.Get("options")
		for _, o := range list(opts) {
			if id := strOf(recordOf(o), "id"); id != "" {
				existingOptionIDs = append(existingOptionIDs, id)
			}
		}
	}
	existingOptions := map[string]*dal.Record{}
	if len(existingOptionIDs) > 0 {
		f := dal.NewRecord()
		f.Set("id", existingOptionIDs)
		rows, err := s.Internal("ProductOption").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			existingOptions[strOf(r, "id")] = r
		}
		var missing []string
		for _, id := range existingOptionIDs {
			if existingOptions[id.(string)] == nil {
				missing = append(missing, id.(string))
			}
		}
		if len(missing) > 0 {
			return nil, invalid("Some product options were not found: [%s]", strings.Join(missing, ", "))
		}
	}
	hydrated := make([]*dal.Record, len(data))
	for i, p := range data {
		opts, _ := p.Get("options")
		if len(list(opts)) == 0 {
			hydrated[i] = p
			continue
		}
		var hopts []any
		for _, o := range list(opts) {
			opt := recordOf(o)
			id := strOf(opt, "id")
			if id == "" {
				hopts = append(hopts, opt)
				continue
			}
			db := existingOptions[id]
			h := dal.NewRecord()
			h.Set("id", id)
			h.Set("title", strOf(db, "title"))
			vals, _ := db.Get("values")
			var vs []any
			for _, v := range list(vals) {
				vr := dal.NewRecord()
				vr.Set("id", strOf(recordOf(v), "id"))
				vv, _ := recordOf(v).Get("value")
				vr.Set("value", vv)
				vs = append(vs, vr)
			}
			h.Set("values", orEmpty(vs))
			vids, _ := opt.Get("value_ids")
			h.Set("value_ids", vids)
			hopts = append(hopts, h)
		}
		c := clone(p)
		c.Set("options", hopts)
		hydrated[i] = c
	}

	normalized := make([]*dal.Record, len(hydrated))
	for i, p := range hydrated {
		normalized[i] = normalizeCreateProductInput(p)
		if err := validateProductCreatePayload(normalized[i]); err != nil {
			return nil, err
		}
	}

	// tags must exist
	var tagIDs []any
	for _, p := range normalized {
		tags, _ := p.Get("tags")
		for _, t := range list(tags) {
			tagIDs = append(tagIDs, strOf(recordOf(t), "id"))
		}
	}
	existingTags := map[string]bool{}
	if len(tagIDs) > 0 {
		f := dal.NewRecord()
		f.Set("id", tagIDs)
		rows, err := s.Internal("ProductTag").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			existingTags[strOf(r, "id")] = true
		}
	}

	type pendingOption struct {
		productID string
		options   []*dal.Record
	}
	var optionsToCreate []pendingOption
	var toCreate []*dal.Record
	productIDs := make([]string, len(normalized))
	for i, p := range normalized {
		pid := common.GenerateEntityID(strOf(p, "id"), "prod")
		p.Set("id", pid)
		productIDs[i] = pid
		if opts, _ := p.Get("options"); len(list(opts)) > 0 {
			var newOpts []*dal.Record
			for _, o := range list(opts) {
				if strOf(recordOf(o), "id") == "" {
					newOpts = append(newOpts, recordOf(o))
				}
			}
			if len(newOpts) > 0 {
				optionsToCreate = append(optionsToCreate, pendingOption{productID: pid, options: newOpts})
			}
		}
		if vs, _ := p.Get("variants"); len(list(vs)) > 0 {
			return nil, invalid("Creating variants through createProducts is not supported yet")
		}
		if tags, _ := p.Get("tags"); len(list(tags)) > 0 {
			for _, t := range list(tags) {
				id := strOf(recordOf(t), "id")
				if !existingTags[id] {
					return nil, invalid("Tag with id %s not found. Please create the tag before associating it with the product.", id)
				}
			}
		}
		c := clone(p)
		c.Delete("options")
		toCreate = append(toCreate, c)
	}

	if _, err := s.Internal("Product").Create(ctx, toCreate); err != nil {
		return nil, err
	}

	// options created with the products
	optionIDsByProduct := map[string][]string{}
	var allOptions []*dal.Record
	for _, po := range optionsToCreate {
		for _, o := range po.options {
			c := clone(o)
			id := common.GenerateEntityID("", "opt")
			c.Set("id", id)
			optionIDsByProduct[po.productID] = append(optionIDsByProduct[po.productID], id)
			allOptions = append(allOptions, c)
		}
	}
	if len(allOptions) > 0 {
		if _, err := s.createOptions(ctx, allOptions); err != nil {
			return nil, err
		}
	}
	var pairs []optionPair
	for i, pid := range productIDs {
		opts, _ := hydrated[i].Get("options")
		for _, o := range list(opts) {
			opt := recordOf(o)
			if id := strOf(opt, "id"); id != "" {
				vids, _ := opt.Get("value_ids")
				pairs = append(pairs, optionPair{productID: pid, optionID: id, valueIDs: list(vids), hasValueIDs: vids != nil})
			}
		}
		for _, id := range optionIDsByProduct[pid] {
			pairs = append(pairs, optionPair{productID: pid, optionID: id})
		}
	}
	if len(pairs) > 0 {
		// upstream flushes the created options before linking them
		if len(allOptions) > 0 {
			dal.FlushEvents(s.db)
		}
		if err := s.addProductOptionToProduct(ctx, pairs); err != nil {
			return nil, err
		}
	}
	dal.FlushEvents(s.db)
	return s.findByIDsWithSplitPopulate(ctx, productIDs, []string{"options", "options.values", "variants", "images", "tags"})
}

// createOptions mirrors createOptions_.
func (s *ProductModuleService) createOptions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	normalized := make([]*dal.Record, len(data))
	for i, opt := range data {
		ranks := recordOf(func() any { x, _ := opt.Get("ranks"); return x }())
		vals, _ := opt.Get("values")
		values := list(vals)
		if ranks != nil {
			for _, k := range ranks.Keys() {
				found := false
				for _, v := range values {
					if v == k {
						found = true
					}
				}
				if !found {
					return nil, invalid(`Value "%s" is assigned a rank but is not defined in the list of values.`, k)
				}
			}
		}
		c := clone(opt)
		c.Delete("ranks")
		if vals != nil {
			out := make([]any, len(values))
			for j, v := range values {
				var vo *dal.Record
				if s, ok := v.(string); ok {
					vo = dal.NewRecord()
					vo.Set("value", s)
				} else {
					vo = clone(recordOf(v))
				}
				if ranks != nil {
					if rank, ok := ranks.Get(strOf(vo, "value")); ok {
						vo.Set("rank", rank)
					}
				}
				out[j] = vo
			}
			c.Set("values", out)
		}
		normalized[i] = c
	}
	return s.Internal("ProductOption").Create(ctx, normalized)
}

type optionPair struct {
	productID, optionID string
	valueIDs            []any
	hasValueIDs         bool
}

// addProductOptionToProduct mirrors addProductOptionToProduct_.
func (s *ProductModuleService) addProductOptionToProduct(ctx context.Context, pairs []optionPair) error {
	var optionIDs []string
	seenOpt := map[string]bool{}
	for _, p := range pairs {
		if !seenOpt[p.optionID] {
			seenOpt[p.optionID] = true
			optionIDs = append(optionIDs, p.optionID)
		}
	}
	rows, err := dal.QueryRows(ctx, s.db, `select "id", "option_id" from "product_option_value" where "option_id" = any($1) and "deleted_at" is null`, optionIDs)
	if err != nil {
		return err
	}
	valuesByOption := map[string][]any{}
	for _, r := range rows {
		valuesByOption[fmt.Sprint(r["option_id"])] = append(valuesByOption[fmt.Sprint(r["option_id"])], r["id"])
	}
	if err := s.canAssignProductOptionToProduct(ctx, pairs, optionIDs); err != nil {
		return err
	}
	ppos := make([]*dal.Record, len(pairs))
	for i, p := range pairs {
		r := dal.NewRecord()
		r.Set("product_id", p.productID)
		r.Set("product_option_id", p.optionID)
		ppos[i] = r
	}
	created, err := s.Internal("ProductProductOption").Create(ctx, ppos)
	if err != nil {
		return err
	}
	ppoID := map[string]any{}
	for _, c := range created {
		ppoID[strOf(c, "product_id")+"_"+strOf(c, "product_option_id")] = strOf(c, "id")
	}
	var ppovs []*dal.Record
	for _, p := range pairs {
		id, ok := ppoID[p.productID+"_"+p.optionID]
		if !ok {
			continue
		}
		vids := p.valueIDs
		if !p.hasValueIDs {
			vids = valuesByOption[p.optionID]
		}
		for _, v := range vids {
			r := dal.NewRecord()
			r.Set("product_product_option_id", id)
			r.Set("product_option_value_id", v)
			ppovs = append(ppovs, r)
		}
	}
	if len(ppovs) > 0 {
		if _, err := s.Internal("ProductProductOptionValue").Create(ctx, ppovs); err != nil {
			return err
		}
	}
	return nil
}

// canAssignProductOptionToProduct mirrors
// ProductRepository.canAssignProductOptionToProduct.
func (s *ProductModuleService) canAssignProductOptionToProduct(ctx context.Context, pairs []optionPair, optionIDs []string) error {
	seen := map[string]bool{}
	var dups []string
	for _, p := range pairs {
		k := p.productID + "_" + p.optionID
		if seen[k] {
			dups = append(dups, k)
		}
		seen[k] = true
	}
	if len(dups) > 0 {
		return invalid("Duplicate product option assignments are not allowed; remove duplicate pairs: %s", strings.Join(dups, ", "))
	}
	var ph []string
	var args []any
	for _, p := range pairs {
		args = append(args, p.productID, p.optionID)
		ph = append(ph, fmt.Sprintf("($%d, $%d)", len(args)-1, len(args)))
	}
	linked, err := dal.QueryRows(ctx, s.db, `SELECT DISTINCT product_option_id as option_id FROM product_product_option WHERE (product_id, product_option_id) IN (`+strings.Join(ph, ", ")+`) AND deleted_at IS NULL`, args...)
	if err != nil {
		return err
	}
	exclusive, err := dal.QueryRows(ctx, s.db, `WITH input_pairs(product_id, option_id) AS (VALUES `+strings.Join(ph, ", ")+`)
       SELECT DISTINCT po.id as option_id
       FROM product_option po
       WHERE po.is_exclusive = true
         AND po.id IN (SELECT option_id FROM input_pairs)
         AND (
           (SELECT COUNT(DISTINCT ip.product_id) FROM input_pairs ip WHERE ip.option_id = po.id) > 1
           OR EXISTS (
             SELECT 1 FROM product_product_option ppo
             WHERE ppo.product_option_id = po.id
               AND ppo.deleted_at IS NULL
               AND ppo.product_id NOT IN (SELECT ip.product_id FROM input_pairs ip WHERE ip.option_id = po.id)
           )
         )`, args...)
	if err != nil {
		return err
	}
	in := func(rows []dal.Row) []string {
		set := map[string]bool{}
		for _, r := range rows {
			set[fmt.Sprint(r["option_id"])] = true
		}
		var out []string
		for _, id := range optionIDs {
			if set[id] {
				out = append(out, id)
			}
		}
		return out
	}
	if l := in(linked); len(l) > 0 {
		return invalid("Product options are already linked to products: %s", strings.Join(l, ", "))
	}
	if e := in(exclusive); len(e) > 0 {
		return invalid("Product options are already assigned to another product: %s", strings.Join(e, ", "))
	}
	return nil
}

// findByIDsWithSplitPopulate mirrors findByIdsWithSplitPopulate: the
// products in the order of the ids with the relations populated.
func (s *ProductModuleService) findByIDsWithSplitPopulate(ctx context.Context, ids []string, relations []string) ([]*dal.Record, error) {
	if len(ids) == 0 {
		return []*dal.Record{}, nil
	}
	f := dal.NewRecord()
	f.Set("id", list(ids))
	take := len(ids)
	rows, err := s.Internal("Product").Repository.Find(ctx, dal.FindOptions{Where: f, Relations: relations, Take: &take})
	if err != nil {
		return nil, err
	}
	pos := map[string]int{}
	for i, id := range ids {
		pos[id] = i
	}
	sort.SliceStable(rows, func(i, j int) bool { return pos[strOf(rows[i], "id")] < pos[strOf(rows[j], "id")] })
	return rows, nil
}

// CreateProductVariants mirrors createProductVariants.
func (s *ProductModuleService) CreateProductVariants(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		v, err := t.createVariants(ctx, data)
		out = v
		return err
	})
	return out, err
}

func (s *ProductModuleService) createVariants(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var productIDs []any
	seen := map[string]bool{}
	for _, v := range data {
		pid := strOf(v, "product_id")
		if pid == "" {
			return nil, invalid("Unable to create variants without specifying a product_id")
		}
		if !seen[pid] {
			seen[pid] = true
			productIDs = append(productIDs, pid)
		}
	}
	f := dal.NewRecord()
	f.Set("product_id", productIDs)
	existing, err := s.Internal("ProductVariant").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"options"}})
	if err != nil {
		return nil, err
	}
	optionsByProduct, valueIDsByProduct, err := s.loadOptionsAndValuesByProductID(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	withOptions, err := s.assignOptionsToVariants(data, existing, optionsByProduct, valueIDsByProduct)
	if err != nil {
		return nil, err
	}
	return s.Internal("ProductVariant").Create(ctx, withOptions)
}

// loadOptionsAndValuesByProductID mirrors loadOptionsAndValuesByProductId_.
func (s *ProductModuleService) loadOptionsAndValuesByProductID(ctx context.Context, productIDs []any) (map[string][]*dal.Record, map[string]map[string]bool, error) {
	pf := dal.NewRecord()
	pf.Set("id", productIDs)
	f := dal.NewRecord()
	f.Set("products", pf)
	options, err := s.Internal("ProductOption").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
	if err != nil {
		return nil, nil, err
	}
	byID := map[string]*dal.Record{}
	for _, o := range options {
		byID[strOf(o, "id")] = o
	}
	rows, err := dal.QueryRows(ctx, s.db, `select "product_id", "product_option_id" from "product_product_option" where "product_id" = any($1) and "deleted_at" is null`, toStrings(productIDs))
	if err != nil {
		return nil, nil, err
	}
	optionsByProduct := map[string][]*dal.Record{}
	seen := map[string]bool{}
	for _, r := range rows {
		pid, oid := fmt.Sprint(r["product_id"]), fmt.Sprint(r["product_option_id"])
		if seen[pid+oid] {
			continue
		}
		seen[pid+oid] = true
		if o := byID[oid]; o != nil {
			optionsByProduct[pid] = append(optionsByProduct[pid], o)
		}
	}
	values, err := s.getOptionValueIDsByProductIDs(ctx, productIDs)
	if err != nil {
		return nil, nil, err
	}
	return optionsByProduct, values, nil
}

// assignOptionsToVariants mirrors assignOptionsToVariants and
// checkIfVariantWithOptionsAlreadyExists: options { title: value } become
// option value references.
func (s *ProductModuleService) assignOptionsToVariants(data []*dal.Record, existing []*dal.Record, optionsByProduct map[string][]*dal.Record, valueIDsByProduct map[string]map[string]bool) ([]*dal.Record, error) {
	withOptions := make([]*dal.Record, len(data))
	for i, v := range data {
		vo := recordOf(func() any { x, _ := v.Get("options"); return x }())
		n := 0
		if vo != nil {
			n = vo.Len()
		}
		pid := strOf(v, "product_id")
		opts := optionsByProduct[pid]
		if n > 0 && len(opts) != n {
			return nil, invalid("Product has %d option values but there were %d provided option values for the variant: %s.", len(opts), n, strOf(v, "title"))
		}
		if n == 0 {
			withOptions[i] = v
			continue
		}
		var ids []any
		for _, key := range vo.Keys() {
			val, _ := vo.Get(key)
			var option *dal.Record
			for _, o := range opts {
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
			ov, err := resolveAllowedOptionValues(key, val, values, valueIDsByProduct[pid])
			if err != nil {
				return nil, err
			}
			r := dal.NewRecord()
			r.Set("id", strOf(ov, "id"))
			ids = append(ids, r)
		}
		c := clone(v)
		c.Set("options", ids)
		withOptions[i] = c
	}
	// checkIfVariantWithOptionsAlreadyExists
	for _, v := range withOptions {
		vo, _ := v.Get("options")
		ids := list(vo)
		if len(ids) == 0 {
			continue
		}
		for _, ev := range existing {
			if strOf(ev, "product_id") != strOf(v, "product_id") {
				continue
			}
			if id := strOf(v, "id"); id != "" && id == strOf(ev, "id") {
				continue
			}
			eo, _ := ev.Get("options")
			have := map[string]bool{}
			for _, x := range list(eo) {
				have[strOf(recordOf(x), "id")] = true
			}
			all := true
			for _, x := range ids {
				if !have[strOf(recordOf(x), "id")] {
					all = false
					break
				}
			}
			if all {
				return nil, invalid("Variant (%s) with provided options already exists.", strOf(ev, "title"))
			}
		}
	}
	return withOptions, nil
}
