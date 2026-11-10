// Package services mirrors @medusajs/translation/src/services.
package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/models"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/utils"
)

// TranslationModuleService mirrors TranslationModuleService.
type TranslationModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewTranslationModuleService builds the service.
func NewTranslationModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*TranslationModuleService, error) {
	return &TranslationModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

func (s *TranslationModuleService) translationService() *utilsmodulessdk.InternalService {
	return s.Internal("Translation")
}

func (s *TranslationModuleService) localeService() *utilsmodulessdk.InternalService {
	return s.Internal("Locale")
}

func (s *TranslationModuleService) settingsService() *utilsmodulessdk.InternalService {
	return s.Internal("TranslationSettings")
}

func invalid(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

// upstreamTranslatableOrder is the order in which upstream registers the
// translatable entities: DmlEntity collects them while the module models are
// imported (default modules in configuration order, each module's models in
// import order).
var upstreamTranslatableOrder = []string{
	"ProductCategory", "ProductCollection", "ProductVariant", "ProductOptionValue", "ProductOption",
	"ProductTag", "ProductType", "Product", "CustomerGroup", "Region", "TaxRate", "RefundReason",
	"ReturnReason", "PropertyLabel", "ShippingOptionType", "ShippingOption",
}

// TranslatableEntities mirrors DmlEntity.getTranslatableEntities() in the
// upstream registration order (unknown entities last, in parse order).
func TranslatableEntities() []dml.TranslatableEntityEntry {
	entries := dml.GetTranslatableEntities()
	rank := func(name string) int {
		for i, n := range upstreamTranslatableOrder {
			if n == name {
				return i
			}
		}
		return len(upstreamTranslatableOrder)
	}
	sort.SliceStable(entries, func(i, j int) bool { return rank(entries[i].Entity) < rank(entries[j].Entity) })
	return entries
}

func str(r *dal.Record, key string) string {
	if r == nil {
		return ""
	}
	v, _ := r.Get(key)
	s, _ := v.(string)
	return s
}

func strList(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	return nil
}

func anyList(list []string) []any {
	out := make([]any, len(list))
	for i, s := range list {
		out[i] = s
	}
	return out
}

func copyRecord(r *dal.Record) *dal.Record {
	n := dal.NewRecord()
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		n.Set(k, v)
	}
	return n
}

// OnApplicationStart mirrors __hooks.onApplicationStart: a settings row per
// translatable entity is created, settings of entities no longer
// translatable are deactivated.
func (s *TranslationModuleService) OnApplicationStart(ctx context.Context) error {
	entities := TranslatableEntities()
	translatable := map[string]bool{}
	for _, e := range entities {
		translatable[utils.ToSnakeCase(e.Entity)] = true
	}
	current, err := s.settingsService().List(ctx, nil, utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for _, c := range current {
		existing[str(c, "entity_type")] = true
	}
	var upserts []*dal.Record
	for _, c := range current {
		active, _ := c.Get("is_active")
		if !translatable[str(c, "entity_type")] && active == true {
			r := dal.NewRecord()
			id, _ := c.Get("id")
			r.Set("id", id)
			r.Set("is_active", false)
			upserts = append(upserts, r)
		}
	}
	for _, e := range entities {
		entityType := utils.ToSnakeCase(e.Entity)
		if existing[entityType] {
			continue
		}
		r := dal.NewRecord()
		r.Set("entity_type", entityType)
		r.Set("fields", anyList(e.Fields))
		upserts = append(upserts, r)
	}
	_, err = s.settingsService().Upsert(ctx, upserts)
	return err
}

// TranslatableFields is the result of getTranslatableFields: entity type ->
// fields, in settings order.
type TranslatableFields struct {
	Keys   []string
	Fields map[string][]string
}

// GetTranslatableFields mirrors getTranslatableFields: the fields of the
// active settings ([] for inactive ones).
func (s *TranslationModuleService) GetTranslatableFields(ctx context.Context, entityType string) (TranslatableFields, error) {
	filters := dal.NewRecord()
	if entityType != "" {
		filters.Set("entity_type", entityType)
	}
	settings, err := s.settingsService().List(ctx, filters, utilsmodulessdk.FindConfig{})
	if err != nil {
		return TranslatableFields{}, err
	}
	out := TranslatableFields{Fields: map[string][]string{}}
	for _, setting := range settings {
		et := str(setting, "entity_type")
		if _, seen := out.Fields[et]; !seen {
			out.Keys = append(out.Keys, et)
		}
		active, _ := setting.Get("is_active")
		if active == true {
			f, _ := setting.Get("fields")
			out.Fields[et] = strList(f)
			if out.Fields[et] == nil {
				out.Fields[et] = []string{}
			}
		} else {
			out.Fields[et] = []string{}
		}
	}
	return out, nil
}

// GetInactiveTranslatableFields mirrors getInactiveTranslatableFields: the
// translatable fields of each entity missing from its active settings.
func (s *TranslationModuleService) GetInactiveTranslatableFields(ctx context.Context, entityType string) (TranslatableFields, error) {
	fields, err := s.GetTranslatableFields(ctx, entityType)
	if err != nil {
		return TranslatableFields{}, err
	}
	entities := TranslatableEntities()
	out := TranslatableFields{Fields: map[string][]string{}}
	for _, et := range fields.Keys {
		var entity *dml.TranslatableEntityEntry
		for i := range entities {
			if utils.ToSnakeCase(entities[i].Entity) == et {
				entity = &entities[i]
				break
			}
		}
		if entity == nil {
			continue
		}
		out.Keys = append(out.Keys, et)
		out.Fields[et] = common.ArrayDifference(entity.Fields, fields.Fields[et])
	}
	return out, nil
}

// prepareFilters mirrors prepareFilters: q searches the translations JSON.
func prepareFilters(filters *dal.Record) *dal.Record {
	if filters == nil {
		return nil
	}
	q, hasQ := filters.Get("q")
	if !hasQ {
		return filters
	}
	out := dal.NewRecord()
	for _, k := range filters.Keys() {
		if k == "q" {
			continue
		}
		v, _ := filters.Get(k)
		out.Set(k, v)
	}
	if qs, ok := q.(string); ok && qs != "" {
		out.Set("__translations_q", dal.RawWhere{SQL: "translations::text ILIKE ?", Args: []any{"%" + qs + "%"}})
	} else if q != nil && q != "" && q != false {
		out.Set("__translations_q", dal.RawWhere{SQL: "translations::text ILIKE ?", Args: []any{"%" + fmt.Sprint(q) + "%"}})
	}
	return out
}

// ensureReferenceFieldInConfig mirrors ensureReferenceFieldInConfig.
func ensureReferenceFieldInConfig(config utilsmodulessdk.FindConfig) utilsmodulessdk.FindConfig {
	if len(config.Select) == 0 {
		return config
	}
	for _, f := range config.Select {
		if f == "reference" {
			return config
		}
	}
	config.Select = append(append([]string{}, config.Select...), "reference")
	return config
}

func (s *TranslationModuleService) filterTranslations(ctx context.Context, rows []*dal.Record) ([]*dal.Record, error) {
	config, err := s.GetTranslatableFields(ctx, "")
	if err != nil {
		return nil, err
	}
	return utils.FilterTranslationFields(rows, config.Fields), nil
}

// ListTranslations mirrors listTranslations.
func (s *TranslationModuleService) ListTranslations(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	rows, err := s.translationService().List(ctx, prepareFilters(filters), ensureReferenceFieldInConfig(config))
	if err != nil {
		return nil, err
	}
	return s.filterTranslations(ctx, rows)
}

// ListAndCountTranslations mirrors listAndCountTranslations.
func (s *TranslationModuleService) ListAndCountTranslations(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	rows, n, err := s.translationService().ListAndCount(ctx, prepareFilters(filters), ensureReferenceFieldInConfig(config))
	if err != nil {
		return nil, 0, err
	}
	rows, err = s.filterTranslations(ctx, rows)
	return rows, n, err
}

// ListBySuffix routes listTranslations to its override.
func (s *TranslationModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	if suffix == "Translations" {
		return s.ListTranslations(ctx, filters, config)
	}
	return s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
}

// ListAndCountBySuffix routes listAndCountTranslations to its override.
func (s *TranslationModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	if suffix == "Translations" {
		return s.ListAndCountTranslations(ctx, filters, config)
	}
	return s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
}

// ListTranslationSettings mirrors listTranslationSettings.
func (s *TranslationModuleService) ListTranslationSettings(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	return s.settingsService().List(ctx, filters, config)
}

// CreateLocales mirrors createLocales (codes normalized).
func (s *TranslationModuleService) CreateLocales(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	in := make([]*dal.Record, len(data))
	for i, d := range data {
		n := copyRecord(d)
		n.Set("code", common.NormalizeLocale(str(d, "code")))
		in[i] = n
	}
	return s.localeService().Create(ctx, in)
}

// CreateTranslations mirrors createTranslations: locale codes normalized,
// translated_field_count computed from the translatable fields.
func (s *TranslationModuleService) CreateTranslations(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	config, err := s.GetTranslatableFields(ctx, "")
	if err != nil {
		return nil, err
	}
	in := make([]*dal.Record, len(data))
	for i, d := range data {
		n := copyRecord(d)
		n.Set("locale_code", common.NormalizeLocale(str(d, "locale_code")))
		t, _ := d.Get("translations")
		tr, _ := t.(*dal.Record)
		n.Set("translated_field_count", float64(utils.ComputeTranslatedFieldCount(tr, config.Fields[str(d, "reference")])))
		in[i] = n
	}
	return s.translationService().Create(ctx, in)
}

// UpdateTranslations mirrors updateTranslations for an array of updates:
// translated_field_count is recomputed for updates carrying translations.
func (s *TranslationModuleService) UpdateTranslations(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	in := make([]*dal.Record, len(data))
	var withTranslations []*dal.Record
	for i, d := range data {
		in[i] = copyRecord(d)
		if t, ok := d.Get("translations"); ok && t != nil {
			withTranslations = append(withTranslations, in[i])
		}
	}
	if len(withTranslations) > 0 {
		var ids []any
		for _, d := range withTranslations {
			if str(d, "reference") == "" {
				id, _ := d.Get("id")
				ids = append(ids, id)
			}
		}
		referenceMap := map[string]string{}
		if len(ids) > 0 {
			f := dal.NewRecord()
			f.Set("id", ids)
			existing, err := s.translationService().List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "reference"}})
			if err != nil {
				return nil, err
			}
			for _, e := range existing {
				referenceMap[str(e, "id")] = str(e, "reference")
			}
		}
		config, err := s.GetTranslatableFields(ctx, "")
		if err != nil {
			return nil, err
		}
		for _, d := range withTranslations {
			reference := str(d, "reference")
			if reference == "" {
				reference = referenceMap[str(d, "id")]
			}
			t, _ := d.Get("translations")
			tr, _ := t.(*dal.Record)
			d.Set("translated_field_count", float64(utils.ComputeTranslatedFieldCount(tr, config.Fields[reference])))
		}
	}
	items := make([]any, len(in))
	for i, r := range in {
		items[i] = r
	}
	res, err := s.translationService().Update(ctx, items)
	if err != nil {
		return nil, err
	}
	out, _ := res.([]*dal.Record)
	return out, nil
}

// SoftDeleteTranslations mirrors softDeleteTranslations.
func (s *TranslationModuleService) SoftDeleteTranslations(ctx context.Context, ids []any) error {
	_, err := s.translationService().SoftDelete(ctx, ids)
	return err
}

// RestoreTranslations mirrors restoreTranslations.
func (s *TranslationModuleService) RestoreTranslations(ctx context.Context, ids []any) error {
	_, err := s.translationService().Restore(ctx, ids)
	return err
}

// DeleteTranslations mirrors deleteTranslations.
func (s *TranslationModuleService) DeleteTranslations(ctx context.Context, ids []any) error {
	_, err := s.translationService().Delete(ctx, ids)
	return err
}

// ensureEntityType mirrors ensureEntityType_: settings without an
// entity_type take the one of the stored row.
func (s *TranslationModuleService) ensureEntityType(ctx context.Context, settings []*dal.Record) ([]*dal.Record, error) {
	var ids []any
	for _, st := range settings {
		if str(st, "entity_type") == "" && st.Has("id") {
			id, _ := st.Get("id")
			ids = append(ids, id)
		}
	}
	types := map[string]string{}
	if len(ids) > 0 {
		f := dal.NewRecord()
		f.Set("id", ids)
		existing, err := s.settingsService().List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "entity_type"}})
		if err != nil {
			return nil, err
		}
		for _, e := range existing {
			types[str(e, "id")] = str(e, "entity_type")
		}
	}
	out := make([]*dal.Record, len(settings))
	for i, st := range settings {
		n := copyRecord(st)
		et := str(st, "entity_type")
		if et == "" {
			et = types[str(st, "id")]
		}
		if et == "" {
			n.Set("entity_type", nil)
		} else {
			n.Set("entity_type", et)
		}
		out[i] = n
	}
	return out, nil
}

// validateSettings mirrors validateSettings.
func validateSettings(data []*dal.Record) error {
	entities := map[string]dml.TranslatableEntityEntry{}
	for _, e := range TranslatableEntities() {
		entities[utils.ToSnakeCase(e.Entity)] = e
	}
	var lines []string
	for _, item := range data {
		v, _ := item.Get("entity_type")
		entityType := "undefined"
		if s, ok := v.(string); ok {
			entityType = s
		}
		entity, ok := entities[entityType]
		if !ok || v == nil {
			lines = append(lines, fmt.Sprintf("- %s is not a translatable entity", entityType))
			continue
		}
		f, _ := item.Get("fields")
		if invalidFields := common.ArrayDifference(strList(f), entity.Fields); len(invalidFields) > 0 {
			lines = append(lines, fmt.Sprintf("- %s doesn't have the following fields set as translatable: %s", entityType, strings.Join(invalidFields, ", ")))
		}
	}
	if len(lines) > 0 {
		return invalid("Invalid translation settings:\n" + strings.Join(lines, "\n"))
	}
	return nil
}

// CreateTranslationSettings mirrors createTranslationSettings.
func (s *TranslationModuleService) CreateTranslationSettings(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	normalized, err := s.ensureEntityType(ctx, data)
	if err != nil {
		return nil, err
	}
	if err := validateSettings(normalized); err != nil {
		return nil, err
	}
	return s.settingsService().Create(ctx, data)
}

// UpdateTranslationSettings mirrors updateTranslationSettings: the
// translated_field_count of the translations of the updated entity types is
// recomputed.
func (s *TranslationModuleService) UpdateTranslationSettings(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	normalized, err := s.ensureEntityType(ctx, data)
	if err != nil {
		return nil, err
	}
	if err := validateSettings(normalized); err != nil {
		return nil, err
	}
	items := make([]any, len(data))
	for i, d := range data {
		items[i] = d
	}
	res, err := s.settingsService().Update(ctx, items)
	if err != nil {
		return nil, err
	}
	updated, _ := res.([]*dal.Record)

	var entityTypes []any
	seen := map[string]bool{}
	for _, n := range normalized {
		et := str(n, "entity_type")
		if !seen[et] {
			seen[et] = true
			entityTypes = append(entityTypes, et)
		}
	}
	config, err := s.GetTranslatableFields(ctx, "")
	if err != nil {
		return nil, err
	}
	f := dal.NewRecord()
	f.Set("reference", entityTypes)
	translations, err := s.translationService().List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(translations) > 0 {
		toUpdate := make([]any, len(translations))
		for i, t := range translations {
			r := dal.NewRecord()
			id, _ := t.Get("id")
			r.Set("id", id)
			tv, _ := t.Get("translations")
			tr, _ := tv.(*dal.Record)
			r.Set("translated_field_count", float64(utils.ComputeTranslatedFieldCount(tr, config.Fields[str(t, "reference")])))
			toUpdate[i] = r
		}
		if _, err := s.translationService().Update(ctx, toUpdate); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// DeleteTranslationSettings mirrors deleteTranslationSettings.
func (s *TranslationModuleService) DeleteTranslationSettings(ctx context.Context, ids []any) error {
	_, err := s.settingsService().Delete(ctx, ids)
	return err
}

// StatisticsCounts are the expected / translated / missing field counts.
type StatisticsCounts struct {
	Expected   int
	Translated int
	Missing    int
}

func (c StatisticsCounts) record() *dal.Record {
	r := dal.NewRecord()
	r.Set("expected", float64(c.Expected))
	r.Set("translated", float64(c.Translated))
	r.Set("missing", float64(c.Missing))
	return r
}

// GetStatistics mirrors getStatistics. entities maps entity type -> count,
// in entityOrder.
func (s *TranslationModuleService) GetStatistics(ctx context.Context, locales []string, entityOrder []string, counts map[string]int) (*dal.Record, error) {
	if len(locales) == 0 {
		return nil, invalid("At least one locale must be provided")
	}
	if len(entityOrder) == 0 {
		return nil, invalid("At least one entity type must be provided")
	}
	normalized := make([]string, len(locales))
	for i, l := range locales {
		normalized[i] = common.NormalizeLocale(l)
	}
	config, err := s.GetTranslatableFields(ctx, "")
	if err != nil {
		return nil, err
	}

	type entityStats struct {
		total    StatisticsCounts
		byLocale map[string]*StatisticsCounts
		order    []string
	}
	result := map[string]*entityStats{}
	newStats := func(perLocale int) *entityStats {
		st := &entityStats{byLocale: map[string]*StatisticsCounts{}}
		for _, l := range normalized {
			if _, ok := st.byLocale[l]; !ok {
				st.order = append(st.order, l)
			}
			st.byLocale[l] = &StatisticsCounts{Expected: perLocale, Missing: perLocale}
		}
		return st
	}
	// JS object key order: the entity types without translatable fields are
	// set first, the others when their counts are computed.
	var entityTypes, keyOrder []string
	for _, et := range entityOrder {
		if _, seen := result[et]; seen {
			continue
		}
		fields := config.Fields[et]
		if len(fields) == 0 {
			result[et] = newStats(0)
			keyOrder = append(keyOrder, et)
			continue
		}
		result[et] = nil
		entityTypes = append(entityTypes, et)
	}
	for _, et := range entityTypes {
		delete(result, et)
	}
	keyOrder = append(keyOrder, entityTypes...)
	render := func() *dal.Record {
		out := dal.NewRecord()
		for _, et := range keyOrder {
			st := result[et]
			r := st.total.record()
			byLocale := dal.NewRecord()
			for _, l := range st.order {
				byLocale.Set(l, st.byLocale[l].record())
			}
			r.Set("by_locale", byLocale)
			out.Set(et, r)
		}
		return out
	}
	if len(entityTypes) == 0 {
		return render(), nil
	}

	rows, err := s.DB().Query(ctx, `
      SELECT
        reference,
        locale_code,
        COALESCE(SUM(translated_field_count), 0)::int AS translated_field_count
      FROM translation
      WHERE reference = ANY($1)
        AND locale_code = ANY($2)
        AND deleted_at IS NULL
      GROUP BY reference, locale_code
      `, entityTypes, normalized)
	if err != nil {
		return nil, err
	}
	type row struct {
		reference, locale string
		count             int
	}
	var found []row
	for rows.Next() {
		var r row
		var n int32
		if err := rows.Scan(&r.reference, &r.locale, &n); err != nil {
			rows.Close()
			return nil, err
		}
		r.count = int(n)
		found = append(found, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, et := range entityTypes {
		perLocale := counts[et] * len(config.Fields[et])
		st := newStats(perLocale)
		st.total = StatisticsCounts{Expected: perLocale * len(normalized), Missing: perLocale * len(normalized)}
		result[et] = st
	}
	for _, r := range found {
		st := result[r.reference]
		c := st.byLocale[r.locale]
		c.Translated = r.count
		c.Missing = c.Expected - r.count
		st.total.Translated += r.count
	}
	for _, et := range entityTypes {
		result[et].total.Missing = result[et].total.Expected - result[et].total.Translated
	}
	return render(), nil
}
