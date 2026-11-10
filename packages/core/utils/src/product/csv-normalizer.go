// Package product mirrors @medusajs/utils/src/product: the CSV import
// normalizer and the validators of the import flow.
package product

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Row is a CSV row: column -> value, in column order.
type Row = *dal.Record

func createError(rowNumber int, message string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Row %d: %s", rowNumber, message))
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// isPresent mirrors isPresent.
func isPresent(v any) bool {
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

// NormalizeCSVValue mirrors normalizeCSVValue: a literal "\r" suffix is
// dropped and the value trimmed.
func NormalizeCSVValue(v any) any {
	if s, ok := v.(string); ok {
		s = strings.TrimSuffix(s, `\r`)
		return strings.TrimFunc(s, isJSWhitespace)
	}
	return v
}

func isJSWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\v', '\f', 0xa0, 0xfeff, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

// tryConvertToBoolean mirrors tryConvertToBoolean(value, default).
func tryConvertToBoolean(v any, def any) any {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(t) {
		case "true":
			return true
		case "false":
			return false
		}
	}
	return def
}

// JSNumber mirrors Number(value) for strings, numbers and booleans.
func JSNumber(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, !math.IsNaN(t)
	case int:
		return float64(t), true
	case bool:
		if t {
			return 1, true
		}
		return 0, true
	case nil:
		return 0, true
	case string:
		s := strings.TrimFunc(t, isJSWhitespace)
		if s == "" {
			return 0, true
		}
		switch s {
		case "Infinity", "+Infinity":
			return math.Inf(1), true
		case "-Infinity":
			return math.Inf(-1), true
		}
		lower := strings.ToLower(s)
		for prefix, base := range map[string]int{"0x": 16, "0o": 8, "0b": 2} {
			if strings.HasPrefix(lower, prefix) {
				n, err := strconv.ParseUint(s[2:], base, 64)
				if err != nil {
					return 0, false
				}
				return float64(n), true
			}
		}
		if strings.ContainsAny(lower, "_xpnio") {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
				return f, true
			}
			return 0, false
		}
		return f, true
	}
	return 0, false
}

type processor func(row Row, columns []string, rowNumber int, out *dal.Record, ctx *Context) error

func asString(in, outKey string) processor {
	return func(row Row, _ []string, _ int, out *dal.Record, _ *Context) error {
		if v := get(row, in); isPresent(v) {
			out.Set(outKey, v)
		}
		return nil
	}
}

func asJSON(in, outKey string) processor {
	return func(row Row, _ []string, rowNumber int, out *dal.Record, _ *Context) error {
		v := get(row, in)
		s, ok := v.(string)
		if !isPresent(v) || !ok {
			return nil
		}
		if !json.Valid([]byte(s)) {
			return createError(rowNumber, fmt.Sprintf(`Invalid value provided for "%s". Expected a valid JSON string, received "%s"`, in, s))
		}
		parsed, err := dal.DecodeOrderedJSON([]byte(s))
		if err != nil {
			return createError(rowNumber, fmt.Sprintf(`Invalid value provided for "%s". Expected a valid JSON string, received "%s"`, in, s))
		}
		out.Set(outKey, parsed)
		return nil
	}
}

func ignored() processor {
	return func(Row, []string, int, *dal.Record, *Context) error { return nil }
}

func asBoolean(in, outKey string) processor {
	return func(row Row, _ []string, _ int, out *dal.Record, _ *Context) error {
		if v := get(row, in); isPresent(v) {
			out.Set(outKey, tryConvertToBoolean(v, v))
		}
		return nil
	}
}

func asNumber(in, outKey string) processor {
	return func(row Row, _ []string, rowNumber int, out *dal.Record, _ *Context) error {
		v := get(row, in)
		if !isPresent(v) {
			return nil
		}
		n, ok := JSNumber(v)
		if !ok {
			return createError(rowNumber, fmt.Sprintf(`Invalid value provided for "%s". Expected value to be a number, received "%v"`, in, v))
		}
		out.Set(outKey, n)
		return nil
	}
}

func asCounterValue(matcher *regexp.Regexp, itemKey, outKey string) processor {
	return func(row Row, columns []string, _ int, out *dal.Record, _ *Context) error {
		var matching []string
		for _, c := range columns {
			if matcher.MatchString(c) {
				matching = append(matching, c)
			}
		}
		if len(matching) == 0 {
			return nil
		}
		cur, _ := get(out, outKey).([]any)
		if cur == nil {
			cur = []any{}
		}
		existing := map[any]bool{}
		for _, item := range cur {
			existing[get(item.(*dal.Record), itemKey)] = true
		}
		for _, c := range matching {
			v := get(row, c)
			if !existing[v] && isPresent(v) {
				item := dal.NewRecord()
				item.Set(itemKey, v)
				cur = append(cur, item)
			}
		}
		out.Set(outKey, cur)
		return nil
	}
}

type column struct {
	name string
	fn   processor
}

var productStaticColumns = []column{
	{"product id", asString("product id", "id")},
	{"product handle", asString("product handle", "handle")},
	{"product title", asString("product title", "title")},
	{"product subtitle", asString("product subtitle", "subtitle")},
	{"product status", asString("product status", "status")},
	{"product description", asString("product description", "description")},
	{"product external id", asString("product external id", "external_id")},
	{"product thumbnail", asString("product thumbnail", "thumbnail")},
	{"product collection id", asString("product collection id", "collection_id")},
	{"product type id", asString("product type id", "type_id")},
	{"product discountable", asBoolean("product discountable", "discountable")},
	{"product height", asNumber("product height", "height")},
	{"product hs code", asString("product hs code", "hs_code")},
	{"product length", asNumber("product length", "length")},
	{"product material", asString("product material", "material")},
	{"product mid code", asString("product mid code", "mid_code")},
	{"product origin country", asString("product origin country", "origin_country")},
	{"product weight", asNumber("product weight", "weight")},
	{"product width", asNumber("product width", "width")},
	{"product metadata", asJSON("product metadata", "metadata")},
	{"shipping profile id", asString("shipping profile id", "shipping_profile_id")},
	{"product is giftcard", asBoolean("product is giftcard", "is_giftcard")},
	{"product created at", ignored()},
	{"product deleted at", ignored()},
	{"product updated at", ignored()},
}

var productWildcardColumns = []column{
	{"product category", asCounterValue(regexp.MustCompile(`product category \d`), "id", "categories")},
	{"product image", asCounterValue(regexp.MustCompile(`product image \d`), "url", "images")},
	{"product tag", asCounterValue(regexp.MustCompile(`product tag \d`), "id", "tags")},
	{"product sales channel", asCounterValue(regexp.MustCompile(`product sales channel \d`), "id", "sales_channels")},
}

var variantStaticColumns = []column{
	{"variant id", asString("variant id", "id")},
	{"variant title", asString("variant title", "title")},
	{"variant sku", asString("variant sku", "sku")},
	{"variant upc", asString("variant upc", "upc")},
	{"variant ean", asString("variant ean", "ean")},
	{"variant hs code", asString("variant hs code", "hs_code")},
	{"variant mid code", asString("variant mid code", "mid_code")},
	{"variant manage inventory", asBoolean("variant manage inventory", "manage_inventory")},
	{"variant allow backorder", asBoolean("variant allow backorder", "allow_backorder")},
	{"variant barcode", asString("variant barcode", "barcode")},
	{"variant height", asNumber("variant height", "height")},
	{"variant length", asNumber("variant length", "length")},
	{"variant material", asString("variant material", "material")},
	{"variant metadata", asJSON("variant metadata", "metadata")},
	{"variant origin country", asString("variant origin country", "origin_country")},
	{"variant variant rank", asNumber("variant variant rank", "variant_rank")},
	{"variant width", asNumber("variant width", "width")},
	{"variant weight", asNumber("variant weight", "weight")},
	{"variant created at", ignored()},
	{"variant deleted at", ignored()},
	{"variant updated at", ignored()},
	{"variant product id", ignored()},
}

var (
	exportPriceRe = regexp.MustCompile(`^(.+) \[([^\]]+)\]$`)
	bracketRe     = regexp.MustCompile(`\[(.*)\]`)
	optionNameRe  = regexp.MustCompile(`variant option \d+ name`)
)

// parseVariantPriceColumn mirrors parseVariantPriceColumn.
func parseVariantPriceColumn(name string, rowNumber int) (iso string, region *string, err error) {
	normalized := strings.TrimPrefix(name, "variant price ")
	if m := exportPriceRe.FindStringSubmatch(normalized); m != nil {
		r := m[1]
		return strings.ToLower(m[2]), &r, nil
	}
	if m := bracketRe.FindStringSubmatch(normalized); m != nil {
		r := m[1]
		region = &r
	}
	parts := strings.Split(normalized, " ")
	iso = parts[len(parts)-1]
	if iso == "" {
		return "", nil, createError(rowNumber, fmt.Sprintf(`Invalid price format used by "%s". Expect column name to contain the ISO code as the last segment. For example: "Variant Price [Europe] EUR" or "Variant Price EUR"`, name))
	}
	return strings.ToLower(iso), region, nil
}

func variantPrices(row Row, columns []string, rowNumber int, out *dal.Record, ctx *Context) error {
	var priceColumns []string
	for _, c := range columns {
		if strings.HasPrefix(c, "variant price ") && isPresent(get(row, c)) {
			priceColumns = append(priceColumns, c)
		}
	}
	prices, _ := get(out, "prices").([]any)
	if prices == nil {
		prices = []any{}
	}
	for _, c := range priceColumns {
		iso, regionName, err := parseVariantPriceColumn(c, rowNumber)
		if err != nil {
			return err
		}
		v := get(row, c)
		n, ok := JSNumber(v)
		if !ok {
			return createError(rowNumber, fmt.Sprintf(`Invalid value provided for "%s". Expected value to be a number, received "%v"`, c, v))
		}
		p := dal.NewRecord()
		if regionName == nil {
			p.Set("currency_code", iso)
			p.Set("amount", n)
			prices = append(prices, p)
			continue
		}
		var region *Region
		if ctx != nil {
			for i := range ctx.Regions {
				if strings.ToLower(ctx.Regions[i].Name) == *regionName {
					region = &ctx.Regions[i]
					break
				}
			}
		}
		if region == nil {
			return createError(rowNumber, fmt.Sprintf(`Region with name "%s" not found, used by "%s"`, *regionName, c))
		}
		rules := dal.NewRecord()
		rules.Set("region_id", region.ID)
		p.Set("currency_code", region.CurrencyCode)
		p.Set("amount", n)
		p.Set("rules", rules)
		prices = append(prices, p)
	}
	out.Set("prices", prices)
	return nil
}

var knownStaticColumns, knownWildcardColumns = func() (map[string]bool, []string) {
	static := map[string]bool{}
	for _, c := range append(append([]column{}, productStaticColumns...), variantStaticColumns...) {
		static[c.name] = true
	}
	var wildcard []string
	for _, c := range productWildcardColumns {
		wildcard = append(wildcard, c.name)
	}
	return static, append(wildcard, "variant price", "variant option")
}()

// Region is the region data the normalizer resolves region prices with.
type Region struct {
	ID           string
	Name         string
	CurrencyCode string
}

// Context mirrors CSVNormalizerContext.
type Context struct {
	Regions []Region
}

// PreProcess mirrors CSVNormalizer.preProcess.
func PreProcess(row Row, rowNumber int) (Row, error) {
	var unknown []string
	normalized := dal.NewRecord()
	for _, k := range row.Keys() {
		lower := strings.ToLower(NormalizeCSVValue(k).(string))
		if !knownStaticColumns[lower] {
			known := false
			for _, w := range knownWildcardColumns {
				if strings.HasPrefix(lower, w) {
					known = true
					break
				}
			}
			if !known {
				unknown = append(unknown, k)
			}
		}
		v, _ := row.Get(k)
		normalized.Set(lower, NormalizeCSVValue(v))
	}
	if len(unknown) > 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Invalid column name(s) "%s"`, strings.Join(unknown, `","`)))
	}
	if !isPresent(get(normalized, "product id")) && !isPresent(get(normalized, "product handle")) {
		return nil, createError(rowNumber, "Missing product id and handle. One of these columns are required to process the row")
	}
	return normalized, nil
}

// Products is the normalizer output: products to create by handle and to
// update by id, in first-seen order.
type Products struct {
	ToCreate *dal.Record
	ToUpdate *dal.Record
}

// CSVNormalizer mirrors CSVNormalizer.
type CSVNormalizer struct {
	rows     []Row
	products Products
	ctx      *Context
}

// NewCSVNormalizer mirrors new CSVNormalizer(rows, context).
func NewCSVNormalizer(rows []Row, ctx *Context) *CSVNormalizer {
	return &CSVNormalizer{rows: rows, ctx: ctx, products: Products{ToCreate: dal.NewRecord(), ToUpdate: dal.NewRecord()}}
}

func getOrInit(r *dal.Record, key string) *dal.Record {
	if v, ok := r.Get(key); ok {
		return v.(*dal.Record)
	}
	p := dal.NewRecord()
	r.Set(key, p)
	return p
}

func jsString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

func (n *CSVNormalizer) processRow(row Row, rowNumber int) error {
	columns := row.Keys()
	var product *dal.Record
	if id := get(row, "product id"); id != nil && id != "" && id != false {
		product = getOrInit(n.products.ToUpdate, jsString(id))
	} else {
		product = getOrInit(n.products.ToCreate, jsString(get(row, "product handle")))
	}
	for _, c := range productStaticColumns {
		if err := c.fn(row, columns, rowNumber, product, n.ctx); err != nil {
			return err
		}
	}
	for _, c := range productWildcardColumns {
		if err := c.fn(row, columns, rowNumber, product, n.ctx); err != nil {
			return err
		}
	}
	variant := dal.NewRecord()
	for _, c := range variantStaticColumns {
		if err := c.fn(row, columns, rowNumber, variant, n.ctx); err != nil {
			return err
		}
	}
	if err := variantPrices(row, columns, rowNumber, variant, n.ctx); err != nil {
		return err
	}

	type option struct {
		key, value  any
		id          *string
		isExclusive any
	}
	var options []option
	for _, c := range columns {
		if !optionNameRe.MatchString(c) || !isPresent(get(row, c)) {
			continue
		}
		counter := strings.Split(c, " ")[2]
		value := get(row, "variant option "+counter+" value")
		if !isPresent(value) {
			return createError(rowNumber, fmt.Sprintf(`Missing option value for "%s"`, c))
		}
		o := option{key: get(row, c), value: value}
		if id := get(row, "variant option "+counter+" id"); isPresent(id) {
			s := jsString(id)
			o.id = &s
		}
		if raw := get(row, "variant option "+counter+" is exclusive"); isPresent(raw) {
			if b := tryConvertToBoolean(raw, nil); b != nil {
				o.isExclusive = b
			}
		}
		options = append(options, o)
	}
	for _, o := range options {
		vo, _ := get(variant, "options").(*dal.Record)
		if vo == nil {
			vo = dal.NewRecord()
			variant.Set("options", vo)
		}
		vo.Set(jsString(o.key), o.value)
		popts, _ := get(product, "options").([]any)
		if popts == nil {
			popts = []any{}
		}
		var match *dal.Record
		for _, po := range popts {
			if r := po.(*dal.Record); get(r, "title") == o.key {
				match = r
				break
			}
		}
		if match == nil {
			no := dal.NewRecord()
			no.Set("title", o.key)
			no.Set("values", []any{o.value})
			if o.id != nil {
				no.Set("id", *o.id)
			}
			if o.isExclusive != nil {
				no.Set("is_exclusive", o.isExclusive)
			}
			popts = append(popts, no)
		} else {
			values, _ := get(match, "values").([]any)
			found := false
			for _, v := range values {
				if v == o.value {
					found = true
					break
				}
			}
			if !found {
				match.Set("values", append(values, o.value))
			}
			if !match.Has("id") && o.id != nil {
				match.Set("id", *o.id)
			}
			if !match.Has("is_exclusive") && o.isExclusive != nil {
				match.Set("is_exclusive", o.isExclusive)
			}
		}
		product.Set("options", popts)
	}
	variants, _ := get(product, "variants").([]any)
	if variants == nil {
		variants = []any{}
	}
	product.Set("variants", append(variants, variant))
	return nil
}

// Process mirrors proccess(resumingFromIndex).
func (n *CSVNormalizer) Process(resumingFromIndex int) (Products, error) {
	for i, row := range n.rows {
		if err := n.processRow(row, resumingFromIndex+i+1); err != nil {
			return Products{}, err
		}
	}
	n.rows = nil
	return n.products, nil
}
