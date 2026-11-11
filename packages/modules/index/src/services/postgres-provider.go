// Package services mirrors @medusajs/index/src/services.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbuslocal "github.com/hitkey-io/medusago/packages/modules/event-bus-local/src/services"
	"github.com/hitkey-io/medusago/packages/modules/index/src/utils"
)

// beginner is a database handle that opens transactions (the pool).
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PostgresProvider mirrors PostgresProvider: the index storage over the
// index_data / index_relation partitions.
type PostgresProvider struct {
	db     dal.Querier
	query  func() *query.Query
	schema *utils.SchemaObjectRepresentation

	// locking resolves the locking module (nil: none registered)
	locking func() lockingtypes.ILockingModule

	ready     chan struct{}
	readyErr  error
	readyOnce sync.Once
}

var eventActionToMethodMap = map[string]string{
	"created":  "onCreate",
	"updated":  "onUpdate",
	"deleted":  "onDelete",
	"restored": "onCreate",
	"attached": "onAttach",
	"detached": "onDetach",
}

// NewPostgresProvider builds the provider.
func NewPostgresProvider(db dal.Querier, q func() *query.Query, schema *utils.SchemaObjectRepresentation) *PostgresProvider {
	return &PostgresProvider{db: db, query: q, schema: schema, ready: make(chan struct{})}
}

// createPartitionsLockKey mirrors CREATE_PARTITIONS_LOCK_KEY.
const createPartitionsLockKey = "index-module:create-partitions"

// createPartitionsLockTimeout mirrors CREATE_PARTITIONS_LOCK_TIMEOUT
// (seconds).
const createPartitionsLockTimeout = 300

// OnApplicationStart mirrors onApplicationStart: creates the partitions
// (serialized across instances with the locking module); queries and event
// consumers wait for it.
func (p *PostgresProvider) OnApplicationStart(ctx context.Context) error {
	run := func(ctx context.Context) (any, error) {
		return nil, utils.CreatePartitions(ctx, p.db, p.schema, nil)
	}
	var err error
	var locking lockingtypes.ILockingModule
	if p.locking != nil {
		locking = p.locking()
	}
	if locking != nil {
		timeout := float64(createPartitionsLockTimeout)
		_, err = locking.Execute(ctx, []string{createPartitionsLockKey}, run, lockingtypes.ExecuteArgs{Timeout: &timeout})
	} else {
		_, err = run(ctx)
	}
	p.readyOnce.Do(func() {
		p.readyErr = err
		close(p.ready)
	})
	return err
}

func (p *PostgresProvider) isReady(ctx context.Context) error {
	select {
	case <-p.ready:
		return p.readyErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

type parsedData struct {
	data              []*dal.Record
	entityProperties  []string
	parentsProperties []parentProperties
}

type parentProperties struct {
	entity string
	fields []string
}

// parseData mirrors PostgresProvider.parseData.
func parseData(data []*dal.Record, rep *utils.SchemaObjectEntityRepresentation) (*parsedData, error) {
	out := &parsedData{data: data, entityProperties: []string{"id"}}
	index := map[string]int{}
	for _, field := range rep.Fields {
		if !strings.Contains(field, ".") {
			out.entityProperties = append(out.entityProperties, field)
			continue
		}
		parentAlias := strings.Split(field, ".")[0]
		var parent *utils.Parent
		for i := range rep.Parents {
			if rep.Parents[i].InverseSideProp == parentAlias {
				parent = &rep.Parents[i]
				break
			}
		}
		if parent == nil {
			return nil, fmt.Errorf("IndexModule error, unable to parse data for %s. The parent schema object representation could not be found for the alias %s for the entity %s.", rep.Entity, parentAlias, rep.Entity)
		}
		i, ok := index[parent.Ref.Entity]
		if !ok {
			i = len(out.parentsProperties)
			index[parent.Ref.Entity] = i
			out.parentsProperties = append(out.parentsProperties, parentProperties{entity: parent.Ref.Entity})
		}
		out.parentsProperties[i].fields = append(out.parentsProperties[i].fields, field)
	}
	return out, nil
}

// pick mirrors entityProperties.reduce((acc, p) => (acc[p] = data[p], acc)):
// undefined values are dropped by JSON.stringify.
func pick(entityData *dal.Record, properties []string) *dal.Record {
	out := dal.NewRecord()
	for _, prop := range properties {
		if v, ok := entityData.Get(prop); ok {
			out.Set(prop, v)
		}
	}
	return out
}

func recordsOf(v any) []*dal.Record {
	switch t := v.(type) {
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, e := range t {
			if r, ok := e.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	case []*dal.Record:
		return t
	case *dal.Record:
		return []*dal.Record{t}
	}
	return nil
}

func idsOf(data []*dal.Record) []string {
	var ids []string
	for _, d := range data {
		v, _ := d.Get("id")
		switch t := v.(type) {
		case []any:
			for _, e := range t {
				ids = append(ids, fmt.Sprint(e))
			}
		case string:
			ids = append(ids, t)
		case nil:
		default:
			ids = append(ids, fmt.Sprint(t))
		}
	}
	return ids
}

// ConsumeEvent mirrors consumeEvent(schemaEntityObjectRepresentation).
func (p *PostgresProvider) ConsumeEvent(rep *utils.SchemaObjectEntityRepresentation) eventbuslocal.Subscriber {
	return func(ctx context.Context, event eventbuslocal.Event) error {
		if err := p.isReady(ctx); err != nil {
			return err
		}
		data := recordsOf(event.Data)
		ids := idsOf(data)
		parts := strings.Split(event.Name, ".")
		action := parts[len(parts)-1]
		if event.Data != nil && event.Metadata != nil {
			if a, ok := event.Metadata.Get("action"); ok && a != nil {
				action = fmt.Sprint(a)
				ids = idsOf(data)
			}
		}
		method := eventActionToMethodMap[action]
		if method == "" {
			return nil
		}
		withDeleted := action == "deleted" || action == "detached"

		fields := []string{"id"}
		seen := map[string]bool{"id": true}
		for _, f := range rep.Fields {
			if !seen[f] {
				seen[f] = true
				fields = append(fields, f)
			}
		}
		const batchSize = 100
		for i := 0; i < len(ids); i += batchSize {
			end := i + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			batch := make([]any, 0, end-i)
			for _, id := range ids[i:end] {
				batch = append(batch, id)
			}
			filters := dal.NewRecord()
			filters.Set("id", batch)
			res, err := p.query().Graph(ctx, query.GraphInput{Entity: rep.Alias, Filters: filters, Fields: fields, WithDeleted: withDeleted}, joiner.Options{})
			if err != nil {
				return err
			}
			entityData := recordsOf(res.Data)
			var callErr error
			switch method {
			case "onCreate":
				callErr = p.OnCreate(ctx, rep.Entity, entityData, rep)
			case "onUpdate":
				callErr = p.OnUpdate(ctx, rep.Entity, entityData, rep)
			case "onDelete", "onDetach":
				callErr = p.OnDelete(ctx, rep.Entity, entityData, rep)
			case "onAttach":
				callErr = p.OnAttach(ctx, rep.Entity, entityData, rep)
			}
			if callErr != nil {
				return callErr
			}
		}
		return nil
	}
}

func marshal(r *dal.Record) (string, error) {
	b, err := r.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// inTransaction mirrors @InjectTransactionManager.
func (p *PostgresProvider) inTransaction(ctx context.Context, fn func(tx dal.Querier) error) error {
	b, ok := p.db.(beginner)
	if !ok {
		return fn(p.db)
	}
	tx, err := b.Begin(ctx)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

type indexDataRow struct {
	id, name, data string
}

type indexRelationRow struct {
	parentID, parentName, childID, childName, pivot string
}

func upsertIndexData(ctx context.Context, tx dal.Querier, rows []indexDataRow) error {
	if len(rows) == 0 {
		return nil
	}
	var values []string
	var args []any
	for _, r := range rows {
		n := len(args)
		values = append(values, fmt.Sprintf("($%d::jsonb, $%d, $%d, NULL)", n+1, n+2, n+3))
		args = append(args, r.data, r.id, r.name)
	}
	_, err := tx.Exec(ctx, `insert into "index_data" ("data", "id", "name", "staled_at") values `+strings.Join(values, ", ")+
		` on conflict ("id", "name") do update set "data" = excluded."data", "staled_at" = excluded."staled_at"`, args...)
	return err
}

func upsertIndexRelations(ctx context.Context, tx dal.Querier, rows []indexRelationRow) error {
	if len(rows) == 0 {
		return nil
	}
	var values []string
	var args []any
	for _, r := range rows {
		n := len(args)
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, NULL)", n+1, n+2, n+3, n+4, n+5))
		args = append(args, r.childID, r.childName, r.parentID, r.parentName, r.pivot)
	}
	_, err := tx.Exec(ctx, `insert into "index_relation" ("child_id", "child_name", "parent_id", "parent_name", "pivot", "staled_at") values `+strings.Join(values, ", ")+
		` on conflict ("pivot", "parent_id", "child_id", "parent_name", "child_name") do update set "staled_at" = excluded."staled_at"`, args...)
	return err
}

func idString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// OnCreate mirrors onCreate: upserts the index entries and their parent
// relations.
func (p *PostgresProvider) OnCreate(ctx context.Context, entity string, data []*dal.Record, rep *utils.SchemaObjectEntityRepresentation) error {
	parsed, err := parseData(data, rep)
	if err != nil {
		return err
	}
	var entities []indexDataRow
	seenEntities := map[string]bool{}
	var relations []indexRelationRow
	seenRelations := map[string]bool{}
	for index, entityData := range parsed.data {
		cleaned := pick(entityData, parsed.entityProperties)
		js, err := marshal(cleaned)
		if err != nil {
			return err
		}
		id := idString(get(cleaned, "id"))
		key := id + "\x00" + js
		if !seenEntities[key] {
			seenEntities[key] = true
			entities = append(entities, indexDataRow{id: id, name: entity, data: js})
		}
		for _, parent := range parsed.parentsProperties {
			parentAlias := strings.Split(parent.fields[0], ".")[0]
			parentData, ok := parsed.data[index].Get(parentAlias)
			if !ok || parentData == nil {
				continue
			}
			for _, pd := range recordsOf(parentData) {
				rel := indexRelationRow{parentID: idString(get(pd, "id")), parentName: parent.entity, childID: id, childName: entity, pivot: parent.entity + "-" + entity}
				k := rel.parentID + "\x00" + rel.parentName + "\x00" + rel.childID + "\x00" + rel.pivot
				if !seenRelations[k] {
					seenRelations[k] = true
					relations = append(relations, rel)
				}
			}
		}
	}
	return p.inTransaction(ctx, func(tx dal.Querier) error {
		if err := upsertIndexData(ctx, tx, entities); err != nil {
			return err
		}
		return upsertIndexRelations(ctx, tx, relations)
	})
}

// OnUpdate mirrors onUpdate.
func (p *PostgresProvider) OnUpdate(ctx context.Context, entity string, data []*dal.Record, rep *utils.SchemaObjectEntityRepresentation) error {
	parsed, err := parseData(data, rep)
	if err != nil {
		return err
	}
	var rows []indexDataRow
	for _, entityData := range parsed.data {
		cleaned := pick(entityData, parsed.entityProperties)
		js, err := marshal(cleaned)
		if err != nil {
			return err
		}
		rows = append(rows, indexDataRow{id: idString(get(entityData, "id")), name: entity, data: js})
	}
	return p.inTransaction(ctx, func(tx dal.Querier) error {
		return upsertIndexData(ctx, tx, rows)
	})
}

// OnDelete mirrors onDelete (and onDetach): removes the entries and every
// relation they take part in.
func (p *PostgresProvider) OnDelete(ctx context.Context, entity string, data []*dal.Record, rep *utils.SchemaObjectEntityRepresentation) error {
	parsed, err := parseData(data, rep)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(parsed.data))
	for _, d := range parsed.data {
		ids = append(ids, idString(get(d, "id")))
	}
	return p.inTransaction(ctx, func(tx dal.Querier) error {
		if _, err := tx.Exec(ctx, `delete from "index_data" where "id" = any($1) and "name" = $2`, ids, entity); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `delete from "index_relation" where ("parent_id" = any($1) and "parent_name" = $2) or ("child_id" = any($1) and "child_name" = $2)`, ids, entity)
		return err
	})
}

// OnAttach mirrors onAttach: a link entry and its relations to both linked
// entities.
func (p *PostgresProvider) OnAttach(ctx context.Context, entity string, data []*dal.Record, rep *utils.SchemaObjectEntityRepresentation) error {
	parsed, err := parseData(data, rep)
	if err != nil {
		return err
	}
	rels := rep.ModuleConfig.Relationships
	if len(rels) < 2 {
		return fmt.Errorf("IndexModule error, unable to handle attach event for %s.", entity)
	}
	parentPropertyID := rels[0].ForeignKey
	parentEntityName := p.schema.LinkableKeys[rels[0].ServiceName][parentPropertyID]
	if parentEntityName == "" {
		return fmt.Errorf("IndexModule error, unable to handle attach event for %s. The parent entity name could not be found using the linkable keys from the module %s.", entity, rels[0].ServiceName)
	}
	childPropertyID := rels[1].ForeignKey
	childEntityName := p.schema.LinkableKeys[rels[1].ServiceName][childPropertyID]
	if childEntityName == "" {
		return fmt.Errorf("IndexModule error, unable to handle attach event for %s. The child entity name could not be found using the linkable keys from the module %s.", entity, rels[1].ServiceName)
	}
	var entities []indexDataRow
	var relations []indexRelationRow
	for _, entityData := range parsed.data {
		cleaned := pick(entityData, parsed.entityProperties)
		js, err := marshal(cleaned)
		if err != nil {
			return err
		}
		id := idString(get(cleaned, "id"))
		relations = append(relations,
			indexRelationRow{parentID: idString(get(cleaned, parentPropertyID)), parentName: parentEntityName, childID: id, childName: entity, pivot: parentEntityName + "-" + entity},
			indexRelationRow{parentID: id, parentName: entity, childID: idString(get(cleaned, childPropertyID)), childName: childEntityName, pivot: entity + "-" + childEntityName},
		)
		entities = append(entities, indexDataRow{id: id, name: entity, data: js})
	}
	return p.inTransaction(ctx, func(tx dal.Querier) error {
		if err := upsertIndexData(ctx, tx, entities); err != nil {
			return err
		}
		return upsertIndexRelations(ctx, tx, relations)
	})
}

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

// QueryResult mirrors IndexTypes.QueryResultSet.
type QueryResult struct {
	Data     []any
	Metadata *dal.Record
}

// Query mirrors query(config): the index query builder SQL, run with the
// count estimate when skip is given.
func (p *PostgresProvider) Query(ctx context.Context, config utils.IndexQueryConfig) (*QueryResult, error) {
	if err := p.isReady(ctx); err != nil {
		return nil, err
	}
	prepared := utils.PrepareIndexQuery(p.schema, config)
	sql, sqlCount, err := prepared.Builder.BuildQuery(prepared.HasPagination, prepared.HasCount)
	if err != nil {
		return nil, err
	}
	rows, err := p.db.Query(ctx, sql, pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		return nil, err
	}
	var columns []string
	for _, fd := range rows.FieldDescriptions() {
		columns = append(columns, fd.Name)
	}
	var resultSet [][]any
	for rows.Next() {
		raw := rows.RawValues()
		row := make([]any, len(raw))
		for i, v := range raw {
			if v == nil {
				continue
			}
			oid := rows.FieldDescriptions()[i].DataTypeOID
			if oid == 3802 || oid == 114 { // jsonb, json
				decoded, err := dal.DecodeOrderedJSON(v)
				if err != nil {
					rows.Close()
					return nil, err
				}
				row[i] = decoded
				continue
			}
			row[i] = string(v)
		}
		resultSet = append(resultSet, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var estimate any
	if prepared.HasCount {
		r, err := p.db.Query(ctx, sqlCount, pgx.QueryExecModeSimpleProtocol)
		if err != nil {
			return nil, err
		}
		estimate = float64(0)
		if r.Next() {
			raw := r.RawValues()
			if len(raw) > 0 && raw[0] != nil {
				var f float64
				if err := json.Unmarshal(raw[0], &f); err == nil {
					estimate = f
				}
			}
		}
		r.Close()
		if err := r.Err(); err != nil {
			return nil, err
		}
	}

	out := &QueryResult{Data: prepared.Builder.BuildObjectFromResultset(columns, resultSet)}
	if prepared.HasPagination {
		md := dal.NewRecord()
		if prepared.HasCount {
			md.Set("estimate_count", estimate)
		}
		md.Set("skip", prepared.Skip)
		md.Set("take", prepared.Take)
		out.Metadata = md
	}
	return out, nil
}
