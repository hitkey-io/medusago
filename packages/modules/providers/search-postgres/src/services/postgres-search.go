// Package services mirrors @medusajs/search-postgres/src/services.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	searchutils "github.com/hitkey-io/medusago/packages/core/utils/src/search"
	"github.com/hitkey-io/medusago/packages/modules/providers/search-postgres/src/utils"
)

// Identifier mirrors PostgresSearchService.identifier.
const Identifier = "search-postgres"

// upsertChunkSize mirrors UPSERT_CHUNK_SIZE.
const upsertChunkSize = 200

// Options mirrors PostgresSearchProviderOptions (the embedder and the
// vector distance only apply to the lakebase engine).
type Options struct {
	Language string
	Engine   string
}

// PostgresSearchService mirrors PostgresSearchService for the native engine
// (GIN + ts_rank + pg_trgm). The lakebase engine (BM25 / ANN vectors) only
// exists on Medusa Cloud and is not ported.
type PostgresSearchService struct {
	db       dal.Querier
	logger   *slog.Logger
	language string
	engine   string
}

var languageRe = regexp.MustCompile(`(?i)^[a-z0-9_]+$`)

// NewPostgresSearchService mirrors the constructor.
func NewPostgresSearchService(db dal.Querier, logger *slog.Logger, options Options) (*PostgresSearchService, error) {
	if db == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, "The postgres search provider requires a database manager. Ensure the Search Module is configured with a Postgres connection.")
	}
	language := options.Language
	if language == "" {
		language = "english"
	}
	if !languageRe.MatchString(language) {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Invalid postgres search language "%s". Use a built-in text search configuration name such as "english" or "simple".`, language))
	}
	engine := options.Engine
	if engine == "" {
		engine = utils.EngineNative
	}
	if engine != utils.EngineNative && engine != utils.EngineLakebase {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Invalid postgres search engine "%s". Use "native" or "lakebase".`, options.Engine))
	}
	if engine == utils.EngineLakebase {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, `The "lakebase" postgres search engine is only available on Medusa Cloud. Use "native".`)
	}
	return &PostgresSearchService{db: db, logger: logger, language: language, engine: engine}, nil
}

// Identifier returns the provider identifier.
func (s *PostgresSearchService) Identifier() string { return Identifier }

func (s *PostgresSearchService) tsConfig() string { return utils.TextSearchConfigName(s.language) }

// positional turns the `?` placeholders into $n.
func positional(sql string) string {
	var b strings.Builder
	n := 0
	for _, r := range sql {
		if r == '?' {
			n++
			b.WriteString("$" + strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// execute mirrors manager.execute(sql, params): the parameters are inlined
// into the statement (simple protocol), like MikroORM formats them.
func execute(ctx context.Context, db dal.Querier, sql string, params ...any) ([]dal.Row, error) {
	return dal.QueryRows(ctx, db, positional(sql), params...)
}

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func (s *PostgresSearchService) withTransaction(ctx context.Context, run func(db dal.Querier) error) error {
	b, ok := s.db.(txBeginner)
	if !ok {
		return run(s.db)
	}
	return pgx.BeginFunc(ctx, b, func(tx pgx.Tx) error { return run(tx) })
}

type storedIndex struct {
	name          string
	tableName     string
	schemaHash    string
	plan          *utils.IndexPlan
	documentCount int
}

func (s *PostgresSearchService) getCatalog(ctx context.Context, db dal.Querier, name string) (*storedIndex, error) {
	rows, err := execute(ctx, db, fmt.Sprintf(`SELECT * FROM "%s" WHERE "name" = ?`, utils.CatalogTable), name)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	row := rows[0]
	out := &storedIndex{plan: utils.DeserializePlan(row["plan"])}
	out.name, _ = row["name"].(string)
	out.tableName, _ = row["table_name"].(string)
	out.schemaHash, _ = row["schema_hash"].(string)
	out.documentCount = int(searchutils.JSNumber(row["document_count"]))
	return out, nil
}

func (s *PostgresSearchService) createDocumentTable(ctx context.Context, db dal.Querier, table string, plan *utils.IndexPlan) error {
	var vectorColumns []string
	for _, path := range plan.Vectors {
		planned := plan.Field(path)
		vectorColumns = append(vectorColumns, fmt.Sprintf(`"%s" vector(%d)`, utils.VectorColumnName(path), *planned.Dimensions))
	}
	vectorSQL := ""
	if len(vectorColumns) > 0 {
		vectorSQL = strings.Join(vectorColumns, ",\n        ") + ","
	}
	stmts := []string{fmt.Sprintf(`
      CREATE TABLE IF NOT EXISTS "%[1]s" (
        "id" text NOT NULL,
        "document" jsonb NOT NULL,
        "indexed" jsonb NOT NULL DEFAULT '{}'::jsonb,
        "search_vector" tsvector NOT NULL DEFAULT ''::tsvector,
        "search_text" text NOT NULL DEFAULT '',
        %[2]s
        "created_at" timestamptz NOT NULL DEFAULT now(),
        "updated_at" timestamptz NOT NULL DEFAULT now(),
        CONSTRAINT "%[1]s_pkey" PRIMARY KEY ("id")
      )
    `, table, vectorSQL)}
	for _, path := range plan.Vectors {
		planned := plan.Field(path)
		stmts = append(stmts, fmt.Sprintf(`
        ALTER TABLE "%s"
        ADD COLUMN IF NOT EXISTS "%s" vector(%d)
      `, table, utils.VectorColumnName(path), *planned.Dimensions))
	}
	stmts = append(stmts,
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "%[1]s_fts_idx" ON "%[1]s" USING GIN ("search_vector")`, table),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "%[1]s_trgm_idx" ON "%[1]s" USING GIN ("search_text" gin_trgm_ops)`, table),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "%[1]s_indexed_idx" ON "%[1]s" USING GIN ("indexed" jsonb_path_ops)`, table),
	)
	for _, stmt := range stmts {
		if _, err := execute(ctx, db, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresSearchService) dropDocumentTable(ctx context.Context, db dal.Querier, table string) error {
	_, err := execute(ctx, db, fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE`, table))
	return err
}

func (s *PostgresSearchService) adjustDocumentCount(ctx context.Context, db dal.Querier, name string, delta int) error {
	if delta == 0 {
		return nil
	}
	_, err := execute(ctx, db, fmt.Sprintf(`UPDATE "%s"
       SET "document_count" = GREATEST("document_count" + ?, 0), "updated_at" = now()
       WHERE "name" = ?`, utils.CatalogTable), delta, name)
	return err
}

func (s *PostgresSearchService) setDocumentCount(ctx context.Context, db dal.Querier, name string, count int) error {
	_, err := execute(ctx, db, fmt.Sprintf(`UPDATE "%s"
       SET "document_count" = ?, "updated_at" = now()
       WHERE "name" = ?`, utils.CatalogTable), count, name)
	return err
}

func (s *PostgresSearchService) task(index string) searchtypes.Task {
	return searchtypes.Task{Index: index, Status: "succeeded"}
}

// UpsertIndex mirrors upsertIndex.
func (s *PostgresSearchService) UpsertIndex(ctx context.Context, index *searchtypes.ResolvedDefinition) (searchtypes.Task, error) {
	if err := utils.AssertIndexSupported(index, s.engine, s.logger); err != nil {
		return searchtypes.Task{}, err
	}
	plan, err := utils.BuildIndexPlan(index, s.engine)
	if err != nil {
		return searchtypes.Task{}, err
	}
	table, err := utils.TableNameForIndex(index.PhysicalName)
	if err != nil {
		return searchtypes.Task{}, err
	}
	existing, err := s.getCatalog(ctx, s.db, index.PhysicalName)
	if err != nil {
		return searchtypes.Task{}, err
	}
	rebuilt := existing != nil && !utils.SameSchema(existing.plan, plan) && existing.tableName != ""
	planJSON, err := utils.SerializePlan(plan)
	if err != nil {
		return searchtypes.Task{}, err
	}
	err = s.withTransaction(ctx, func(db dal.Querier) error {
		if rebuilt {
			if err := s.dropDocumentTable(ctx, db, existing.tableName); err != nil {
				return err
			}
		}
		if err := s.createDocumentTable(ctx, db, table, plan); err != nil {
			return err
		}
		if existing != nil {
			reset := ""
			if rebuilt {
				reset = `"document_count" = 0,`
			}
			_, err := execute(ctx, db, fmt.Sprintf(`UPDATE "%s"
           SET "table_name" = ?, "schema_hash" = ?, "plan" = ?::jsonb,
               %s
               "updated_at" = now()
           WHERE "name" = ?`, utils.CatalogTable, reset), table, plan.SchemaHash, planJSON, index.PhysicalName)
			return err
		}
		_, err := execute(ctx, db, fmt.Sprintf(`INSERT INTO "%s"
            ("name", "table_name", "schema_hash", "plan", "document_count")
           VALUES (?, ?, ?, ?::jsonb, 0)`, utils.CatalogTable), index.PhysicalName, table, plan.SchemaHash, planJSON)
		return err
	})
	if err != nil {
		return searchtypes.Task{}, err
	}
	return s.task(index.PhysicalName), nil
}

// DeleteIndex mirrors deleteIndex.
func (s *PostgresSearchService) DeleteIndex(ctx context.Context, index string) (searchtypes.Task, error) {
	existing, err := s.getCatalog(ctx, s.db, index)
	if err != nil {
		return searchtypes.Task{}, err
	}
	if existing != nil {
		if err := s.dropDocumentTable(ctx, s.db, existing.tableName); err != nil {
			return searchtypes.Task{}, err
		}
		if _, err := execute(ctx, s.db, fmt.Sprintf(`DELETE FROM "%s" WHERE "name" = ?`, utils.CatalogTable), index); err != nil {
			return searchtypes.Task{}, err
		}
	}
	return s.task(index), nil
}

// ListIndexes mirrors listIndexes.
func (s *PostgresSearchService) ListIndexes(ctx context.Context) ([]searchtypes.IndexInfo, error) {
	rows, err := execute(ctx, s.db, fmt.Sprintf(`SELECT "name", "document_count", "created_at", "updated_at"
       FROM "%s"
       ORDER BY "name"`, utils.CatalogTable))
	if err != nil {
		return nil, err
	}
	out := make([]searchtypes.IndexInfo, len(rows))
	for i, row := range rows {
		name, _ := row["name"].(string)
		created, _ := row["created_at"].(time.Time)
		updated, _ := row["updated_at"].(time.Time)
		out[i] = searchtypes.IndexInfo{Name: name, Provider: Identifier, DocumentCount: int(searchutils.JSNumber(row["document_count"])), CreatedAt: created, UpdatedAt: updated}
	}
	return out, nil
}

func (s *PostgresSearchService) buildSearchVectorSQL(parts []utils.WeightedPart, params *[]any) string {
	if len(parts) == 0 {
		return `''::tsvector`
	}
	out := make([]string, len(parts))
	for i, part := range parts {
		*params = append(*params, part.Text)
		out[i] = fmt.Sprintf(`setweight(to_tsvector('%s', coalesce(?, '')), '%s')`, s.tsConfig(), part.Weight)
	}
	return strings.Join(out, " || ")
}

// UpsertDocuments mirrors upsertDocuments.
func (s *PostgresSearchService) UpsertDocuments(ctx context.Context, index string, _ *searchtypes.ResolvedDefinition, documents []*dal.Record) (searchtypes.Task, error) {
	catalog, err := s.retrieve(ctx, index)
	if err != nil {
		return searchtypes.Task{}, err
	}
	if len(documents) == 0 {
		return s.task(index), nil
	}
	type entry struct {
		document   *dal.Record
		projection *utils.Projection
	}
	// Last write wins on duplicate ids (Map insertion order kept).
	var order []string
	byID := map[string]entry{}
	for _, document := range documents {
		projection, err := utils.ProjectIndexedDocument(document, catalog.plan)
		if err != nil {
			return searchtypes.Task{}, err
		}
		if _, ok := byID[projection.ID]; !ok {
			order = append(order, projection.ID)
		}
		byID[projection.ID] = entry{document: document, projection: projection}
	}
	if engineEmbedded(catalog.plan) {
		return searchtypes.Task{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, fmt.Sprintf(`Vector field "%s" declares "embed", which requires an "embedder" function on the postgres search provider options`, firstEmbedded(catalog.plan)))
	}
	vectors := catalog.plan.Vectors
	err = s.withTransaction(ctx, func(db dal.Querier) error {
		inserted := 0
		for offset := 0; offset < len(order); offset += upsertChunkSize {
			end := offset + upsertChunkSize
			if end > len(order) {
				end = len(order)
			}
			var rowsSQL []string
			params := []any{}
			for _, id := range order[offset:end] {
				e := byID[id]
				docJSON, err := dal.MarshalValue(e.document)
				if err != nil {
					return err
				}
				indexedJSON, err := dal.MarshalValue(e.projection.Indexed)
				if err != nil {
					return err
				}
				params = append(params, e.projection.ID, string(docJSON), string(indexedJSON))
				vectorSQL := s.buildSearchVectorSQL(e.projection.WeightedParts, &params)
				params = append(params, e.projection.SearchText)
				var vectorPlaceholders []string
				for _, path := range vectors {
					if embedding, ok := e.projection.Vectors[path]; ok {
						parts := make([]string, len(embedding))
						for i, f := range embedding {
							parts[i] = searchutils.JSNumberString(f)
						}
						params = append(params, "["+strings.Join(parts, ",")+"]")
					} else {
						params = append(params, nil)
					}
					vectorPlaceholders = append(vectorPlaceholders, "?::vector")
				}
				vp := ""
				if len(vectorPlaceholders) > 0 {
					vp = ", " + strings.Join(vectorPlaceholders, ", ")
				}
				rowsSQL = append(rowsSQL, fmt.Sprintf("(?, ?::jsonb, ?::jsonb, %s, ?%s, now())", vectorSQL, vp))
			}
			var vectorCols, vectorUpdates []string
			for _, path := range vectors {
				col := `"` + utils.VectorColumnName(path) + `"`
				vectorCols = append(vectorCols, col)
				vectorUpdates = append(vectorUpdates, col+" = EXCLUDED."+col)
			}
			vc, vu := "", ""
			if len(vectorCols) > 0 {
				vc = ", " + strings.Join(vectorCols, ", ")
				vu = ", " + strings.Join(vectorUpdates, ", ")
			}
			rows, err := execute(ctx, db, fmt.Sprintf(`
          INSERT INTO "%s"
            ("id", "document", "indexed", "search_vector", "search_text"%s, "updated_at")
          VALUES %s
          ON CONFLICT ("id") DO UPDATE SET
            "document" = EXCLUDED."document",
            "indexed" = EXCLUDED."indexed",
            "search_vector" = EXCLUDED."search_vector",
            "search_text" = EXCLUDED."search_text"%s,
            "updated_at" = now()
          RETURNING (xmax = 0)::int AS inserted
          `, catalog.tableName, vc, strings.Join(rowsSQL, ",\n"), vu), params...)
			if err != nil {
				return err
			}
			for _, row := range rows {
				inserted += int(searchutils.JSNumber(row["inserted"]))
			}
		}
		return s.adjustDocumentCount(ctx, db, index, inserted)
	})
	if err != nil {
		return searchtypes.Task{}, err
	}
	return s.task(index), nil
}

func engineEmbedded(plan *utils.IndexPlan) bool { return firstEmbedded(plan) != "" }

func firstEmbedded(plan *utils.IndexPlan) string {
	for _, path := range plan.Vectors {
		if f := plan.Field(path); f != nil && f.Field.Embed != nil && *f.Field.Embed {
			return path
		}
	}
	return ""
}

// DeleteDocuments mirrors deleteDocuments.
func (s *PostgresSearchService) DeleteDocuments(ctx context.Context, index string, filters *dal.Record) (searchtypes.Task, error) {
	catalog, err := s.retrieve(ctx, index)
	if err != nil {
		return searchtypes.Task{}, err
	}
	err = s.withTransaction(ctx, func(db dal.Querier) error {
		var deleted []dal.Row
		if ids, ok := searchutils.ExtractPrimaryKeyFilter(filters, catalog.plan.PrimaryKey); ok {
			if len(ids) > 0 {
				placeholders := make([]string, len(ids))
				params := make([]any, len(ids))
				for i, id := range ids {
					placeholders[i] = "?"
					params[i] = id
				}
				rows, err := execute(ctx, db, fmt.Sprintf(`DELETE FROM "%s" WHERE "id" = ANY(ARRAY[%s]::text[]) RETURNING "id"`, catalog.tableName, strings.Join(placeholders, ", ")), params...)
				if err != nil {
					return err
				}
				deleted = rows
			}
		} else {
			where, err := utils.ToWhereClause(filters, catalog.plan)
			if err != nil {
				return err
			}
			if where != nil && where.SQL != "" {
				rows, err := execute(ctx, db, fmt.Sprintf(`DELETE FROM "%s" WHERE %s RETURNING "id"`, catalog.tableName, where.SQL), where.Params...)
				if err != nil {
					return err
				}
				deleted = rows
			}
		}
		return s.adjustDocumentCount(ctx, db, index, -len(deleted))
	})
	if err != nil {
		return searchtypes.Task{}, err
	}
	return s.task(index), nil
}

// ClearIndex mirrors clearIndex.
func (s *PostgresSearchService) ClearIndex(ctx context.Context, index string) (searchtypes.Task, error) {
	catalog, err := s.retrieve(ctx, index)
	if err != nil {
		return searchtypes.Task{}, err
	}
	if _, err := execute(ctx, s.db, fmt.Sprintf(`TRUNCATE TABLE "%s"`, catalog.tableName)); err != nil {
		return searchtypes.Task{}, err
	}
	if err := s.setDocumentCount(ctx, s.db, index, 0); err != nil {
		return searchtypes.Task{}, err
	}
	return s.task(index), nil
}

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

func intOr(v any, fallback int) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return fallback
}

func notSearchable(path, index string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Field "%s" is not searchable on search index "%s"`, path, index))
}

func stringList(v any) ([]string, bool) {
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, len(arr))
	for i, e := range arr {
		out[i], _ = e.(string)
	}
	return out, true
}

func contains(list []string, v string) bool {
	for _, e := range list {
		if e == v {
			return true
		}
	}
	return false
}

// Search mirrors search (keyword and filter-only queries; vector and
// hybrid queries need the lakebase engine and are rejected by
// assertQuerySupported).
func (s *PostgresSearchService) Search(ctx context.Context, input *searchtypes.ProviderQuery) (*dal.Record, error) {
	if err := utils.AssertQuerySupported(input, s.engine); err != nil {
		return nil, err
	}
	catalog, err := s.retrieve(ctx, input.Index.PhysicalName)
	if err != nil {
		return nil, err
	}
	plan := catalog.plan
	options := input.SearchOptions
	skip := intOr(get(input.Pagination, "skip"), 0)
	take := intOr(get(input.Pagination, "take"), 20)
	started := time.Now()

	if attrs, ok := stringList(get(options, "attributes_to_search_on")); ok {
		for _, path := range attrs {
			if !contains(plan.Searchable, path) {
				return nil, notSearchable(path, input.Index.Name)
			}
		}
	}

	distinct, _ := get(options, "distinct").(string)
	if get(options, "distinct") == nil {
		if input.Index.Settings != nil {
			distinct = input.Index.Settings.DistinctAttribute
		}
	}
	if _, err := s.resolveDistinct(distinct, plan, input.Index.Name); err != nil {
		return nil, err
	}

	q := ""
	if input.Q != nil {
		q = strings.TrimSpace(*input.Q)
	}
	if q != "" && len(plan.Searchable) == 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Search index "%s" has no searchable fields, so it cannot be queried by text`, input.Index.Name))
	}
	useKeyword := q != ""

	hitRows, count, err := s.searchKeyword(ctx, input, catalog, distinct, skip, take)
	if err != nil {
		return nil, err
	}

	scope, err := s.buildFacetScope(input, catalog, useKeyword)
	if err != nil {
		return nil, err
	}
	var facetRequests []any
	if arr, ok := get(options, "facets").([]any); ok {
		facetRequests = arr
	}
	facets, err := s.resolveFacets(ctx, catalog.tableName, plan, scope, facetRequests)
	if err != nil {
		return nil, err
	}

	includeScore := searchutils.Truthy(get(options, "include_score"))
	hits := make([]any, len(hitRows))
	for i, row := range hitRows {
		document, _ := row["document"].(*dal.Record)
		hit := dal.NewRecord()
		hit.Set("id", row["id"])
		if includeScore {
			hit.Set("score", searchutils.JSNumber(row["score"]))
		}
		hit.Set("document", searchutils.ProjectDocument(document, input.AttributesToRetrieve))
		hits[i] = hit
	}
	out := dal.NewRecord()
	out.Set("hits", hits)
	if facets != nil {
		out.Set("facets", facets)
	}
	metadata := dal.NewRecord()
	metadata.Set("skip", float64(skip))
	metadata.Set("take", float64(take))
	if get(options, "count") == "none" || count == nil {
		metadata.Set("count", nil)
	} else {
		metadata.Set("count", float64(*count))
	}
	if input.Q != nil {
		metadata.Set("query", *input.Q)
	}
	metadata.Set("processing_time_ms", float64(time.Since(started).Milliseconds()))
	out.Set("metadata", metadata)
	return out, nil
}

type keywordFragments struct {
	q        string
	rankSQL  func(params *[]any) string
	matchSQL func(params *[]any) string
}

// keywordFragments mirrors keywordFragments for the native engine.
func (s *PostgresSearchService) keywordFragments(query *searchtypes.ProviderQuery, catalog *storedIndex) keywordFragments {
	plan := catalog.plan
	options := query.SearchOptions
	q := ""
	if query.Q != nil {
		q = strings.TrimSpace(*query.Q)
	}
	typoEnabled := true
	if st := query.Index.Settings; st != nil && st.TypoTolerance != nil && st.TypoTolerance.Enabled != nil {
		typoEnabled = *st.TypoTolerance.Enabled
	}
	typo := q != "" && searchutils.Truthy(get(options, "typo_tolerance")) && typoEnabled
	matchStrategy, _ := get(options, "match_strategy").(string)
	searchOn := plan.Searchable
	if attrs, ok := stringList(get(options, "attributes_to_search_on")); ok {
		searchOn = attrs
	}
	searchAllFields := len(searchOn) == len(plan.Searchable)
	for _, path := range searchOn {
		if !contains(plan.Searchable, path) {
			searchAllFields = false
		}
	}
	tsConfig := s.tsConfig()
	vectorExpr := `"search_vector"`
	textExpr := `"search_text"`
	if !searchAllFields {
		vectorExpr = s.onTheFlyVectorExpr(searchOn, plan)
		textExpr = onTheFlyTextExpr(searchOn)
	}
	tsquery := func(params *[]any) string {
		*params = append(*params, q)
		return utils.KeywordTsQuerySql(tsConfig, matchStrategy)
	}
	wordSim := func(params *[]any) string {
		*params = append(*params, q)
		return utils.WordSimilarityCall("?", textExpr)
	}
	rank := func(params *[]any) string {
		if q == "" {
			return `0::float4`
		}
		base := fmt.Sprintf("ts_rank_cd(%s, %s)", vectorExpr, tsquery(params))
		if typo {
			return fmt.Sprintf("GREATEST(%s, %s)", base, wordSim(params))
		}
		return base
	}
	f := keywordFragments{q: q, rankSQL: rank}
	if q != "" {
		f.matchSQL = func(params *[]any) string {
			match := fmt.Sprintf("%s @@ %s", vectorExpr, tsquery(params))
			if typo {
				return fmt.Sprintf("(%s OR %s > 0.3)", match, wordSim(params))
			}
			return match
		}
	}
	return f
}

func (s *PostgresSearchService) searchKeyword(ctx context.Context, query *searchtypes.ProviderQuery, catalog *storedIndex, distinct string, skip, take int) ([]dal.Row, *int, error) {
	plan := catalog.plan
	options := query.SearchOptions
	filterWhere, err := utils.ToWhereClause(query.Filters, plan)
	if err != nil {
		return nil, nil, err
	}
	fragments := s.keywordFragments(query, catalog)
	conditionsSQL := func(params *[]any) []string {
		var conditions []string
		if fragments.matchSQL != nil {
			conditions = append(conditions, fragments.matchSQL(params))
		}
		if filterWhere != nil {
			conditions = append(conditions, "("+filterWhere.SQL+")")
			*params = append(*params, filterWhere.Params...)
		}
		return conditions
	}
	orderSQL, err := s.resolveOrderBy(query, plan, fragments.q != "")
	if err != nil {
		return nil, nil, err
	}
	distinctExpr, err := s.resolveDistinct(distinct, plan, query.Index.Name)
	if err != nil {
		return nil, nil, err
	}
	var minScore *float64
	if v, ok := get(options, "min_score").(float64); ok {
		minScore = &v
	}
	return s.executeQuery(ctx, queryParts{
		table:         catalog.tableName,
		rankSQL:       fragments.rankSQL,
		conditionsSQL: conditionsSQL,
		orderSQL:      orderSQL,
		minScore:      minScore,
		distinctExpr:  distinctExpr,
		count:         get(options, "count") != "none",
		take:          take,
		skip:          skip,
	})
}

type queryParts struct {
	table         string
	rankSQL       func(params *[]any) string
	conditionsSQL func(params *[]any) []string
	orderSQL      string
	minScore      *float64
	distinctExpr  string
	count         bool
	take          int
	skip          int
}

// executeQuery mirrors executeQuery for a single-arm search.
func (s *PostgresSearchService) executeQuery(ctx context.Context, parts queryParts) ([]dal.Row, *int, error) {
	wrap := parts.minScore != nil || parts.distinctExpr != ""
	buildBase := func(params *[]any) string {
		rank := parts.rankSQL(params)
		conditions := parts.conditionsSQL(params)
		whereSQL := ""
		if len(conditions) > 0 {
			whereSQL = "WHERE " + strings.Join(conditions, " AND ")
		}
		return fmt.Sprintf(`SELECT "id", "document", "indexed", %s AS score FROM "%s" %s`, rank, parts.table, whereSQL)
	}
	buildWrappedFrom := func(params *[]any) string {
		from := fmt.Sprintf("(%s) ranked", buildBase(params))
		if parts.distinctExpr != "" {
			from = fmt.Sprintf(`(
          SELECT ranked.*, ROW_NUMBER() OVER (
            PARTITION BY %s ORDER BY %s
          ) AS __rn
          FROM %s
        ) deduped`, parts.distinctExpr, parts.orderSQL, from)
		}
		return from
	}
	wrappedConditions := func() []string {
		var outer []string
		if parts.distinctExpr != "" {
			outer = append(outer, "__rn = 1")
		}
		if parts.minScore != nil {
			outer = append(outer, "score >= ?")
		}
		return outer
	}
	whereOf := func(outer []string) string {
		if len(outer) == 0 {
			return ""
		}
		return "WHERE " + strings.Join(outer, " AND ")
	}

	hitsParams := []any{}
	var hitsSQL string
	if !wrap {
		base := buildBase(&hitsParams)
		hitsSQL = fmt.Sprintf("%s ORDER BY %s LIMIT ? OFFSET ?", base, parts.orderSQL)
		hitsParams = append(hitsParams, parts.take, parts.skip)
	} else {
		from := buildWrappedFrom(&hitsParams)
		hitsSQL = fmt.Sprintf(`
        SELECT * FROM %s
        %s
        ORDER BY %s
        LIMIT ? OFFSET ?
      `, from, whereOf(wrappedConditions()), parts.orderSQL)
		if parts.minScore != nil {
			hitsParams = append(hitsParams, *parts.minScore)
		}
		hitsParams = append(hitsParams, parts.take, parts.skip)
	}

	var countSQL string
	countParams := []any{}
	if parts.count {
		if !wrap {
			conditions := parts.conditionsSQL(&countParams)
			where := ""
			if len(conditions) > 0 {
				where = "WHERE " + strings.Join(conditions, " AND ")
			}
			countSQL = fmt.Sprintf(`SELECT COUNT(*)::int AS count FROM "%s" %s`, parts.table, where)
		} else {
			from := buildWrappedFrom(&countParams)
			countSQL = fmt.Sprintf(`
          SELECT COUNT(*)::int AS count FROM %s
          %s
        `, from, whereOf(wrappedConditions()))
			if parts.minScore != nil {
				countParams = append(countParams, *parts.minScore)
			}
		}
	}

	hitRows := []dal.Row{}
	if parts.take != 0 {
		rows, err := execute(ctx, s.db, hitsSQL, hitsParams...)
		if err != nil {
			return nil, nil, err
		}
		hitRows = rows
	}
	var count *int
	if countSQL != "" {
		rows, err := execute(ctx, s.db, countSQL, countParams...)
		if err != nil {
			return nil, nil, err
		}
		n := 0
		if len(rows) > 0 {
			n = int(searchutils.JSNumber(rows[0]["count"]))
		}
		count = &n
	}
	return hitRows, count, nil
}

// SearchMany mirrors searchMany: one search per query.
func (s *PostgresSearchService) SearchMany(ctx context.Context, inputs []*searchtypes.ProviderQuery) ([]*dal.Record, error) {
	out := make([]*dal.Record, len(inputs))
	for i, input := range inputs {
		r, err := s.Search(ctx, input)
		if err != nil {
			return nil, err
		}
		out[i] = r
	}
	return out, nil
}

// buildFacetScope mirrors buildFacetScope: the filters plus the match
// predicate of the hits.
func (s *PostgresSearchService) buildFacetScope(query *searchtypes.ProviderQuery, catalog *storedIndex, useKeyword bool) (*utils.SqlFragment, error) {
	params := []any{}
	var conditions []string
	var fragments *keywordFragments
	if useKeyword {
		f := s.keywordFragments(query, catalog)
		fragments = &f
	}
	if fragments != nil && fragments.matchSQL != nil {
		conditions = append(conditions, fragments.matchSQL(&params))
	}
	if v, ok := get(query.SearchOptions, "min_score").(float64); ok && fragments != nil {
		conditions = append(conditions, fragments.rankSQL(&params)+" >= ?")
		params = append(params, v)
	}
	filterWhere, err := utils.ToWhereClause(query.Filters, catalog.plan)
	if err != nil {
		return nil, err
	}
	if filterWhere != nil {
		conditions = append(conditions, "("+filterWhere.SQL+")")
		params = append(params, filterWhere.Params...)
	}
	if len(conditions) == 0 {
		return nil, nil
	}
	return &utils.SqlFragment{SQL: strings.Join(conditions, " AND "), Params: params}, nil
}

func (s *PostgresSearchService) resolveFacets(ctx context.Context, table string, plan *utils.IndexPlan, scope *utils.SqlFragment, requests []any) (*dal.Record, error) {
	if len(requests) == 0 {
		return nil, nil
	}
	normalized, err := utils.NormalizeFacetRequests(requests, plan)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	result := dal.NewRecord()
	for _, request := range normalized {
		query := utils.BuildFacetQuery(table, scope, request, plan)
		rows, err := execute(ctx, s.db, query.SQL, query.Params...)
		if err != nil {
			return nil, err
		}
		result.Set(request.Field, utils.MapFacetResult(request, rows))
	}
	return result, nil
}

func jsonPathLiteral(path string) string {
	return "'{" + strings.Join(strings.Split(path, "."), ",") + "}'"
}

func onTheFlyTextExpr(paths []string) string {
	if len(paths) == 0 {
		return `''`
	}
	parts := make([]string, len(paths))
	for i, path := range paths {
		parts[i] = fmt.Sprintf("coalesce(indexed#>>%s, '')", jsonPathLiteral(path))
	}
	return strings.Join(parts, ` || ' ' || `)
}

func (s *PostgresSearchService) onTheFlyVectorExpr(paths []string, plan *utils.IndexPlan) string {
	if len(paths) == 0 {
		return `''::tsvector`
	}
	parts := make([]string, len(paths))
	for i, path := range paths {
		weight := "D"
		if planned := plan.Field(path); planned != nil {
			weight = utils.WeightLabel(planned.Field.Searchable)
		}
		parts[i] = fmt.Sprintf("setweight(to_tsvector('%s', coalesce(indexed#>>%s, '')), '%s')", s.tsConfig(), jsonPathLiteral(path), weight)
	}
	return strings.Join(parts, " || ")
}

func (s *PostgresSearchService) resolveOrderBy(input *searchtypes.ProviderQuery, plan *utils.IndexPlan, hasScore bool) (string, error) {
	order, _ := get(input.Pagination, "order").(*dal.Record)
	if order == nil || order.Len() == 0 {
		if hasScore {
			return `score DESC NULLS LAST, "id" ASC`, nil
		}
		return `"id" ASC`, nil
	}
	var clauses []string
	for _, property := range order.Keys() {
		direction, _ := order.Get(property)
		dir := "DESC"
		if direction == "ASC" {
			dir = "ASC"
		}
		if property == "_score" {
			clauses = append(clauses, "score "+dir+" NULLS LAST")
			continue
		}
		planned := plan.Field(property)
		if planned == nil {
			return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Cannot sort search index "%s" by unknown field "%s"`, input.Index.Name, property))
		}
		if planned.IsArray || planned.Kind == "vector" {
			return "", common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf(`The postgres search provider cannot sort by "%s"`, property))
		}
		path := jsonPathLiteral(property)
		switch planned.Kind {
		case "number":
			clauses = append(clauses, fmt.Sprintf("(indexed#>>%s)::float8 %s NULLS LAST", path, dir))
		case "boolean":
			clauses = append(clauses, fmt.Sprintf("(indexed#>>%s)::boolean %s NULLS LAST", path, dir))
		default:
			clauses = append(clauses, fmt.Sprintf("indexed#>>%s %s NULLS LAST", path, dir))
		}
	}
	return strings.Join(clauses, ", "), nil
}

func (s *PostgresSearchService) resolveDistinct(distinct string, plan *utils.IndexPlan, indexName string) (string, error) {
	if distinct == "" {
		return "", nil
	}
	planned := plan.Field(distinct)
	if planned == nil {
		return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Cannot deduplicate search index "%s" by unknown field "%s"`, indexName, distinct))
	}
	if planned.IsArray || planned.Kind == "vector" {
		return "", common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf(`The postgres search provider cannot deduplicate by "%s"`, distinct))
	}
	return "indexed#>>" + jsonPathLiteral(distinct), nil
}

func (s *PostgresSearchService) retrieve(ctx context.Context, index string) (*storedIndex, error) {
	catalog, err := s.getCatalog(ctx, s.db, index)
	if err != nil {
		return nil, err
	}
	if catalog != nil {
		return catalog, nil
	}
	known, err := s.ListIndexes(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(known))
	for i, k := range known {
		names[i] = k.Name
	}
	list := strings.Join(names, ", ")
	if list == "" {
		list = "(none)"
	}
	return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`The postgres search provider has no index "%s". Known indexes: %s`, index, list))
}
