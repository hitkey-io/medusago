// Package services mirrors @medusajs/event-bus-redis/src/services.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/utils/src/bullmq"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// groupedEventsChunkSize mirrors GROUPED_EVENTS_CHUNK_SIZE.
const groupedEventsChunkSize = 1000

// Options mirrors EventBusRedisModuleOptions.
type Options struct {
	RedisURL     string
	RedisOptions map[string]any
	QueueName    string
	QueueOptions map[string]any
	// WorkerOptions are the bullmq worker options (concurrency...).
	WorkerOptions map[string]any
	// JobOptions are the global job options of every emit.
	JobOptions map[string]any
}

// OptionsFromMap reads the module options.
func OptionsFromMap(m map[string]any) Options {
	var o Options
	o.RedisURL, _ = m["redisUrl"].(string)
	o.RedisOptions, _ = m["redisOptions"].(map[string]any)
	o.QueueName, _ = m["queueName"].(string)
	o.QueueOptions, _ = m["queueOptions"].(map[string]any)
	o.WorkerOptions, _ = m["workerOptions"].(map[string]any)
	o.JobOptions, _ = m["jobOptions"].(map[string]any)
	return o
}

type subscriberDescriptor struct {
	id         string
	subscriber eventbus.Subscriber
}

// RedisEventBusService mirrors RedisEventBusService: events are jobs of a
// bullmq queue ("events-queue", prefix "RedisEventBusService") processed by
// the subscribers of whichever instance's worker takes them; grouped events
// are staged in the Redis list staging:<eventGroupId> until released.
type RedisEventBusService struct {
	logger       *slog.Logger
	client       *ioredis.Client
	queueName    string
	queue        *bullmq.Queue
	worker       *bullmq.Worker
	workerOpts   bullmq.WorkerOptions
	jobOptions   *bullmq.Obj
	isWorkerMode bool

	mu          sync.RWMutex
	subscribers map[string][]subscriberDescriptor
	wg          sync.WaitGroup
}

// NewRedisEventBusService mirrors the loader and the constructor.
func NewRedisEventBusService(ctx context.Context, logger *slog.Logger, opts Options) (*RedisEventBusService, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if opts.RedisURL == "" {
		return nil, errors.New("No `redisUrl` provided in project config. It is required for the Redis Event Bus.")
	}
	client := ioredis.New(opts.RedisURL, opts.RedisOptions)
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error(fmt.Sprintf("An error occurred while connecting to Redis in module 'event-bus-redis':\n %v", err))
	} else {
		logger.Info("Connection to Redis in module 'event-bus-redis' established")
	}
	queueName := opts.QueueName
	if queueName == "" {
		queueName = "events-queue"
	}
	qopts := bullmq.QueueOptions{Prefix: "RedisEventBusService"}
	if o := bullmq.QueueOptionsFromMap(opts.QueueOptions); o.Prefix != "" || o.DefaultJobOptions != nil || o.MaxLenEvents != 0 {
		if o.Prefix == "" {
			o.Prefix = qopts.Prefix
		}
		qopts = o
	}
	wopts := bullmq.WorkerOptionsFromMap(opts.WorkerOptions)
	if wopts.Prefix == "" {
		wopts.Prefix = "RedisEventBusService"
	}
	wopts.Logger = logger
	s := &RedisEventBusService{
		logger: logger, client: client, queueName: queueName,
		queue:        bullmq.NewQueue(ctx, queueName, client.Client, qopts),
		workerOpts:   wopts,
		jobOptions:   bullmq.ObjFromMap(opts.JobOptions),
		isWorkerMode: eventbus.IsWorkerMode(),
		subscribers:  map[string][]subscriberDescriptor{},
	}
	if s.jobOptions == nil {
		s.jobOptions = bullmq.NewObj()
	}
	return s, nil
}

// OnApplicationStart mirrors __hooks.onApplicationStart: the worker runs.
func (s *RedisEventBusService) OnApplicationStart(ctx context.Context) error {
	if !s.isWorkerMode || s.worker != nil {
		return nil
	}
	s.worker = bullmq.NewWorker(s.queueName, s.client.Client, s.process, s.workerOpts)
	s.worker.Run(ctx)
	return nil
}

// OnApplicationShutdown mirrors onApplicationPrepareShutdown +
// onApplicationShutdown: the worker finishes its jobs, the connection closes.
func (s *RedisEventBusService) OnApplicationShutdown(context.Context) error {
	if s.worker != nil {
		s.worker.Close()
		s.worker = nil
	}
	s.wg.Wait()
	return s.client.Close()
}

// Subscribe mirrors subscribe(eventName, subscriber) (subscriber id
// `${event}-${ulid()}`).
func (s *RedisEventBusService) Subscribe(event string, sub eventbus.Subscriber) {
	s.SubscribeWithID(event, event+"-"+ulid.Make().String(), sub)
}

// SubscribeWithID mirrors subscribe(eventName, subscriber, { subscriberId }).
func (s *RedisEventBusService) SubscribeWithID(event, id string, sub eventbus.Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.subscribers[event] {
		if d.id == id {
			panic(fmt.Sprintf("Subscriber with id %s already exists", id))
		}
	}
	s.subscribers[event] = append(s.subscribers[event], subscriberDescriptor{id: id, subscriber: sub})
}

func (s *RedisEventBusService) hasSubscribers(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers[name]) > 0 || len(s.subscribers["*"]) > 0
}

// Emit mirrors emit(eventsData).
func (s *RedisEventBusService) Emit(ctx context.Context, messages []composer.Message) error {
	return s.EmitWithOptions(ctx, messages, nil)
}

// EmitWithOptions mirrors emit(eventsData, options): options are bullmq
// job options plus groupedEventsTTL (seconds, 600) and internal.
func (s *RedisEventBusService) EmitWithOptions(ctx context.Context, messages []composer.Message, options *dal.Record) error {
	opts := bullmq.ObjFromMap(options)
	if opts == nil {
		opts = bullmq.NewObj()
	}
	opts = opts.Clone()
	ttl := 600
	if v, ok := opts.Get("groupedEventsTTL"); ok && v != nil {
		if n, ok := v.(int); ok {
			ttl = n
		} else if f, ok := v.(float64); ok {
			ttl = int(f)
		}
	}
	opts.Delete("groupedEventsTTL")

	var toEmit []composer.Message
	groups := map[string][]composer.Message{}
	var groupOrder []string
	for _, m := range messages {
		m.Metadata = eventbus.WithCreatedAtMetadata(m.Metadata)
		gid := ""
		if v, ok := m.Metadata.Get("eventGroupId"); ok && v != nil {
			gid, _ = v.(string)
		}
		if gid == "" {
			toEmit = append(toEmit, m)
			continue
		}
		if _, seen := groups[gid]; !seen {
			groupOrder = append(groupOrder, gid)
		}
		groups[gid] = append(groups[gid], m)
	}
	var withSubscribers []composer.Message
	for _, m := range toEmit {
		if s.hasSubscribers(m.Name) {
			m.Metadata = eventbus.WithPublishedAtMetadata(m.Metadata)
			withSubscribers = append(withSubscribers, m)
		}
	}
	if len(withSubscribers) > 0 {
		var jobs []bullmq.BulkJob
		for _, e := range s.buildEvents(withSubscribers, opts) {
			jobs = append(jobs, bullmq.BulkJob{Name: e.name, Data: e.data, Opts: e.opts})
		}
		if _, err := s.queue.AddBulk(ctx, jobs); err != nil {
			return err
		}
	}
	for _, gid := range groupOrder {
		if err := s.groupEvents(ctx, gid, s.buildEvents(groups[gid], opts), ttl); err != nil {
			return err
		}
	}
	return nil
}

type builtEvent struct {
	name string
	data *bullmq.Obj
	opts *bullmq.Obj
}

func (e builtEvent) json() ([]byte, error) {
	return bullmq.Stringify(bullmq.NewObj("data", e.data, "name", e.name, "opts", e.opts))
}

// buildEvents mirrors buildEvents: default, module and emit options, then
// the message options; the payload is { data, metadata }.
func (s *RedisEventBusService) buildEvents(messages []composer.Message, options *bullmq.Obj) []builtEvent {
	priority := eventbus.EventPriority.DEFAULT
	if v, ok := options.Get("internal"); ok && v == true {
		priority = eventbus.EventPriority.LOWEST
	}
	opts := bullmq.NewObj("removeOnComplete", true, "attempts", 1, "priority", priority).Assign(s.jobOptions, options)
	out := make([]builtEvent, 0, len(messages))
	for _, m := range messages {
		var metadata any
		if m.Metadata != nil {
			metadata = m.Metadata
		}
		event := bullmq.NewObj("data", m.Data, "metadata", metadata)
		final := opts.Clone().Assign(bullmq.ObjFromMap(m.Options))
		if p, ok := final.Get("priority"); ok && p != nil {
			n := toFloat(p)
			if n < 1 || n > float64(eventbus.EventPriority.LOWEST) {
				s.logger.Warn(fmt.Sprintf("Invalid priority value: %v for event %s. Must be between 1 and %d", p, m.Name, eventbus.EventPriority.LOWEST))
				final.Set("priority", eventbus.EventPriority.DEFAULT)
				s.logger.Warn(fmt.Sprintf("Setting priority to default value: %d for event %s", eventbus.EventPriority.DEFAULT, m.Name))
			}
		}
		out = append(out, builtEvent{name: m.Name, data: event, opts: final})
	}
	return out
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	}
	return 0
}

func (s *RedisEventBusService) stagingKey(eventGroupID string) string {
	return s.client.KeyPrefix + "staging:" + eventGroupID
}

// groupEvents mirrors groupEvents: RPUSH then EXPIRE in one pipeline.
func (s *RedisEventBusService) groupEvents(ctx context.Context, eventGroupID string, events []builtEvent, ttl int) error {
	key := s.stagingKey(eventGroupID)
	values := make([]any, 0, len(events))
	for _, e := range events {
		b, err := e.json()
		if err != nil {
			return err
		}
		values = append(values, string(b))
	}
	pipe := s.client.Pipeline()
	pipe.RPush(ctx, key, values...)
	pipe.Expire(ctx, key, secondsDuration(ttl))
	_, err := pipe.Exec(ctx)
	return err
}

type stagedEvent struct {
	raw  string
	name string
	data *bullmq.Obj
	opts *bullmq.Obj
}

func (s *RedisEventBusService) getGroupedEvents(ctx context.Context, eventGroupID string) ([]stagedEvent, error) {
	list, err := s.client.LRange(ctx, s.stagingKey(eventGroupID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]stagedEvent, 0, len(list))
	for _, raw := range list {
		v, err := bullmq.ParseJSON([]byte(raw))
		if err != nil {
			return nil, err
		}
		o, _ := v.(*bullmq.Obj)
		if o == nil {
			continue
		}
		e := stagedEvent{raw: raw}
		if n, ok := o.Get("name"); ok {
			e.name, _ = n.(string)
		}
		if d, ok := o.Get("data"); ok {
			e.data, _ = d.(*bullmq.Obj)
		}
		if op, ok := o.Get("opts"); ok {
			e.opts, _ = op.(*bullmq.Obj)
		}
		out = append(out, e)
	}
	return out, nil
}

// ReleaseGroupedEvents mirrors releaseGroupedEvents.
func (s *RedisEventBusService) ReleaseGroupedEvents(ctx context.Context, eventGroupID string) error {
	events, err := s.getGroupedEvents(ctx, eventGroupID)
	if err != nil {
		return err
	}
	var jobs []bullmq.BulkJob
	for _, e := range events {
		if !s.hasSubscribers(e.name) {
			continue
		}
		data := bullmq.NewObj()
		if e.data != nil {
			data = e.data.Clone()
		}
		md, _ := data.Get("metadata")
		data.Set("metadata", withPublishedAt(md))
		jobs = append(jobs, bullmq.BulkJob{Name: e.name, Data: data, Opts: e.opts})
	}
	if len(jobs) > 0 {
		if _, err := s.queue.AddBulk(ctx, jobs); err != nil {
			return err
		}
	}
	return s.ClearGroupedEvents(ctx, eventGroupID, nil)
}

func withPublishedAt(metadata any) *bullmq.Obj {
	out := bullmq.NewObj()
	if o, ok := metadata.(*bullmq.Obj); ok && o != nil {
		out = o.Clone()
	}
	out.Set("published_at", nowTime())
	return out
}

// ClearGroupedEvents mirrors clearGroupedEvents: the whole group, or only
// the events of the given names.
func (s *RedisEventBusService) ClearGroupedEvents(ctx context.Context, eventGroupID string, eventNames []string) error {
	if eventGroupID == "" {
		return nil
	}
	key := s.stagingKey(eventGroupID)
	if len(eventNames) == 0 {
		return s.client.Unlink(ctx, key).Err()
	}
	drop := map[string]bool{}
	for _, n := range eventNames {
		drop[n] = true
	}
	events, err := s.getGroupedEvents(ctx, eventGroupID)
	if err != nil {
		return err
	}
	ttl, err := s.client.PTTL(ctx, key).Result()
	if err != nil {
		return err
	}
	var keep []any
	for _, e := range events {
		if !drop[e.name] {
			b, err := bullmq.Stringify(bullmq.NewObj("data", e.data, "name", e.name, "opts", e.opts))
			if err != nil {
				return err
			}
			keep = append(keep, string(b))
		}
	}
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	for i := 0; i < len(keep); i += groupedEventsChunkSize {
		end := min(i+groupedEventsChunkSize, len(keep))
		pipe.RPush(ctx, key, keep[i:end]...)
	}
	if len(keep) > 0 && ttl > 0 {
		pipe.PExpire(ctx, key, ttl)
	}
	_, err = pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return nil
}

// process mirrors worker_: the subscribers of the event that did not
// complete in an earlier attempt run concurrently; when some fail and
// attempts remain, the completed ones are recorded in the job data and the
// job fails to be retried.
func (s *RedisEventBusService) process(ctx context.Context, job *bullmq.Job) (any, error) {
	s.wg.Add(1)
	defer s.wg.Done()
	name := job.Name
	s.mu.RLock()
	eventSubscribers := append([]subscriberDescriptor{}, s.subscribers[name]...)
	wildcard := append([]subscriberDescriptor{}, s.subscribers["*"]...)
	s.mu.RUnlock()
	all := append(append([]subscriberDescriptor{}, eventSubscribers...), wildcard...)

	decoded, err := dal.DecodeOrderedJSON(job.RawData)
	if err != nil {
		return nil, err
	}
	data, _ := decoded.(*dal.Record)
	if data == nil {
		data = dal.NewRecord()
	}
	completed := map[string]bool{}
	var completedList []any
	if ids, ok := data.Get("completedSubscriberIds"); ok {
		if arr, ok := ids.([]any); ok {
			for _, id := range arr {
				if sid, ok := id.(string); ok {
					completed[sid] = true
					completedList = append(completedList, sid)
				}
			}
		}
	}
	var inAttempt []subscriberDescriptor
	for _, sub := range all {
		if sub.id != "" && !completed[sub.id] {
			inAttempt = append(inAttempt, sub)
		}
	}
	currentAttempt := job.AttemptsMade
	isRetry := currentAttempt > 1
	configured := 0
	if v, ok := job.Opts.Get("attempts"); ok {
		configured = int(toFloat(v))
	}
	isFinalAttempt := currentAttempt == configured
	internal, _ := job.Opts.Get("internal")
	if internal != true {
		if isRetry {
			if isFinalAttempt {
				s.logger.Info(fmt.Sprintf("Final retry attempt for %s", name))
			}
			s.logger.Info(fmt.Sprintf("Retrying %s which has %d subscribers (%d of them failed)", name, len(eventSubscribers), len(inAttempt)))
		} else {
			priorityInfo := ""
			if p, ok := job.Opts.Get("priority"); ok && p != nil {
				priorityInfo = fmt.Sprintf(" (priority: %v)", p)
			}
			s.logger.Info(fmt.Sprintf("Processing %s%s which has %d subscribers", name, priorityInfo, len(eventSubscribers)))
		}
	}
	md, _ := data.Get("metadata")
	metadata, _ := md.(*dal.Record)
	payload, _ := data.Get("data")
	event := eventbus.Event{Name: name, Data: payload, Metadata: eventbus.ParseEventMetadataDates(metadata)}

	results := make([]any, len(inAttempt))
	var mu sync.Mutex
	var completedNow []any
	var wg sync.WaitGroup
	for i, sub := range inAttempt {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := callSubscriber(ctx, sub.subscriber, event)
			if err != nil {
				s.logger.Warn(fmt.Sprintf("An error occurred while processing %s:", name))
				s.logger.Warn(err.Error())
				results[i] = map[string]any{}
				return
			}
			mu.Lock()
			completedNow = append(completedNow, sub.id)
			mu.Unlock()
		}()
	}
	wg.Wait()
	didFail := len(completedNow) != len(inAttempt)
	shouldRetry := didFail && configured > 1 && !isFinalAttempt
	if shouldRetry {
		data.Set("completedSubscriberIds", append(completedList, completedNow...))
		if err := job.UpdateData(ctx, data); err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("One or more subscribers of %s failed. Retrying...", name)
		s.logger.Warn(msg)
		return nil, errors.New(msg)
	}
	if didFail && !isFinalAttempt {
		s.logger.Warn(fmt.Sprintf("One or more subscribers of %s failed. Retrying is not configured. Use 'attempts' option when emitting events.", name))
	}
	return results, nil
}

func callSubscriber(ctx context.Context, sub eventbus.Subscriber, e eventbus.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return sub(ctx, e)
}

// Wait waits for the jobs being processed by this instance (tests).
func (s *RedisEventBusService) Wait() { s.wg.Wait() }

func secondsDuration(n int) time.Duration { return time.Duration(n) * time.Second }

func nowTime() time.Time { return time.Now() }
