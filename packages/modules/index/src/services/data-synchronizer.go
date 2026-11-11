package services

import (
	"context"
	"errors"
	"fmt"
	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"log/slog"
	"time"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	eventbuslocal "github.com/hitkey-io/medusago/packages/modules/event-bus-local/src/services"
	"github.com/hitkey-io/medusago/packages/modules/index/src/utils"
	indexsync "github.com/hitkey-io/medusago/packages/modules/index/src/utils/sync"
)

// DataSynchronizer mirrors DataSynchronizer: (re)builds the index of the
// changed entities from Query.
type DataSynchronizer struct {
	db            dal.Querier
	query         func() *query.Query
	locking       func() lockingtypes.ILockingModule
	logger        *slog.Logger
	indexMetadata *modulessdk.InternalService
	indexSync     *modulessdk.InternalService

	isReady      bool
	schema       *utils.SchemaObjectRepresentation
	storage      *PostgresProvider
	orchestrator *indexsync.Orchestrator
}

// OnApplicationStart mirrors onApplicationStart.
func (d *DataSynchronizer) OnApplicationStart(schema *utils.SchemaObjectRepresentation, storage *PostgresProvider) {
	d.storage = storage
	d.schema = schema
	d.isReady = true
}

func (d *DataSynchronizer) isReadyOrThrow() error {
	if !d.isReady {
		return errors.New("DataSynchronizer is not ready. Call onApplicationStart first.")
	}
	return nil
}

// SyncEntities mirrors syncEntities: one orchestrated task per entity.
func (d *DataSynchronizer) SyncEntities(ctx context.Context, entities []*dal.Record) error {
	if err := d.isReadyOrThrow(); err != nil {
		return err
	}
	names := make([]string, 0, len(entities))
	for _, e := range entities {
		v, _ := e.Get("entity")
		names = append(names, fmt.Sprint(v))
	}
	d.orchestrator = indexsync.NewOrchestrator(d.locking(), names, 60, d.logger)
	return d.orchestrator.Process(ctx, d.taskRunner)
}

// RemoveEntities mirrors removeEntities.
func (d *DataSynchronizer) RemoveEntities(ctx context.Context, entities []string, staleOnly bool) error {
	if err := d.isReadyOrThrow(); err != nil {
		return err
	}
	stale := ""
	if staleOnly {
		stale = " AND staled_at IS NOT NULL"
	}
	for _, entity := range entities {
		if _, err := d.db.Exec(ctx, `WITH deleted_data AS (
          DELETE FROM "index_data"
          WHERE "name" = $1`+stale+`
          RETURNING id
        )
        DELETE FROM "index_relation"
        WHERE ("parent_name" = $1 AND "parent_id" IN (SELECT id FROM deleted_data))
           OR ("child_name" = $1 AND "child_id" IN (SELECT id FROM deleted_data))`, entity); err != nil {
			return err
		}
	}
	return nil
}

func (d *DataSynchronizer) updatedStatus(ctx context.Context, entity, status string) error {
	data := dal.NewRecord()
	data.Set("status", status)
	sel := dal.NewRecord()
	sel.Set("entity", entity)
	in := dal.NewRecord()
	in.Set("data", data)
	in.Set("selector", sel)
	_, err := d.indexMetadata.Update(ctx, in)
	return err
}

func (d *DataSynchronizer) updateLastKey(ctx context.Context, entity string, lastKey any) error {
	data := dal.NewRecord()
	data.Set("last_key", lastKey)
	sel := dal.NewRecord()
	sel.Set("entity", entity)
	in := dal.NewRecord()
	in.Set("data", data)
	in.Set("selector", sel)
	_, err := d.indexSync.Update(ctx, in)
	return err
}

type acknowledgement struct {
	lastCursor any
	done       bool
	err        error
}

func (d *DataSynchronizer) taskRunner(ctx context.Context, entity string) error {
	d.logger.Info(fmt.Sprintf("[Index engine] syncing entity '%s'", entity))

	filter := dal.NewRecord()
	filter.Set("entity", entity)
	cursors, err := d.indexSync.List(ctx, filter, modulessdk.FindConfig{Select: []string{"last_key"}})
	if err != nil {
		return err
	}
	if err := d.updatedStatus(ctx, entity, utils.IndexMetadataStatusProcessing); err != nil {
		return err
	}
	if _, err := d.db.Exec(ctx, `UPDATE "index_data" SET "staled_at" = NOW() WHERE "name" = $1`, entity); err != nil {
		return err
	}
	var cursor any
	if len(cursors) > 0 {
		cursor, _ = cursors[0].Get("last_key")
	}

	start := time.Now()
	chunkStart := start
	final := d.SyncEntity(ctx, entity, cursor, func(ack acknowledgement) error {
		elapsed := time.Since(chunkStart)
		if ack.lastCursor != nil {
			d.logger.Debug(fmt.Sprintf("[Index engine] syncing entity '%s' updating last cursor to %v (+%.2fms)", entity, ack.lastCursor, float64(elapsed.Microseconds())/1000))
			if err := d.updateLastKey(ctx, entity, ack.lastCursor); err != nil {
				return err
			}
			if !ack.done && ack.err == nil {
				d.orchestrator.RenewLock(ctx, entity)
			}
		}
		if ack.err != nil {
			d.logger.Error(fmt.Sprintf("[Index engine] syncing entity '%s' failed with error (+%.2fms):\n%s", entity, float64(elapsed.Microseconds())/1000, ack.err.Error()))
		}
		if ack.done {
			d.logger.Info(fmt.Sprintf("[Index engine] syncing entity '%s' done (+%.2fms)", entity, float64(time.Since(start).Microseconds())/1000))
		}
		chunkStart = time.Now()
		return nil
	})

	if final.done {
		if err := d.updatedStatus(ctx, entity, utils.IndexMetadataStatusDone); err != nil {
			return err
		}
		if err := d.updateLastKey(ctx, entity, final.lastCursor); err != nil {
			return err
		}
		if err := d.RemoveEntities(ctx, []string{entity}, true); err != nil {
			return err
		}
	}
	if final.err != nil {
		return d.updatedStatus(ctx, entity, utils.IndexMetadataStatusError)
	}
	return nil
}

// SyncEntity mirrors syncEntity: walks the entity by ascending id from the
// cursor, in batches of 100, feeding the storage as created (attached for
// links) events.
func (d *DataSynchronizer) SyncEntity(ctx context.Context, entityName string, cursor any, ack func(acknowledgement) error) acknowledgement {
	if err := d.isReadyOrThrow(); err != nil {
		return acknowledgement{lastCursor: cursor, err: err}
	}
	rep := d.schema.Entities[entityName]
	if rep == nil || rep.Alias == "" {
		d.logger.Info(fmt.Sprintf("[Index engine] Cannot find Entity '%s' alias. Skipping.", entityName))
		a := acknowledgement{lastCursor: cursor, done: true}
		_ = ack(a)
		return a
	}
	hasID := false
	for _, pk := range rep.ModuleConfig.PrimaryKeys {
		if pk == "id" {
			hasID = true
		}
	}
	if !hasID {
		a := acknowledgement{lastCursor: cursor, err: fmt.Errorf("Entity %s does not have a property 'id'. The 'id' must be provided and must be orderable (e.g ulid)", entityName)}
		_ = ack(a)
		return a
	}

	eventName := "*.created"
	if rep.ModuleConfig.IsLink {
		eventName = "*.attached"
	}
	consume := d.storage.ConsumeEvent(rep)
	currentCursor := cursor
	var syncErr error
	for {
		filters := dal.NewRecord()
		if currentCursor != nil && currentCursor != "" {
			gt := dal.NewRecord()
			gt.Set("$gt", currentCursor)
			filters.Set("id", gt)
		}
		order := dal.NewRecord()
		order.Set("id", "asc")
		pagination := dal.NewRecord()
		pagination.Set("order", order)
		pagination.Set("take", float64(100))
		res, err := d.query().Graph(ctx, query.GraphInput{Entity: rep.Alias, Fields: []string{"id"}, Filters: filters, Pagination: pagination}, joiner.Options{})
		if err != nil {
			syncErr = err
			break
		}
		if len(res.Data) == 0 {
			break
		}
		if err := consume(ctx, eventbuslocal.Event{Name: eventName, Data: res.Data}); err != nil {
			syncErr = err
			break
		}
		last, _ := res.Data[len(res.Data)-1].(*dal.Record)
		if last != nil {
			currentCursor, _ = last.Get("id")
		}
		if err := ack(acknowledgement{lastCursor: currentCursor}); err != nil {
			syncErr = err
			break
		}
	}
	if syncErr != nil {
		a := acknowledgement{lastCursor: currentCursor, err: syncErr}
		_ = ack(a)
		return a
	}
	a := acknowledgement{lastCursor: currentCursor, done: true}
	_ = ack(a)
	return a
}
