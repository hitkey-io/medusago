package bullmq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrorCode mirrors the bullmq ErrorCode enum (Lua return codes).
const (
	errJobNotExist               = -1
	errJobLockNotExist           = -2
	errJobNotInState             = -3
	errJobPendingDependencies    = -4
	errParentJobNotExist         = -5
	errJobLockMismatch           = -6
	errParentJobCannotBeReplaced = -7
)

// QueueOptions mirrors the QueueOptions used by the Medusa modules.
type QueueOptions struct {
	// Prefix defaults to "bull".
	Prefix string
	// DefaultJobOptions mirrors defaultJobOptions.
	DefaultJobOptions *Obj
	// MaxLenEvents mirrors streams.events.maxLen (10000).
	MaxLenEvents int
}

// QueueOptionsFromMap reads queue options given as module options
// (queueOptions: { prefix, defaultJobOptions, streams }).
func QueueOptionsFromMap(m map[string]any) QueueOptions {
	var o QueueOptions
	if m == nil {
		return o
	}
	if p, ok := m["prefix"].(string); ok {
		o.Prefix = p
	}
	if d := ObjFromMap(m["defaultJobOptions"]); d != nil {
		o.DefaultJobOptions = d
	}
	if s, ok := m["streams"].(map[string]any); ok {
		if ev, ok := s["events"].(map[string]any); ok {
			o.MaxLenEvents = int(toNumber(ev["maxLen"]))
		}
	}
	return o
}

// Queue mirrors bullmq's Queue.
type Queue struct {
	Name   string
	client redis.UniversalClient
	opts   QueueOptions
	keys   map[string]string
	prefix string
}

var keyTypes = []string{"", "active", "wait", "waiting-children", "paused", "id", "delayed", "prioritized", "stalled-check",
	"completed", "failed", "stalled", "repeat", "limiter", "meta", "events", "pc", "marker", "de"}

// NewQueue mirrors new Queue(name, opts): it writes the queue meta
// (opts.maxLenEvents) like the constructor does.
func NewQueue(ctx context.Context, name string, client redis.UniversalClient, opts QueueOptions) *Queue {
	q := newQueueBase(name, client, opts)
	maxLen := opts.MaxLenEvents
	if maxLen == 0 {
		maxLen = 10000
	}
	_ = client.HSet(ctx, q.keys["meta"], "opts.maxLenEvents", maxLen).Err()
	return q
}

func newQueueBase(name string, client redis.UniversalClient, opts QueueOptions) *Queue {
	if opts.Prefix == "" {
		opts.Prefix = "bull"
	}
	q := &Queue{Name: name, client: client, opts: opts, prefix: opts.Prefix}
	q.keys = map[string]string{}
	for _, t := range keyTypes {
		q.keys[t] = q.ToKey(t)
	}
	return q
}

// QualifiedName mirrors getQueueQualifiedName: "<prefix>:<name>".
func (q *Queue) QualifiedName() string { return q.prefix + ":" + q.Name }

// ToKey mirrors toKey(type).
func (q *Queue) ToKey(t string) string { return q.QualifiedName() + ":" + t }

// Client returns the Redis client.
func (q *Queue) Client() redis.UniversalClient { return q.client }

// Add mirrors queue.add(name, data, opts).
func (q *Queue) Add(ctx context.Context, name string, data any, opts *Obj) (*Job, error) {
	if opts != nil && opts.Has("repeat") {
		repeat := ObjFromMap(mustGet(opts, "repeat"))
		if ed, ok := repeat.Get("endDate"); ok && ed != nil {
			if ms, ok := dateMillis(ed); ok && ms < time.Now().UnixMilli() {
				return nil, errors.New("End date must be greater than current timestamp")
			}
		}
		merged := q.defaultJobOptions().Assign(opts)
		return q.updateRepeatableJob(ctx, name, data, merged, true)
	}
	var jobID any
	if opts != nil {
		jobID, _ = opts.Get("jobId")
	}
	if s, ok := jobID.(string); ok && (s == "0" || strings.HasPrefix(s, "0:")) {
		return nil, errors.New("JobId cannot be '0' or start with 0:")
	}
	merged := q.defaultJobOptions().Assign(opts).Set("jobId", jobID)
	return q.createJob(ctx, name, data, merged)
}

// BulkJob is one job of AddBulk.
type BulkJob struct {
	Name string
	Data any
	Opts *Obj
}

// AddBulk mirrors queue.addBulk(jobs): the jobs are added in one pipeline.
func (q *Queue) AddBulk(ctx context.Context, jobs []BulkJob) ([]*Job, error) {
	if len(jobs) == 0 {
		return nil, nil
	}
	instances := make([]*Job, len(jobs))
	pipe := q.client.Pipeline()
	cmds := make([]*redis.Cmd, len(jobs))
	for i, j := range jobs {
		var jobID any
		if j.Opts != nil {
			jobID, _ = j.Opts.Get("jobId")
		}
		opts := q.defaultJobOptions().Assign(j.Opts).Set("jobId", jobID)
		job := newJob(q, j.Name, j.Data, opts, jobID)
		instances[i] = job
		args, err := job.addJobArgs()
		if err != nil {
			return nil, err
		}
		cmds[i] = queueCommand(ctx, pipe, args.command, args.args...)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	for i, c := range cmds {
		v, err := c.Result()
		if err != nil {
			return nil, err
		}
		if n, ok := v.(int64); ok && n < 0 {
			return nil, finishedError(n, "", "", "addJob", "")
		}
		instances[i].ID = argString(v)
	}
	return instances, nil
}

func (q *Queue) defaultJobOptions() *Obj {
	return q.opts.DefaultJobOptions.Clone()
}

// createJob mirrors Job.create.
func (q *Queue) createJob(ctx context.Context, name string, data any, opts *Obj) (*Job, error) {
	jobID, _ := opts.Get("jobId")
	job := newJob(q, name, data, opts, jobID)
	args, err := job.addJobArgs()
	if err != nil {
		return nil, err
	}
	v, err := runCommand(ctx, q.client, args.command, args.args...).Result()
	if err != nil {
		return nil, err
	}
	if n, ok := v.(int64); ok && n < 0 {
		return nil, finishedError(n, "", "", "addJob", "")
	}
	job.ID = argString(v)
	return job, nil
}

// GetJob mirrors queue.getJob(jobId): nil when the job does not exist.
func (q *Queue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	if jobID == "" {
		return nil, nil
	}
	data, err := q.client.HGetAll(ctx, q.ToKey(jobID)).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	return jobFromJSON(q, data, jobID), nil
}

// Remove mirrors queue.remove(jobId, { removeChildren: true }): 1 when the
// job was removed, 0 when it is locked.
func (q *Queue) Remove(ctx context.Context, jobID string) (int64, error) {
	v, err := runCommand(ctx, q.client, "removeJob", q.ToKey(""), q.ToKey("meta"), jobID, 1).Result()
	if err != nil {
		return 0, err
	}
	n, _ := replyInt(v)
	return n, nil
}

// GetCounts mirrors queue.getJobCounts(...types).
func (q *Queue) GetCounts(ctx context.Context, types ...string) (map[string]int64, error) {
	args := []any{q.ToKey("")}
	for _, t := range types {
		if t == "waiting" {
			t = "wait"
		}
		args = append(args, t)
	}
	v, err := runCommand(ctx, q.client, "getCounts", args...).Result()
	if err != nil {
		return nil, err
	}
	out := map[string]int64{}
	arr, _ := v.([]any)
	for i, t := range types {
		if i < len(arr) {
			out[t], _ = replyInt(arr[i])
		}
	}
	return out, nil
}

// Obliterate removes every key of the queue (tests).
func (q *Queue) Obliterate(ctx context.Context) error {
	iter := q.client.Scan(ctx, 0, q.QualifiedName()+":*", 1000).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return q.client.Del(ctx, keys...).Err()
}

// finishedError mirrors scripts.finishedErrors.
func finishedError(code int64, jobID, parentKey, command, state string) error {
	switch code {
	case errJobNotExist:
		return fmt.Errorf("Missing key for job %s. %s", jobID, command)
	case errJobLockNotExist:
		return fmt.Errorf("Missing lock for job %s. %s", jobID, command)
	case errJobNotInState:
		return fmt.Errorf("Job %s is not in the %s state. %s", jobID, state, command)
	case errJobPendingDependencies:
		return fmt.Errorf("Job %s has pending dependencies. %s", jobID, command)
	case errParentJobNotExist:
		return fmt.Errorf("Missing key for parent job %s. %s", parentKey, command)
	case errJobLockMismatch:
		return fmt.Errorf("Lock mismatch for job %s. Cmd %s from %s", jobID, command, state)
	case errParentJobCannotBeReplaced:
		return fmt.Errorf("The parent job %s cannot be replaced. %s", parentKey, command)
	}
	return fmt.Errorf("Unknown code %d error for %s. %s", code, jobID, command)
}

func mustGet(o *Obj, k string) any {
	v, _ := o.Get(k)
	return v
}

// dateMillis mirrors +new Date(v) for numbers, ISO strings and times.
func dateMillis(v any) (int64, bool) {
	switch x := v.(type) {
	case time.Time:
		return x.UnixMilli(), true
	case string:
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z07:00", "2006-01-02"} {
			if t, err := time.Parse(layout, x); err == nil {
				return t.UnixMilli(), true
			}
		}
		return 0, false
	}
	f := toNumber(v)
	if f != f {
		return 0, false
	}
	return int64(f), true
}
