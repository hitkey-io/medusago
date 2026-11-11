// Package services mirrors @medusajs/search/src/services.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"

	coremodulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	modulessdktypes "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/search/src/models"
	"github.com/hitkey-io/medusago/packages/modules/search/src/utils"
)

// SearchModuleService mirrors SearchModuleService: no generated CRUD (the
// models are the module's own bookkeeping), the definitions resolved once
// at construction.
type SearchModuleService struct {
	*modulessdk.MedusaService
	logger       *slog.Logger
	providers    *SearchProviderService
	indexes      *utils.Indexes
	eventRoutes  map[string][]string
	ctx          *utils.Context
	isWorkerMode bool
}

// keyedLocks mirrors the in-process Locking Module: execute(key, job)
// serializes the jobs of one key.
type keyedLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

var processLocks = &keyedLocks{locks: map[string]*sync.Mutex{}}

func (l *keyedLocks) Execute(ctx context.Context, key string, job func(ctx context.Context) error) error {
	l.mu.Lock()
	m, ok := l.locks[key]
	if !ok {
		m = &sync.Mutex{}
		l.locks[key] = m
	}
	l.mu.Unlock()
	m.Lock()
	defer m.Unlock()
	return job(ctx)
}

func optionString(options map[string]any, key string) string {
	s, _ := options[key].(string)
	return s
}

// NewSearchModuleService builds the service. The index definitions are the
// registered ones (defineSearchIndex), like the `indexes` option the
// application passes upstream.
func NewSearchModuleService(opts modulessdktypes.ServiceOptions) (*SearchModuleService, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	var providers []searchtypes.Provider
	for _, p := range opts.Providers {
		if provider, ok := p.Instance.(searchtypes.Provider); ok {
			providers = append(providers, provider)
		}
	}
	providerService, err := NewSearchProviderService(providers)
	if err != nil {
		return nil, err
	}
	s := &SearchModuleService{
		MedusaService: modulessdk.NewMedusaService(opts.DB, models.All, nil),
		logger:        logger,
		providers:     providerService,
		// worker_mode mirrors projectConfig.workerMode (MEDUSA_WORKER_MODE).
		isWorkerMode: os.Getenv("MEDUSA_WORKER_MODE") != "server",
	}
	definitions := coremodulessdk.GetSearchIndexes()
	s.indexes = utils.NewIndexes()
	if len(definitions) > 0 {
		defaultProvider, err := providerService.GetDefaultIdentifier(optionString(opts.Options, "default_provider"))
		if err != nil {
			return nil, err
		}
		s.indexes, err = utils.ResolveIndexDefinitions(definitions, defaultProvider, optionString(opts.Options, "index_prefix"))
		if err != nil {
			return nil, err
		}
	}
	s.eventRoutes = utils.BuildEventRoutes(s.indexes)
	batchSize := 0
	if reindex, ok := opts.Options["reindex"].(map[string]any); ok {
		if n, ok := reindex["batch_size"].(float64); ok {
			batchSize = int(n)
		}
	}
	container := opts.Container
	s.ctx = &utils.Context{
		Container: func() searchtypes.Container {
			var q any
			if container != nil {
				q = container.Resolve(common.ContainerRegistrationKeys.QUERY)
			}
			return searchtypes.Container{Query: q, Logger: logger}
		},
		Logger:         logger,
		BatchSize:      batchSize,
		Indexes:        s.indexes,
		Providers:      providerService,
		IndexService:   s.Internal("SearchIndex"),
		VersionService: s.Internal("SearchIndexVersion"),
		SyncService:    s.Internal("SearchIndexSync"),
		Locking:        processLocks,
	}
	s.ctx.ActiveVersionCache = utils.NewActiveIndexVersionCache(s.fetchActiveVersions)
	return s, nil
}

// SetWorkerMode mirrors the worker_mode of the module declaration: a
// "server" process does not seed at start.
func (s *SearchModuleService) SetWorkerMode(workerMode string) {
	s.isWorkerMode = workerMode != "server"
}

// fetchActiveVersions mirrors fetchActiveVersions_.
func (s *SearchModuleService) fetchActiveVersions(ctx context.Context) (map[string]utils.ActiveIndexVersion, error) {
	records, err := utils.ListIndexRecords(ctx, s.ctx, dal.NewRecord(), modulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(records))
	for i, r := range records {
		ids[i] = r.ID
	}
	versionsByIndexID, err := utils.ListVersionsByIndexID(ctx, s.ctx, ids)
	if err != nil {
		return nil, err
	}
	values := map[string]utils.ActiveIndexVersion{}
	for _, record := range records {
		if record.ActiveVersion == nil {
			continue
		}
		for _, v := range versionsByIndexID[record.ID] {
			if v.Version == *record.ActiveVersion {
				values[record.Name] = utils.ActiveIndexVersion{PhysicalName: v.PhysicalName, Provider: v.Provider, Version: v.Version}
				break
			}
		}
	}
	return values, nil
}

// Events are the events the registered definitions consume (the
// search-ingestion subscriber's config).
func (s *SearchModuleService) Events() []string {
	seen := map[string]bool{}
	var out []string
	for _, d := range s.indexes.Values() {
		for _, e := range d.Events {
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// OnApplicationStart mirrors __hooks.onApplicationStart: seeding runs in
// the background and its failures are logged.
func (s *SearchModuleService) OnApplicationStart(ctx context.Context) error {
	go func() {
		if err := s.onApplicationStart(context.Background()); err != nil {
			s.logger.Error(fmt.Sprintf("[Search] Failed to seed search indexes: %s", err.Error()))
		}
	}()
	return nil
}

func (s *SearchModuleService) onApplicationStart(ctx context.Context) error {
	if s.indexes.Len() == 0 || !s.isWorkerMode {
		return nil
	}
	seeds, err := utils.CreateSeedPlan(ctx, s.ctx)
	if err != nil {
		return err
	}
	if len(seeds) == 0 {
		return nil
	}
	parts := make([]string, len(seeds))
	for i, seed := range seeds {
		parts[i] = fmt.Sprintf("%s (%s)", seed.Index, seed.Reason)
	}
	s.logger.Info(fmt.Sprintf("[Search] Seeding %d index(es): %s", len(seeds), strings.Join(parts, ", ")))
	return utils.ExecuteSeedPlan(ctx, s.ctx, seeds)
}

// Search mirrors search(query).
func (s *SearchModuleService) Search(ctx context.Context, query *dal.Record) (*dal.Record, error) {
	results, err := s.SearchMany(ctx, []*dal.Record{query})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

// SearchMany mirrors searchMany(queries).
func (s *SearchModuleService) SearchMany(ctx context.Context, queries []*dal.Record) ([]*dal.Record, error) {
	if len(queries) == 0 {
		return []*dal.Record{}, nil
	}
	return utils.WithActiveIndexRetry(ctx, s.ctx, func() ([]*dal.Record, error) {
		return s.searchMany(ctx, queries)
	})
}

type flatQuery struct {
	provider      searchtypes.Provider
	query         *searchtypes.ProviderQuery
	originalIndex int
	facetField    string
}

func (s *SearchModuleService) searchMany(ctx context.Context, queries []*dal.Record) ([]*dal.Record, error) {
	type prepared struct {
		normalized *searchtypes.ProviderQuery
		provider   searchtypes.Provider
	}
	preps := make([]prepared, len(queries))
	for i, query := range queries {
		entity, _ := query.Get("entity")
		name, _ := entity.(string)
		index, err := utils.ResolveActiveDefinition(ctx, s.ctx, name, false)
		if err != nil {
			return nil, err
		}
		normalized := utils.NormalizeSearchQuery(query, index)
		provider, err := s.providers.Retrieve(index.Provider)
		if err != nil {
			return nil, err
		}
		if err := utils.ValidateFieldUsage(index, normalized); err != nil {
			return nil, err
		}
		preps[i] = prepared{normalized: normalized, provider: provider}
	}

	var flattened []flatQuery
	for i, p := range preps {
		disjunctive := false
		if v, ok := p.normalized.SearchOptions.Get("disjunctive_facets"); ok {
			disjunctive, _ = v.(bool)
		}
		if !disjunctive {
			flattened = append(flattened, flatQuery{provider: p.provider, query: p.normalized, originalIndex: i})
			continue
		}
		base, order, perFacet := utils.BuildDisjunctiveFacetQueries(p.normalized)
		flattened = append(flattened, flatQuery{provider: p.provider, query: base, originalIndex: i})
		for _, field := range order {
			flattened = append(flattened, flatQuery{provider: p.provider, query: perFacet[field], originalIndex: i, facetField: field})
		}
	}

	// Grouped by provider so each engine gets one searchMany.
	var groupOrder []searchtypes.Provider
	groups := map[searchtypes.Provider][]int{}
	for i, item := range flattened {
		if _, ok := groups[item.provider]; !ok {
			groupOrder = append(groupOrder, item.provider)
		}
		groups[item.provider] = append(groups[item.provider], i)
	}
	flatResults := make([]*dal.Record, len(flattened))
	for _, provider := range groupOrder {
		group := groups[provider]
		inputs := make([]*searchtypes.ProviderQuery, len(group))
		for i, flatIndex := range group {
			inputs[i] = flattened[flatIndex].query
		}
		results, err := provider.SearchMany(ctx, inputs)
		if err != nil {
			return nil, err
		}
		for i, flatIndex := range group {
			flatResults[flatIndex] = results[i]
		}
	}

	results := make([]*dal.Record, len(queries))
	type pending struct {
		order []string
		byKey map[string]*dal.Record
	}
	var pendingOrder []int
	pendingFacets := map[int]*pending{}
	for flatIndex, item := range flattened {
		if item.facetField == "" {
			results[item.originalIndex] = flatResults[flatIndex]
			continue
		}
		p, ok := pendingFacets[item.originalIndex]
		if !ok {
			p = &pending{byKey: map[string]*dal.Record{}}
			pendingFacets[item.originalIndex] = p
			pendingOrder = append(pendingOrder, item.originalIndex)
		}
		if _, ok := p.byKey[item.facetField]; !ok {
			p.order = append(p.order, item.facetField)
		}
		p.byKey[item.facetField] = flatResults[flatIndex]
	}
	for _, index := range pendingOrder {
		p := pendingFacets[index]
		results[index] = utils.MergeDisjunctiveFacetResults(results[index], p.order, p.byKey)
	}
	return results, nil
}

// UpsertDocuments mirrors upsertDocuments({ index, documents }).
func (s *SearchModuleService) UpsertDocuments(ctx context.Context, index string, documents []*dal.Record) (searchtypes.Task, error) {
	if len(documents) == 0 {
		return searchtypes.Task{Index: index, Status: "succeeded"}, nil
	}
	return utils.WithActiveIndexRetry(ctx, s.ctx, func() (searchtypes.Task, error) {
		definition, err := utils.ResolveActiveDefinition(ctx, s.ctx, index, true)
		if err != nil {
			return searchtypes.Task{}, err
		}
		provider, err := s.providers.Retrieve(definition.Provider)
		if err != nil {
			return searchtypes.Task{}, err
		}
		task, err := provider.UpsertDocuments(ctx, definition.PhysicalName, definition, documents)
		if err != nil {
			return searchtypes.Task{}, err
		}
		return utils.AssertTaskAccepted(task, index)
	})
}

// DeleteDocuments mirrors deleteDocuments({ index, filters }).
func (s *SearchModuleService) DeleteDocuments(ctx context.Context, index string, filters *dal.Record) (searchtypes.Task, error) {
	if filters == nil || filters.Len() == 0 {
		return searchtypes.Task{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Deleting from search index "%s" requires filters; pass { id: [...] } to delete by id, or clear the index through a reindex`, index))
	}
	return utils.WithActiveIndexRetry(ctx, s.ctx, func() (searchtypes.Task, error) {
		definition, err := utils.ResolveActiveDefinition(ctx, s.ctx, index, true)
		if err != nil {
			return searchtypes.Task{}, err
		}
		provider, err := s.providers.Retrieve(definition.Provider)
		if err != nil {
			return searchtypes.Task{}, err
		}
		task, err := provider.DeleteDocuments(ctx, definition.PhysicalName, filters)
		if err != nil {
			return searchtypes.Task{}, err
		}
		return utils.AssertTaskAccepted(task, index)
	})
}

// Ingest mirrors ingest(event).
func (s *SearchModuleService) Ingest(ctx context.Context, event searchtypes.Event) ([]searchtypes.Task, error) {
	return utils.IngestEvent(ctx, s.ctx, event, s.eventRoutes)
}

// ListIndexes mirrors listIndexes: the registered indexes by name, with
// the provider and status of their active version.
func (s *SearchModuleService) ListIndexes(ctx context.Context) ([]*dal.Record, error) {
	definitions := s.indexes.Values()
	sort.SliceStable(definitions, func(i, j int) bool { return localeCompare(definitions[i].Name, definitions[j].Name) < 0 })
	out := []*dal.Record{}
	if len(definitions) == 0 {
		return out, nil
	}
	names := make([]any, len(definitions))
	for i, d := range definitions {
		names[i] = d.Name
	}
	where := dal.NewRecord()
	where.Set("name", names)
	records, err := utils.ListIndexRecords(ctx, s.ctx, where, modulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	byName := map[string]utils.IndexRecord{}
	ids := make([]string, len(records))
	for i, r := range records {
		byName[r.Name] = r
		ids[i] = r.ID
	}
	versionsByIndexID, err := utils.ListVersionsByIndexID(ctx, s.ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, definition := range definitions {
		record, hasRecord := byName[definition.Name]
		var active *utils.VersionRecord
		if hasRecord && record.ActiveVersion != nil {
			versions := versionsByIndexID[record.ID]
			for i := range versions {
				if versions[i].Version == *record.ActiveVersion {
					active = &versions[i]
					break
				}
			}
		}
		r := dal.NewRecord()
		r.Set("name", definition.Name)
		r.Set("entity", definition.Entity)
		provider, status := definition.Provider, utils.StatePending
		if active != nil {
			provider, status = active.Provider, active.Status
		}
		r.Set("provider", provider)
		r.Set("status", status)
		r.Set("fields", utils.ListIndexedFields(definition.Fields))
		out = append(out, r)
	}
	return out, nil
}

// localeCompare approximates String.prototype.localeCompare for index
// names (case-insensitive first, then case).
func localeCompare(a, b string) int {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		return strings.Compare(la, lb)
	}
	return -strings.Compare(a, b)
}

// ListRetrievableFields mirrors listRetrievableFields.
func (s *SearchModuleService) ListRetrievableFields(index string) ([]string, error) {
	definition, err := utils.RetrieveIndexDefinition(s.indexes, index)
	if err != nil {
		return nil, err
	}
	return utils.ListRetrievablePaths(definition.Fields), nil
}

// GetIndex mirrors getIndex.
func (s *SearchModuleService) GetIndex(index string) (*searchtypes.ResolvedDefinition, error) {
	return utils.RetrieveIndexDefinition(s.indexes, index)
}

// ResolveIndex is GetIndex for callers that only need the check.
func (s *SearchModuleService) ResolveIndex(index string) error {
	_, err := s.GetIndex(index)
	return err
}

// IndexPrimaryKey returns getIndex(index).primary_key.
func (s *SearchModuleService) IndexPrimaryKey(index string) (string, error) {
	definition, err := s.GetIndex(index)
	if err != nil {
		return "", err
	}
	return definition.PrimaryKey, nil
}

// Reindex mirrors reindex(input).
func (s *SearchModuleService) Reindex(ctx context.Context, input searchtypes.ReindexInput) (searchtypes.ReindexResult, error) {
	return utils.ReindexIndexes(ctx, s.ctx, input)
}

// DeleteIndex mirrors deleteIndex(index): { index, deleted_versions }.
func (s *SearchModuleService) DeleteIndex(ctx context.Context, index string) (*dal.Record, error) {
	result, err := utils.DeleteIndexEntirely(ctx, s.ctx, index)
	if err != nil {
		return nil, err
	}
	out := dal.NewRecord()
	out.Set("index", result.Index)
	out.Set("deleted_versions", float64(result.DeletedVersions))
	return out, nil
}

// CreateIndexMigrationPlan mirrors createIndexMigrationPlan.
func (s *SearchModuleService) CreateIndexMigrationPlan(ctx context.Context) ([]searchtypes.MigrationAction, error) {
	return utils.CreateIndexMigrationPlan(ctx, s.ctx)
}

// ExecuteIndexMigrationPlan mirrors executeIndexMigrationPlan.
func (s *SearchModuleService) ExecuteIndexMigrationPlan(ctx context.Context, actions []searchtypes.MigrationAction) error {
	return utils.ExecuteIndexMigrationPlan(ctx, s.ctx, actions)
}
