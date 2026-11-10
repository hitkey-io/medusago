package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	modulessdktypes "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	eventbuslocal "github.com/hitkey-io/medusago/packages/modules/event-bus-local/src/services"
	"github.com/hitkey-io/medusago/packages/modules/index/src/models"
	"github.com/hitkey-io/medusago/packages/modules/index/src/utils"
	indexsync "github.com/hitkey-io/medusago/packages/modules/index/src/utils/sync"
)

// eventBus is the part of the event bus module the index uses.
type eventBus interface {
	Subscribe(event string, sub eventbuslocal.Subscriber)
	Emit(ctx context.Context, messages []composer.Message) error
}

// syncSubscribers mirrors IndexModuleService.SyncSubscribersDescriptor.
var syncSubscribers = []struct{ eventName, method string }{
	{"index.continue-sync", "continueSync"},
	{"index.full-sync", "fullSync"},
	{"index.reset-sync", "resetSync"},
}

// IndexModuleService mirrors IndexModuleService.
type IndexModuleService struct {
	*modulessdk.MedusaService

	db           dal.Querier
	container    modulessdktypes.Resolver
	logger       *slog.Logger
	isWorkerMode bool

	indexMetadata *modulessdk.InternalService
	indexSync     *modulessdk.InternalService

	schema        *utils.SchemaObjectRepresentation
	storage       *PostgresProvider
	synchronizer  *DataSynchronizer
	configuration *indexsync.Configuration
}

// NewIndexModuleService builds the service. The worker mode comes from
// MEDUSA_WORKER_MODE (projectConfig.workerMode, "shared" by default):
// only "server" instances skip the synchronisation.
func NewIndexModuleService(opts modulessdktypes.ServiceOptions) (*IndexModuleService, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	base := modulessdk.NewMedusaService(opts.DB, models.All, nil)
	s := &IndexModuleService{
		MedusaService: base,
		db:            opts.DB,
		container:     opts.Container,
		logger:        logger,
		isWorkerMode:  workerMode() != "server",
		indexMetadata: base.Internal("IndexMetadata"),
		indexSync:     base.Internal("IndexSync"),
	}
	s.synchronizer = &DataSynchronizer{db: opts.DB, query: s.query, logger: logger, indexMetadata: s.indexMetadata, indexSync: s.indexSync}
	return s, nil
}

func workerMode() string {
	if v := os.Getenv("MEDUSA_WORKER_MODE"); v != "" {
		return v
	}
	return "shared"
}

func (s *IndexModuleService) query() *query.Query {
	if s.container == nil {
		return nil
	}
	q, _ := s.container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	return q
}

func (s *IndexModuleService) eventBus() eventBus {
	if s.container == nil {
		return nil
	}
	b, _ := s.container.Resolve(modulessdk.EVENT_BUS).(eventBus)
	return b
}

// OnApplicationStart mirrors the onApplicationStart hook: the partitions,
// the event listeners and, in worker mode, the configuration check and the
// synchronisation of the changed entities (awaited, before the server
// accepts requests).
func (s *IndexModuleService) OnApplicationStart(ctx context.Context) error {
	if err := s.onApplicationStart(ctx); err != nil {
		s.logger.Error(err.Error())
	}
	return nil
}

func (s *IndexModuleService) onApplicationStart(ctx context.Context) error {
	if s.schema == nil {
		schema, err := utils.BuildSchemaObjectRepresentation()
		if err != nil {
			return err
		}
		s.schema = schema
	}
	s.storage = NewPostgresProvider(s.db, s.query, s.schema)
	s.registerListeners()
	if err := s.storage.OnApplicationStart(ctx); err != nil {
		return err
	}
	if !s.isWorkerMode {
		return nil
	}
	s.synchronizer.OnApplicationStart(s.schema, s.storage)
	s.configuration = &indexsync.Configuration{
		Schema:           s.schema,
		IndexMetadata:    s.indexMetadata,
		IndexSync:        s.indexSync,
		DataSynchronizer: s.synchronizer,
		Logger:           s.logger,
	}
	changed, err := s.configuration.CheckChanges(ctx)
	if err != nil {
		return err
	}
	if len(changed) > 0 {
		return s.synchronizer.SyncEntities(ctx, changed)
	}
	return nil
}

func (s *IndexModuleService) registerListeners() {
	if !s.isWorkerMode {
		return
	}
	bus := s.eventBus()
	if bus == nil {
		return
	}
	for _, key := range s.schema.Keys {
		rep := s.schema.Entities[key]
		for _, listener := range rep.Listeners {
			bus.Subscribe(listener, s.storage.ConsumeEvent(rep))
		}
	}
	for _, sub := range syncSubscribers {
		method := sub.method
		bus.Subscribe(sub.eventName, func(ctx context.Context, _ eventbuslocal.Event) error {
			switch method {
			case "continueSync":
				return s.continueSync(ctx)
			case "fullSync":
				return s.fullSync(ctx)
			default:
				return s.resetSync(ctx)
			}
		})
	}
}

// Query mirrors query(config).
func (s *IndexModuleService) Query(ctx context.Context, config utils.IndexQueryConfig) (*QueryResult, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("Cannot read properties of undefined (reading 'query')")
	}
	return s.storage.Query(ctx, config)
}

// QueryIndex implements query.IndexService (query.index()).
func (s *IndexModuleService) QueryIndex(ctx context.Context, in query.IndexQueryInput) ([]any, *dal.Record, error) {
	res, err := s.Query(ctx, utils.IndexQueryConfig{
		Fields:      in.Fields,
		Filters:     in.Filters,
		JoinFilters: in.JoinFilters,
		Pagination:  in.Pagination,
		IdsOnly:     in.IdsOnly,
	})
	if err != nil {
		return nil, nil, err
	}
	return res.Data, res.Metadata, nil
}

// GetInfo mirrors getInfo: each indexed entity with its sync state.
func (s *IndexModuleService) GetInfo(ctx context.Context) ([]any, error) {
	metadata, err := s.indexMetadata.List(ctx, dal.NewRecord(), modulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	syncs, err := s.indexSync.List(ctx, dal.NewRecord(), modulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	lastKeys := map[string]any{}
	for _, sync := range syncs {
		if v, _ := sync.Get("last_key"); v != nil {
			e, _ := sync.Get("entity")
			lastKeys[fmt.Sprint(e)] = v
		}
	}
	out := make([]any, 0, len(metadata))
	for _, m := range metadata {
		entity, _ := m.Get("entity")
		fields, _ := m.Get("fields")
		fieldList := []any{}
		for _, f := range strings.Split(fmt.Sprint(fields), ",") {
			fieldList = append(fieldList, f)
		}
		info := dal.NewRecord()
		id, _ := m.Get("id")
		status, _ := m.Get("status")
		updatedAt, _ := m.Get("updated_at")
		info.Set("id", id)
		info.Set("entity", entity)
		info.Set("status", status)
		info.Set("fields", fieldList)
		info.Set("updated_at", updatedAt)
		if v, ok := lastKeys[fmt.Sprint(entity)]; ok {
			info.Set("last_synced_key", v)
		} else {
			info.Set("last_synced_key", nil)
		}
		out = append(out, info)
	}
	return out, nil
}

// Sync mirrors sync({ strategy }).
func (s *IndexModuleService) Sync(ctx context.Context, strategy string) error {
	if strategy != "" && strategy != "full" && strategy != "reset" {
		return fmt.Errorf(`Invalid sync strategy: %s. Must be "full" or "reset"`, strategy)
	}
	switch strategy {
	case "full":
		return s.fullSync(ctx)
	case "reset":
		return s.resetSync(ctx)
	default:
		return s.continueSync(ctx)
	}
}

func (s *IndexModuleService) emitInternal(ctx context.Context, name string) error {
	bus := s.eventBus()
	if bus == nil {
		return nil
	}
	options := dal.NewRecord()
	options.Set("internal", true)
	return bus.Emit(ctx, []composer.Message{{Name: name, Data: dal.NewRecord(), Options: options}})
}

func (s *IndexModuleService) resetStatuses(ctx context.Context) error {
	sel := dal.NewRecord()
	sel.Set("status", []any{utils.IndexMetadataStatusDone, utils.IndexMetadataStatusError, utils.IndexMetadataStatusProcessing})
	data := dal.NewRecord()
	data.Set("status", utils.IndexMetadataStatusPending)
	in := dal.NewRecord()
	in.Set("selector", sel)
	in.Set("data", data)
	_, err := s.indexMetadata.Update(ctx, in)
	return err
}

// syncChanges is the worker side of every strategy: the configuration
// check, then the synchronisation of what is not done.
func (s *IndexModuleService) syncChanges(ctx context.Context, failure string) error {
	if s.configuration == nil {
		return fmt.Errorf("[Index engine] %s", failure)
	}
	entities, err := s.configuration.CheckChanges(ctx)
	if err == nil && len(entities) > 0 {
		err = s.synchronizer.SyncEntities(ctx, entities)
	}
	if err != nil {
		s.logger.Error(err.Error())
		return fmt.Errorf("[Index engine] %s", failure)
	}
	return nil
}

func (s *IndexModuleService) continueSync(ctx context.Context) error {
	if !s.isWorkerMode {
		if err := s.resetStatuses(ctx); err != nil {
			return err
		}
		return s.emitInternal(ctx, "index.continue-sync")
	}
	return s.syncChanges(ctx, "Failed to continue sync")
}

func (s *IndexModuleService) fullSync(ctx context.Context) error {
	if !s.isWorkerMode {
		if err := s.resetStatuses(ctx); err != nil {
			return err
		}
		sel := dal.NewRecord()
		ne := dal.NewRecord()
		ne.Set("$ne", nil)
		sel.Set("last_key", ne)
		data := dal.NewRecord()
		data.Set("last_key", nil)
		in := dal.NewRecord()
		in.Set("selector", sel)
		in.Set("data", data)
		if _, err := s.indexSync.Update(ctx, in); err != nil {
			return err
		}
		return s.emitInternal(ctx, "index.full-sync")
	}
	return s.syncChanges(ctx, "Failed to full sync")
}

func (s *IndexModuleService) resetSync(ctx context.Context) error {
	if !s.isWorkerMode {
		if _, err := s.db.Exec(ctx, `TRUNCATE TABLE index_data, index_relation, index_metadata, index_sync CASCADE`); err != nil {
			return err
		}
		return s.emitInternal(ctx, "index.reset-sync")
	}
	return s.syncChanges(ctx, "Failed to reset sync")
}
