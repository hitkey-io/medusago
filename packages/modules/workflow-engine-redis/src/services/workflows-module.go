// Package services mirrors @medusajs/workflow-engine-redis/src/services.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src/models"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src/types"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src/utils"
)

// WorkflowsModuleService mirrors WorkflowsModuleService of the Redis engine:
// the generated WorkflowExecution service, the Redis transaction storage
// (checkpoints, scheduler, cleaner) and the orchestrator (cross-instance
// notifications and async step responses).
type WorkflowsModuleService struct {
	*utilsmodulessdk.MedusaService
	*WorkflowOrchestrator
	storage *utils.RedisDistributedTransactionStorage
	clients []*ioredis.Client
	logger  *slog.Logger
}

var (
	_ composer.ExecutionStore         = (*WorkflowsModuleService)(nil)
	_ composer.StepCoordinator        = (*WorkflowsModuleService)(nil)
	_ composer.WorkflowScheduler      = (*WorkflowsModuleService)(nil)
	_ composer.WorkflowEventPublisher = (*WorkflowsModuleService)(nil)
)

// NewWorkflowsModuleService mirrors the redis loader and the service.
func NewWorkflowsModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*WorkflowsModuleService, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	o := types.OptionsFromMap(opts.Options)
	url := o.RedisURL
	if url == "" {
		url = o.URL
	}
	redisOptions := o.RedisOptions
	if redisOptions == nil {
		redisOptions = o.Options
	}
	if o.URL != "" && o.RedisURL == "" {
		logger.Warn("[Workflow-engine-redis] The `url` option is deprecated. Please use `redisUrl` instead for consistency with other modules.")
	}
	if o.Options != nil && o.RedisOptions == nil {
		logger.Warn("[Workflow-engine-redis] The `options` option is deprecated. Please use `redisOptions` instead for consistency with other modules.")
	}
	if url == "" {
		return nil, errors.New("No `redis.redisUrl` (or deprecated `redis.url`) provided in `workflowOrchestrator` module options. It is required for the Workflow Orchestrator Redis.")
	}
	pubURL, pubOptions := url, redisOptions
	if o.PubSubURL != "" {
		pubURL, pubOptions = o.PubSubURL, o.PubSubOptions
	}
	queueName := o.QueueName
	if queueName == "" {
		queueName = "medusa-workflows"
	}
	jobQueueName := o.JobQueueName
	if jobQueueName == "" {
		jobQueueName = "medusa-workflows-jobs"
	}
	ctx := context.Background()
	connection := ioredis.New(url, redisOptions)
	workerConnection := ioredis.New(url, redisOptions)
	if err := connection.Ping(ctx).Err(); err != nil {
		logger.Error(fmt.Sprintf("[Workflow-engine-redis] An error occurred while connecting to Redis in module 'workflow-engine-redis': %v", err))
	} else {
		logger.Info("[Workflow-engine-redis] Connection to Redis in module 'workflow-engine-redis' established")
	}
	publisher := ioredis.New(pubURL, pubOptions)
	subscriber := ioredis.New(pubURL, pubOptions)
	var c composer.Container
	if opts.Container != nil {
		c = opts.Container
	}
	storage := utils.NewRedisDistributedTransactionStorage(ctx, utils.StorageOptions{
		DB: opts.DB, Client: connection, WorkerClient: workerConnection,
		QueueName: queueName, JobQueueName: jobQueueName,
		MainQueueOptions:     types.Merge(o.QueueOptions, o.MainQueueOptions),
		JobQueueOptions:      types.Merge(o.QueueOptions, o.JobQueueOptions),
		JobWorkerOptions:     types.Merge(o.WorkerOptions, o.JobWorkerOptions),
		CleanerQueueOptions:  types.Merge(o.QueueOptions, o.CleanerQueueOptions),
		CleanerWorkerOptions: types.Merge(o.WorkerOptions, o.CleanerWorkerOptions),
		Logger:               logger,
		IsWorkerMode:         eventbus.IsWorkerMode(),
		Container:            c,
	})
	return &WorkflowsModuleService{
		MedusaService:        utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig),
		WorkflowOrchestrator: NewWorkflowOrchestrator(publisher, subscriber, logger),
		storage:              storage,
		clients:              []*ioredis.Client{connection, workerConnection, publisher, subscriber},
		logger:               logger,
	}, nil
}

// OnApplicationStart mirrors onApplicationStart: the storage workers, the
// notifications subscription and the scheduled workflows.
func (s *WorkflowsModuleService) OnApplicationStart(ctx context.Context) error {
	if err := s.storage.OnApplicationStart(ctx); err != nil {
		return err
	}
	if err := s.WorkflowOrchestrator.Start(ctx); err != nil {
		return err
	}
	return composer.StartScheduler(s)
}

// OnApplicationShutdown mirrors onApplicationPrepareShutdown +
// onApplicationShutdown.
func (s *WorkflowsModuleService) OnApplicationShutdown(context.Context) error {
	composer.StopScheduler(s)
	s.storage.OnApplicationShutdown()
	s.WorkflowOrchestrator.Stop()
	for _, c := range s.clients {
		_ = c.Close()
	}
	return nil
}

// ContainerOptionError mirrors setStepSuccess / setStepFailure called with
// options.container: the options are JSON-cloned
// (JSON.parse(JSON.stringify(options))), the container loses its methods
// and registerStepSuccess throws a TypeError (500 unknown_error).
func (s *WorkflowsModuleService) ContainerOptionError() error {
	return errors.New("parentContainer?.createScope is not a function")
}

// SaveExecution implements composer.ExecutionStore.
func (s *WorkflowsModuleService) SaveExecution(ctx context.Context, row composer.ExecutionRow) error {
	return s.storage.SaveExecution(ctx, row)
}

// DeleteExecution implements composer.ExecutionStore.
func (s *WorkflowsModuleService) DeleteExecution(ctx context.Context, workflowID, transactionID, runID string) error {
	return s.storage.DeleteExecution(ctx, workflowID, transactionID, runID)
}

// Schedule implements composer.WorkflowScheduler.
func (s *WorkflowsModuleService) Schedule(jobID string, opts composer.SchedulerOptions) error {
	return s.storage.Schedule(jobID, opts)
}

// Remove implements composer.WorkflowScheduler.
func (s *WorkflowsModuleService) Remove(jobID string) error { return s.storage.Remove(jobID) }

// ClearExpiredExecutions mirrors clearExpiredExecutions.
func (s *WorkflowsModuleService) ClearExpiredExecutions(ctx context.Context) error {
	return s.storage.ClearExpiredExecutions(ctx)
}
