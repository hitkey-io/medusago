package modulessdk

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// FindConfig mirrors FindConfig.
type FindConfig struct {
	Select      []string
	Relations   []string
	Take        *int
	Skip        *int
	Order       []dal.OrderBy
	WithDeleted bool
}

// InternalService mirrors MedusaInternalService for one model.
type InternalService struct {
	Repository *dal.Repository
}

func (s *InternalService) model() *dml.Entity { return s.Repository.Entity }

// applyDefaultOrdering mirrors applyDefaultOrdering: order by the primary
// keys when no order is given.
func (s *InternalService) applyDefaultOrdering(config *FindConfig) {
	if len(config.Order) > 0 {
		return
	}
	for _, pk := range s.model().Meta.PrimaryKey {
		config.Order = append(config.Order, dal.OrderBy{Field: pk, Direction: "ASC"})
	}
}

func buildQuery(filters *dal.Record, config FindConfig) dal.FindOptions {
	return dal.FindOptions{
		Where:       filters,
		Select:      config.Select,
		Relations:   config.Relations,
		Order:       config.Order,
		Take:        config.Take,
		Skip:        config.Skip,
		WithDeleted: config.WithDeleted,
	}
}

// List mirrors list.
func (s *InternalService) List(ctx context.Context, filters *dal.Record, config FindConfig) ([]*dal.Record, error) {
	s.applyDefaultOrdering(&config)
	return s.Repository.Find(ctx, buildQuery(filters, config))
}

// ListAndCount mirrors listAndCount.
func (s *InternalService) ListAndCount(ctx context.Context, filters *dal.Record, config FindConfig) ([]*dal.Record, int, error) {
	s.applyDefaultOrdering(&config)
	return s.Repository.FindAndCount(ctx, buildQuery(filters, config))
}

func lowerCaseFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// Retrieve mirrors retrieve for single primary key models.
func (s *InternalService) Retrieve(ctx context.Context, id any, config FindConfig) (*dal.Record, error) {
	e := s.model()
	pks := e.Meta.PrimaryKey
	idStr, isString := id.(string)
	if id == nil || (isString && len(pks) > 1) || (!isString && len(pks) == 1) || (isString && idStr == "" && len(pks) == 1) {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND,
			fmt.Sprintf("%s - %s must be defined", lowerCaseFirst(e.Meta.ClassName), strings.Join(pks, ", ")))
	}

	where := dal.NewRecord()
	where.Set(pks[0], id)
	records, err := s.Repository.Find(ctx, buildQuery(where, config))
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND,
			fmt.Sprintf("%s with %s: %v was not found", e.Meta.ClassName, strings.Join(pks, ", "), id))
	}
	return records[0], nil
}

// txBeginner is implemented by *pgxpool.Pool and *pgx.Conn.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// inTransaction runs fn with a repository bound to a transaction, like
// @InjectTransactionManager.
func (s *InternalService) inTransaction(ctx context.Context, fn func(repo *dal.Repository) error) error {
	db, ok := s.Repository.DB.(txBeginner)
	if !ok {
		return fn(s.Repository)
	}
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		repo := *s.Repository
		repo.DB = tx
		return fn(&repo)
	})
}

// Create mirrors create. single reports whether the input was one object.
func (s *InternalService) Create(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	var out []*dal.Record
	err := s.inTransaction(ctx, func(repo *dal.Repository) error {
		var err error
		out, err = repo.Create(ctx, data)
		return err
	})
	return out, err
}

// mergeMetadata mirrors mergeMetadata: "" removes a key, other values
// replace it.
func mergeMetadata(current any, update *dal.Record) *dal.Record {
	merged := dal.NewRecord()
	if cur, ok := current.(*dal.Record); ok {
		for _, k := range cur.Keys() {
			v, _ := cur.Get(k)
			merged.Set(k, v)
		}
	}
	for _, k := range update.Keys() {
		v, _ := update.Get(k)
		if v == "" {
			merged.Delete(k)
			continue
		}
		merged.Set(k, v)
	}
	return merged
}

// Update mirrors update for the three input shapes: one object identified
// by its primary key, an array of them, or { selector, data }.
func (s *InternalService) Update(ctx context.Context, input any) (any, error) {
	e := s.model()
	pks := e.Meta.PrimaryKey

	var items []any
	shouldReturnArray := false
	switch v := input.(type) {
	case []any:
		items, shouldReturnArray = v, true
	case *dal.Record:
		items = []any{v}
		if v.Has("selector") {
			shouldReturnArray = true
		}
	default:
		return nil, nil
	}
	if len(items) == 0 {
		return []*dal.Record{}, nil
	}

	var results []*dal.Record
	err := s.inTransaction(ctx, func(repo *dal.Repository) error {
		keyed := dal.NewRecord()
		var or []any
		var keys []string
		for _, item := range items {
			rec, _ := item.(*dal.Record)
			if rec == nil {
				continue
			}
			if selector, ok := rec.Get("selector"); ok {
				data, _ := rec.Get("data")
				dataRec, _ := data.(*dal.Record)
				sel, _ := selector.(*dal.Record)
				found, err := s.updateMatching(ctx, repo, sel, dataRec, nil)
				if err != nil {
					return err
				}
				results = append(results, found...)
				continue
			}
			criteria := dal.NewRecord()
			parts := make([]string, len(pks))
			for i, pk := range pks {
				v, _ := rec.Get(pk)
				criteria.Set(pk, v)
				parts[i] = fmt.Sprint(v)
			}
			key := strings.Join(parts, ":")
			keyed.Set(key, rec)
			keys = append(keys, key)
			or = append(or, criteria)
		}
		if len(or) == 0 {
			return nil
		}
		where := dal.NewRecord()
		where.Set("$or", or)
		found, err := s.updateMatching(ctx, repo, where, nil, keyed)
		if err != nil {
			return err
		}
		if len(found) != len(keys) {
			foundKeys := map[string]bool{}
			for _, f := range found {
				parts := make([]string, len(pks))
				for i, pk := range pks {
					v, _ := f.Get(pk)
					parts[i] = fmt.Sprint(v)
				}
				foundKeys[strings.Join(parts, ":")] = true
			}
			var missing []string
			for _, k := range keys {
				if !foundKeys[k] {
					missing = append(missing, strings.ReplaceAll(k, ":", " - "))
				}
			}
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND,
				fmt.Sprintf(`%s with %s "%s" not found`, e.Meta.ClassName, strings.Join(pks, ", "), strings.Join(missing, ", ")))
		}
		results = append(results, found...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if shouldReturnArray {
		return results, nil
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// updateMatching updates the entities matching where with data (selector
// form) or with the data keyed by their primary key values.
func (s *InternalService) updateMatching(ctx context.Context, repo *dal.Repository, where, data *dal.Record, keyed *dal.Record) ([]*dal.Record, error) {
	config := FindConfig{}
	s.applyDefaultOrdering(&config)
	opts := buildQuery(where, config)
	pks := s.model().Meta.PrimaryKey
	return repo.UpdateWith(ctx, opts, func(current dal.Row) *dal.Record {
		var update *dal.Record
		if data != nil {
			update = data
		} else {
			parts := make([]string, len(pks))
			for i, pk := range pks {
				parts[i] = fmt.Sprint(current[pk])
			}
			v, _ := keyed.Get(strings.Join(parts, ":"))
			update, _ = v.(*dal.Record)
		}
		if update == nil {
			return nil
		}
		if md, ok := update.Get("metadata"); ok {
			if mdRec, ok := md.(*dal.Record); ok && mdRec.Len() > 0 {
				copyRec := dal.NewRecord()
				for _, k := range update.Keys() {
					v, _ := update.Get(k)
					copyRec.Set(k, v)
				}
				copyRec.Set("metadata", mergeMetadata(current["metadata"], mdRec))
				update = copyRec
			}
		}
		return update
	})
}

func primaryKeyCriteria(e *dml.Entity, ids any) *dal.Record {
	list, ok := ids.([]any)
	if !ok {
		list = []any{ids}
	}
	or := make([]any, 0, len(list))
	for _, id := range list {
		c := dal.NewRecord()
		if rec, isRec := id.(*dal.Record); isRec {
			for _, k := range rec.Keys() {
				v, _ := rec.Get(k)
				c.Set(k, v)
			}
		} else {
			c.Set(e.Meta.PrimaryKey[0], id)
		}
		or = append(or, c)
	}
	where := dal.NewRecord()
	where.Set("$or", or)
	return where
}

// Delete mirrors delete.
func (s *InternalService) Delete(ctx context.Context, idsOrSelector any) ([]any, error) {
	if list, ok := idsOrSelector.([]any); ok && len(list) == 0 || idsOrSelector == nil {
		return []any{}, nil
	}
	where := primaryKeyCriteria(s.model(), idsOrSelector)
	if rec, ok := idsOrSelector.(*dal.Record); ok && rec.Has("selector") {
		sel, _ := rec.Get("selector")
		where = dal.NewRecord()
		where.Set("$or", []any{sel})
	}
	var ids []any
	err := s.inTransaction(ctx, func(repo *dal.Repository) error {
		var err error
		ids, err = repo.Delete(ctx, where)
		return err
	})
	return ids, err
}

// SoftDelete mirrors softDelete.
func (s *InternalService) SoftDelete(ctx context.Context, ids any) (*dal.CascadeResult, error) {
	if list, ok := ids.([]any); ok && len(list) == 0 || ids == nil {
		return &dal.CascadeResult{}, nil
	}
	var res *dal.CascadeResult
	err := s.inTransaction(ctx, func(repo *dal.Repository) error {
		var err error
		res, err = repo.SoftDelete(ctx, primaryKeyCriteria(s.model(), ids))
		return err
	})
	return res, err
}

// Restore mirrors restore.
func (s *InternalService) Restore(ctx context.Context, ids any) (*dal.CascadeResult, error) {
	var res *dal.CascadeResult
	err := s.inTransaction(ctx, func(repo *dal.Repository) error {
		var err error
		res, err = repo.Restore(ctx, primaryKeyCriteria(s.model(), ids))
		return err
	})
	return res, err
}

// Upsert mirrors upsert.
func (s *InternalService) Upsert(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	var out []*dal.Record
	err := s.inTransaction(ctx, func(repo *dal.Repository) error {
		var err error
		out, err = repo.Upsert(ctx, data)
		return err
	})
	return out, err
}
