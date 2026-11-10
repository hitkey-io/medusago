package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/index/src/utils"
)

// EntityRemover is the part of the DataSynchronizer the configuration
// check uses.
type EntityRemover interface {
	RemoveEntities(ctx context.Context, entities []string, staleOnly bool) error
}

// Configuration mirrors Configuration: compares the indexed entities with
// index_metadata and schedules the changed ones.
type Configuration struct {
	Schema           *utils.SchemaObjectRepresentation
	IndexMetadata    *modulessdk.InternalService
	IndexSync        *modulessdk.InternalService
	DataSynchronizer EntityRemover
	Logger           *slog.Logger
}

func get(r *dal.Record, key string) any {
	v, _ := r.Get(key)
	return v
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// CheckChanges mirrors checkChanges: returns the index metadata rows that
// are pending, processing or in error.
func (c *Configuration) CheckChanges(ctx context.Context) ([]*dal.Record, error) {
	c.Logger.Info("[Index engine] Checking for index changes")

	currentConfig, err := c.IndexMetadata.List(ctx, dal.NewRecord(), modulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	currentConfigMap := map[string]*dal.Record{}
	var currentOrder []string
	for _, cfg := range currentConfig {
		entity := str(get(cfg, "entity"))
		if _, ok := currentConfigMap[entity]; !ok {
			currentOrder = append(currentOrder, entity)
		}
		currentConfigMap[entity] = cfg
	}

	entityPresent := map[string]bool{}
	var newConfig, updatedConfig []*dal.Record
	var deleted []string
	var idxSyncEntities []string

	for _, key := range c.Schema.Keys {
		rep := c.Schema.Entities[key]
		entity := rep.Entity
		// Array.prototype.sort mutates the representation's fields.
		sort.Strings(rep.Fields)
		fields := strings.Join(rep.Fields, ",")
		fieldsHash := common.SimpleHash(fields)

		existing := currentConfigMap[entity]
		entityPresent[entity] = true
		if existing == nil || str(get(existing, "fields_hash")) != fieldsHash {
			meta := dal.NewRecord()
			if existing != nil {
				meta.Set("id", get(existing, "id"))
			}
			meta.Set("entity", entity)
			meta.Set("fields", fields)
			meta.Set("fields_hash", fieldsHash)
			if existing == nil {
				newConfig = append(newConfig, meta)
			} else {
				meta.Set("status", utils.IndexMetadataStatusPending)
				updatedConfig = append(updatedConfig, meta)
			}
			idxSyncEntities = append(idxSyncEntities, entity)
		}
	}
	for _, entity := range currentOrder {
		if !entityPresent[entity] {
			deleted = append(deleted, entity)
		}
	}

	if len(newConfig) > 0 {
		if _, err := c.IndexMetadata.Create(ctx, newConfig); err != nil {
			return nil, err
		}
	}
	if len(updatedConfig) > 0 {
		items := make([]any, len(updatedConfig))
		for i, u := range updatedConfig {
			items[i] = u
		}
		if _, err := c.IndexMetadata.Update(ctx, items); err != nil {
			return nil, err
		}
	}
	if len(deleted) > 0 {
		for _, entity := range deleted {
			sel := dal.NewRecord()
			sel.Set("entity", entity)
			if _, err := c.IndexMetadata.Delete(ctx, sel); err != nil {
				return nil, err
			}
		}
		if err := c.DataSynchronizer.RemoveEntities(ctx, deleted, false); err != nil {
			return nil, err
		}
	}

	if len(idxSyncEntities) > 0 {
		filter := dal.NewRecord()
		entities := make([]any, len(idxSyncEntities))
		for i, e := range idxSyncEntities {
			entities[i] = e
		}
		filter.Set("entity", entities)
		existing, err := c.IndexSync.List(ctx, filter, modulessdk.FindConfig{})
		if err != nil {
			return nil, err
		}
		var data []*dal.Record
		for _, entity := range idxSyncEntities {
			row := dal.NewRecord()
			row.Set("entity", entity)
			row.Set("last_key", nil)
			for _, e := range existing {
				if str(get(e, "entity")) == entity {
					row.Set("id", get(e, "id"))
					break
				}
			}
			data = append(data, row)
		}
		if _, err := c.IndexSync.Upsert(ctx, data); err != nil {
			return nil, err
		}
	}

	filter := dal.NewRecord()
	filter.Set("status", []any{utils.IndexMetadataStatusPending, utils.IndexMetadataStatusProcessing, utils.IndexMetadataStatusError})
	changes, err := c.IndexMetadata.List(ctx, filter, modulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	plural := ""
	if len(changes) > 1 {
		plural = "s"
	}
	c.Logger.Info(fmt.Sprintf("[Index engine] Found %d index change%s that are either pending or processing", len(changes), plural))
	return changes, nil
}
