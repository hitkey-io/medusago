package bullmq

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// Processor processes a job; its result is stored as the job return value.
type Processor func(ctx context.Context, job *Job) (any, error)

// WorkerOptions mirrors the WorkerOptions used by the Medusa modules.
type WorkerOptions struct {
	Prefix          string
	Concurrency     int
	LockDuration    int64
	StalledInterval int64
	MaxStalledCount int
	// DrainDelay is in seconds (5).
	DrainDelay       float64
	RemoveOnComplete any
	RemoveOnFail     any
	Name             string
	Logger           *slog.Logger
}

// WorkerOptionsFromMap reads worker options given as module options.
func WorkerOptionsFromMap(m map[string]any) WorkerOptions {
	var o WorkerOptions
	if m == nil {
		return o
	}
	if p, ok := m["prefix"].(string); ok {
		o.Prefix = p
	}
	num := func(k string) float64 {
		if v, ok := m[k]; ok {
			return toNumber(v)
		}
		return 0
	}
	o.Concurrency = int(num("concurrency"))
	o.LockDuration = int64(num("lockDuration"))
	o.StalledInterval = int64(num("stalledInterval"))
	o.MaxStalledCount = int(num("maxStalledCount"))
	o.DrainDelay = num("drainDelay")
	o.RemoveOnComplete = normalize(m["removeOnComplete"])
	o.RemoveOnFail = normalize(m["removeOnFail"])
	if n, ok := m["name"].(string); ok {
		o.Name = n
	}
	return o
}

// Worker mirrors bullmq's Worker: it moves jobs to active (with a lock),
// processes them with the given concurrency, completes or fails them
// (retries with backoff), renews the locks of the running jobs and moves
// stalled jobs back to wait.
type Worker struct {
	queue     *Queue
	processor Processor
	opts      WorkerOptions
	id        string
	tokenSeq  atomic.Int64
	closing   atomic.Bool
	cancel    context.CancelFunc
	done      chan struct{}
	inflight  sync.Map
	wg        sync.WaitGroup
	logger    *slog.Logger

	drained    bool
	blockUntil int64
}

// NewWorker mirrors new Worker(name, processor, { autorun: false, ... }).
func NewWorker(name string, client redis.UniversalClient, processor Processor, opts WorkerOptions) *Worker {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.LockDuration <= 0 {
		opts.LockDuration = 30000
	}
	if opts.StalledInterval <= 0 {
		opts.StalledInterval = 30000
	}
	if opts.MaxStalledCount <= 0 {
		opts.MaxStalledCount = 1
	}
	if opts.DrainDelay <= 0 {
		opts.DrainDelay = 5
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	q := newQueueBase(name, client, QueueOptions{Prefix: opts.Prefix})
	return &Worker{queue: q, processor: processor, opts: opts, id: newUUID(), logger: logger}
}

// Queue returns the worker's queue keys.
func (w *Worker) Queue() *Queue { return w.queue }

// Run mirrors worker.run(): it starts the loops and returns immediately.
func (w *Worker) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	w.cancel = cancel
	w.done = make(chan struct{})
	go w.stalledLoop(ctx)
	go w.lockLoop(ctx)
	go w.mainLoop(ctx)
}

// Close mirrors worker.close(): no new jobs are fetched; the running jobs
// finish.
func (w *Worker) Close() {
	if w.closing.Swap(true) {
		return
	}
	if w.cancel != nil {
		w.cancel()
		<-w.done
	}
	w.wg.Wait()
}

func (w *Worker) mainLoop(ctx context.Context) {
	defer close(w.done)
	sem := make(chan struct{}, w.opts.Concurrency)
	for !w.closing.Load() {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		job, err := w.nextJob(ctx)
		if err != nil {
			<-sem
			if ctx.Err() != nil {
				return
			}
			w.logger.Error(fmt.Sprintf("bullmq worker %s: %v", w.queue.Name, err))
			sleep(ctx, 15*time.Second)
			continue
		}
		if job == nil {
			<-sem
			continue
		}
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			defer func() { <-sem }()
			w.processJob(context.WithoutCancel(ctx), job)
		}()
	}
}

// nextJob mirrors _getNextJob: moveToActive, or block on the marker while
// the queue is drained.
func (w *Worker) nextJob(ctx context.Context) (*Job, error) {
	for !w.closing.Load() {
		if w.drained {
			bu, err := w.waitForJob(ctx, w.blockUntil)
			if err != nil {
				return nil, err
			}
			w.blockUntil = bu
			if !(bu <= 0 || bu-time.Now().UnixMilli() < 1) {
				continue
			}
		}
		job, err := w.moveToActive(ctx)
		if err != nil {
			return nil, err
		}
		if job == nil {
			w.drained = true
			continue
		}
		w.drained = false
		return job, nil
	}
	return nil, nil
}

func (w *Worker) blockTimeout(blockUntil int64) float64 {
	if blockUntil > 0 {
		d := blockUntil - time.Now().UnixMilli()
		switch {
		case d <= 0:
			return float64(d)
		case d < 1:
			return 0.001
		default:
			return math.Min(float64(d)/1000, 10)
		}
	}
	return math.Max(w.opts.DrainDelay, 0.001)
}

// waitForJob mirrors waitForJob: BZPOPMIN on the marker.
func (w *Worker) waitForJob(ctx context.Context, blockUntil int64) (int64, error) {
	timeout := w.blockTimeout(blockUntil)
	if timeout <= 0 {
		return 0, nil
	}
	w.blockUntil = 0
	if timeout < 1 {
		// go-redis blocks for whole seconds: shorter waits sleep and
		// check again
		sleep(ctx, time.Duration(timeout*float64(time.Second)))
		return 0, ctx.Err()
	}
	// the blocking call does not observe the context: closing the worker
	// does not wait for it
	type popped struct {
		res *redis.ZWithKey
		err error
	}
	ch := make(chan popped, 1)
	go func() {
		res, err := w.queue.client.BZPopMin(context.WithoutCancel(ctx), time.Duration(timeout*float64(time.Second)), w.queue.keys["marker"]).Result()
		ch <- popped{res, err}
	}()
	var res *redis.ZWithKey
	var err error
	select {
	case p := <-ch:
		res, err = p.res, p.err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, err
	}
	if res != nil && res.Member != nil {
		return int64(res.Score), nil
	}
	return 0, nil
}

// moveToActive mirrors worker.moveToActive.
func (w *Worker) moveToActive(ctx context.Context) (*Job, error) {
	q := w.queue
	token := fmt.Sprintf("%s:%d", w.id, w.tokenSeq.Add(1)-1)
	var name any
	if w.opts.Name != "" {
		name = w.opts.Name
	}
	v, err := runCommand(ctx, q.client, "moveToActive", q.keys["wait"], q.keys["active"], q.keys["prioritized"], q.keys["events"], q.keys["stalled"],
		q.keys["limiter"], q.keys["delayed"], q.keys["paused"], q.keys["meta"], q.keys["pc"], q.keys["marker"],
		q.ToKey(""), time.Now().UnixMilli(), pack(NewObj("token", token, "lockDuration", w.opts.LockDuration, "limiter", nil, "name", name))).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	return w.nextJobFromReply(ctx, v, token)
}

// nextJobFromReply mirrors raw2NextJobData + nextJobFromJobData (the next
// iteration of a repeatable job is scheduled when one is fetched).
func (w *Worker) nextJobFromReply(ctx context.Context, v any, token string) (*Job, error) {
	arr, _ := v.([]any)
	if len(arr) == 0 {
		return nil, nil
	}
	var limitUntil, delayUntil int64
	if len(arr) > 2 {
		limitUntil, _ = replyInt(arr[2])
	}
	if len(arr) > 3 {
		delayUntil, _ = replyInt(arr[3])
	}
	_ = limitUntil
	w.blockUntil = max(delayUntil, 0)
	fields, _ := arr[0].([]any)
	if len(fields) == 0 {
		return nil, nil
	}
	id := ""
	if len(arr) > 1 {
		id = argString(arr[1])
	}
	job := jobFromArray(w.queue, fields, id)
	job.Token = token
	if job.Opts.Has("repeat") {
		if _, err := w.queue.updateRepeatableJob(ctx, job.Name, job.Data, job.Opts, false); err != nil {
			w.logger.Error(fmt.Sprintf("bullmq: failed to schedule the next repeatable job %s: %v", job.Name, err))
		}
	}
	return job, nil
}

// processJob mirrors processJob: completed or failed.
func (w *Worker) processJob(ctx context.Context, job *Job) {
	w.inflight.Store(job, time.Now().UnixMilli())
	defer w.inflight.Delete(job)
	result, err := callProcessor(ctx, w.processor, job)
	if err == nil {
		if cerr := w.moveToCompleted(ctx, job, result); cerr != nil {
			w.logger.Error(fmt.Sprintf("bullmq: %v", cerr))
		}
		return
	}
	if ferr := w.moveToFailed(ctx, job, err); ferr != nil {
		w.logger.Error(fmt.Sprintf("bullmq: %v", ferr))
	}
}

func callProcessor(ctx context.Context, p Processor, job *Job) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	return p(ctx, job)
}

// lockLoop mirrors startLockExtenderTimer: the locks of the running jobs
// are renewed every lockRenewTime / 2.
func (w *Worker) lockLoop(ctx context.Context) {
	renew := time.Duration(w.opts.LockDuration/2) * time.Millisecond
	t := time.NewTicker(renew / 2)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		now := time.Now().UnixMilli()
		w.inflight.Range(func(k, v any) bool {
			job := k.(*Job)
			if v.(int64)+int64(renew/time.Millisecond)/2 < now {
				w.inflight.Store(job, now)
				q := w.queue
				res, err := runCommand(ctx, q.client, "extendLock", q.ToKey(job.ID)+":lock", q.keys["stalled"], job.Token, w.opts.LockDuration, job.ID).Result()
				if n, _ := replyInt(res); err != nil || n == 0 {
					w.logger.Error(fmt.Sprintf("could not renew lock for job %s", job.ID))
				}
			}
			return true
		})
	}
}

// stalledLoop mirrors startStalledCheckTimer.
func (w *Worker) stalledLoop(ctx context.Context) {
	for {
		q := w.queue
		_, err := runCommand(ctx, q.client, "moveStalledJobsToWait", q.keys["stalled"], q.keys["wait"], q.keys["active"], q.keys["failed"],
			q.keys["stalled-check"], q.keys["meta"], q.keys["paused"], q.keys["marker"], q.keys["events"],
			w.opts.MaxStalledCount, q.ToKey(""), time.Now().UnixMilli(), w.opts.StalledInterval).Result()
		if err != nil && ctx.Err() == nil && !errors.Is(err, redis.Nil) {
			w.logger.Error(fmt.Sprintf("bullmq: stalled check of %s: %v", q.Name, err))
		}
		if !sleep(ctx, time.Duration(w.opts.StalledInterval)*time.Millisecond) {
			return
		}
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// newUUID mirrors uuid v4 (the worker id).
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
