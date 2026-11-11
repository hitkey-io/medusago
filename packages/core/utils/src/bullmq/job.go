package bullmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// PriorityLimit mirrors PRIORITY_LIMIT (2 ** 21).
const PriorityLimit = 1 << 21

var optsDecodeMap = map[string]string{
	"de":   "debounce",
	"fpof": "failParentOnFailure",
	"idof": "ignoreDependencyOnFailure",
	"kl":   "keepLogs",
	"rdof": "removeDependencyOnFailure",
}

var optsEncodeMap = func() map[string]string {
	out := map[string]string{}
	for k, v := range optsDecodeMap {
		out[v] = k
	}
	return out
}()

// Job mirrors bullmq's Job.
type Job struct {
	queue *Queue
	ID    string
	Name  string
	// Data is the job payload: the value given to Add, or the JSON.parse'd
	// payload of a job read from Redis (objects as *Obj).
	Data any
	// RawData is the stored JSON of the payload (read jobs).
	RawData         []byte
	Opts            *Obj
	Timestamp       int64
	Delay           int64
	AttemptsMade    int
	AttemptsStarted int
	ProcessedOn     int64
	FinishedOn      int64
	FailedReason    string
	Stacktrace      []string
	RepeatJobKey    string
	// Token is the lock token of a job fetched by a worker.
	Token string
}

// newJob mirrors the Job constructor.
func newJob(q *Queue, name string, data any, opts *Obj, id any) *Job {
	rest := opts.Clone()
	repeatJobKey, _ := rest.Get("repeatJobKey")
	rest.Delete("repeatJobKey")
	o := NewObj("attempts", 0, "delay", 0).Assign(rest)
	j := &Job{queue: q, Name: name, Data: data, Opts: o}
	if s, ok := id.(string); ok {
		j.ID = s
	}
	if s, ok := repeatJobKey.(string); ok {
		j.RepeatJobKey = s
	}
	j.Delay = int64(numOr(o, "delay", 0))
	if ts, ok := opts.Get("timestamp"); ok && truthy(ts) {
		j.Timestamp = int64(toNumber(ts))
	} else {
		j.Timestamp = time.Now().UnixMilli()
	}
	backoff, _ := opts.Get("backoff")
	o.Set("backoff", normalizeBackoff(backoff))
	return j
}

// Queue returns the queue of the job.
func (j *Job) Queue() *Queue { return j.queue }

// optsAsJSON mirrors optsAsJSON (short keys).
func optsAsJSON(opts *Obj) *Obj {
	out := NewObj()
	for _, k := range opts.Keys() {
		v, _ := opts.Get(k)
		if short, ok := optsEncodeMap[k]; ok {
			out.Set(short, v)
		} else {
			out.Set(k, v)
		}
	}
	return out
}

func optsFromJSON(raw string) *Obj {
	if raw == "" {
		raw = "{}"
	}
	v, err := parseJSON([]byte(raw))
	o, _ := v.(*Obj)
	if err != nil || o == nil {
		return NewObj()
	}
	out := NewObj()
	for _, k := range o.Keys() {
		val, _ := o.Get(k)
		if long, ok := optsDecodeMap[k]; ok {
			out.Set(long, val)
		} else {
			out.Set(k, val)
		}
	}
	return out
}

type commandArgs struct {
	command string
	args    []any
}

// validateOptions mirrors validateOptions.
func (j *Job) validateOptions(data []byte) error {
	if limit := numOr(j.Opts, "sizeLimit", 0); limit > 0 && float64(len(data)) > limit {
		return fmt.Errorf("The size of job %s exceeds the limit %s bytes", j.Name, formatNumber(limit))
	}
	repeat, _ := j.Opts.Get("repeat")
	if truthy(mustGet(j.Opts, "delay")) && repeat != nil && !truthy(mustGet(ObjFromMap(repeat), "count")) {
		return errors.New("Delay and repeat options could not be used together")
	}
	if n, err := strconv.ParseInt(j.ID, 10, 64); err == nil && strconv.FormatInt(n, 10) == j.ID {
		return errors.New("Custom Ids cannot be integers")
	}
	if p := numOr(j.Opts, "priority", 0); p != 0 {
		if math.Trunc(p) != p {
			return errors.New("Priority should not be float")
		}
		if p > PriorityLimit {
			return fmt.Errorf("Priority should be between 0 and %d", PriorityLimit)
		}
	}
	return nil
}

// addJobArgs mirrors Job.addJob / scripts.addJob: the command and its
// arguments.
func (j *Job) addJobArgs() (commandArgs, error) {
	q := j.queue
	data := j.Data
	if data == nil {
		data = NewObj()
	}
	dataJSON, err := jsonStringify(data)
	if err != nil {
		return commandArgs{}, err
	}
	if err := j.validateOptions(dataJSON); err != nil {
		return commandArgs{}, err
	}
	encoded := optsAsJSON(j.Opts)
	if repeat, ok := encoded.Get("repeat"); ok && repeat != nil {
		r := ObjFromMap(repeat).Clone()
		for _, k := range []string{"startDate", "endDate"} {
			if v, ok := r.Get(k); ok && truthy(v) {
				if ms, ok := dateMillis(v); ok {
					r.Set(k, ms)
				}
			}
		}
		encoded = encoded.Clone().Set("repeat", r)
	}
	var jobID any = ""
	if j.ID != "" {
		jobID = j.ID
	}
	var repeatJobKey any
	if j.RepeatJobKey != "" {
		repeatJobKey = j.RepeatJobKey
	}
	packedArgs := pack([]any{q.ToKey(""), jobID, j.Name, j.Timestamp, nil, nil, nil, nil, repeatJobKey, nil})
	packedOpts := pack(encoded)
	switch {
	case truthy(mustGet(j.Opts, "delay")):
		return commandArgs{"addDelayedJob", []any{q.keys["marker"], q.keys["meta"], q.keys["id"], q.keys["delayed"], q.keys["completed"], q.keys["events"],
			packedArgs, string(dataJSON), packedOpts}}, nil
	case truthy(mustGet(j.Opts, "priority")):
		return commandArgs{"addPrioritizedJob", []any{q.keys["marker"], q.keys["meta"], q.keys["id"], q.keys["prioritized"], q.keys["completed"], q.keys["active"],
			q.keys["events"], q.keys["pc"], packedArgs, string(dataJSON), packedOpts}}, nil
	}
	return commandArgs{"addStandardJob", []any{q.keys["wait"], q.keys["paused"], q.keys["meta"], q.keys["id"], q.keys["completed"], q.keys["active"],
		q.keys["events"], q.keys["marker"], packedArgs, string(dataJSON), packedOpts}}, nil
}

// jobFromJSON mirrors Job.fromJSON over the job hash.
func jobFromJSON(q *Queue, raw map[string]string, jobID string) *Job {
	dataRaw := raw["data"]
	if dataRaw == "" {
		dataRaw = "{}"
	}
	data, _ := parseJSON([]byte(dataRaw))
	opts := optsFromJSON(raw["opts"])
	id := raw["id"]
	if id == "" {
		id = jobID
	}
	j := newJob(q, raw["name"], data, opts, id)
	j.RawData = []byte(dataRaw)
	j.Delay, _ = strconv.ParseInt(raw["delay"], 10, 64)
	j.Timestamp, _ = strconv.ParseInt(raw["timestamp"], 10, 64)
	j.FinishedOn, _ = strconv.ParseInt(raw["finishedOn"], 10, 64)
	j.ProcessedOn, _ = strconv.ParseInt(raw["processedOn"], 10, 64)
	if raw["rjk"] != "" {
		j.RepeatJobKey = raw["rjk"]
	}
	j.FailedReason = raw["failedReason"]
	j.AttemptsStarted, _ = strconv.Atoi(raw["ats"])
	atm := raw["attemptsMade"]
	if atm == "" {
		atm = raw["atm"]
	}
	j.AttemptsMade, _ = strconv.Atoi(atm)
	if raw["stacktrace"] != "" {
		_ = json.Unmarshal([]byte(raw["stacktrace"]), &j.Stacktrace)
	}
	return j
}

// jobFromArray mirrors array2obj + fromJSON for a moveToActive reply.
func jobFromArray(q *Queue, arr []any, jobID string) *Job {
	raw := map[string]string{}
	for i := 0; i+1 < len(arr); i += 2 {
		raw[argString(arr[i])] = argString(arr[i+1])
	}
	return jobFromJSON(q, raw, jobID)
}

// DecodeData decodes the raw payload into v.
func (j *Job) DecodeData(v any) error {
	if j.RawData != nil {
		return json.Unmarshal(j.RawData, v)
	}
	b, err := jsonStringify(j.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// UpdateData mirrors job.updateData(data).
func (j *Job) UpdateData(ctx context.Context, data any) error {
	b, err := jsonStringify(data)
	if err != nil {
		return err
	}
	v, err := runCommand(ctx, j.queue.client, "updateData", j.queue.ToKey(j.ID), string(b)).Result()
	if err != nil {
		return err
	}
	if n, ok := v.(int64); ok && n < 0 {
		return finishedError(n, j.ID, "", "updateData", "")
	}
	j.Data = data
	j.RawData = b
	return nil
}

// Remove mirrors job.remove().
func (j *Job) Remove(ctx context.Context) error {
	n, err := j.queue.Remove(ctx, j.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("Job %s could not be removed because it is locked by another worker", j.ID)
	}
	return nil
}

func normalizeBackoff(backoff any) any {
	if backoff == nil {
		return nil
	}
	if f := toNumber(backoff); !math.IsNaN(f) && !math.IsInf(f, 0) {
		if _, isStr := backoff.(string); !isStr {
			return NewObj("type", "fixed", "delay", f)
		}
	}
	if !truthy(backoff) {
		return nil
	}
	return ObjFromMap(backoff)
}

// calculateBackoff mirrors Backoffs.calculate (builtin strategies);
// ok false when there is no backoff.
func calculateBackoff(backoff any, attemptsMade int) (int64, bool, error) {
	b := ObjFromMap(backoff)
	if b == nil {
		return 0, false, nil
	}
	delay := numOr(b, "delay", 0)
	switch mustGet(b, "type") {
	case "fixed":
		return int64(delay), true, nil
	case "exponential":
		return int64(math.Round(math.Pow(2, float64(attemptsMade-1)) * delay)), true, nil
	}
	return 0, false, fmt.Errorf("Unknown backoff strategy %v.\n      If a custom backoff strategy is used, specify it when the queue is created.", mustGet(b, "type"))
}

// getKeepJobs mirrors scripts.getKeepJobs.
func getKeepJobs(shouldRemove, workerKeepJobs any) any {
	if shouldRemove == nil {
		if workerKeepJobs != nil && truthy(workerKeepJobs) {
			return workerKeepJobs
		}
		return NewObj("count", -1)
	}
	switch x := shouldRemove.(type) {
	case bool:
		if x {
			return NewObj("count", 0)
		}
		return NewObj("count", -1)
	case *Obj, map[string]any:
		return ObjFromMap(x)
	}
	return NewObj("count", toNumber(shouldRemove))
}

// moveToFinishedArgs mirrors scripts.moveToFinishedArgs.
func (w *Worker) moveToFinishedArgs(j *Job, val string, propVal string, shouldRemove any, target string, token string, timestamp int64, fetchNext bool) []any {
	q := j.queue
	workerKeep := w.opts.RemoveOnComplete
	if target == "failed" {
		workerKeep = w.opts.RemoveOnFail
	}
	fetch := 0
	if fetchNext && !w.closing.Load() {
		fetch = 1
	}
	attempts, _ := j.Opts.Get("attempts")
	return []any{q.keys["wait"], q.keys["active"], q.keys["prioritized"], q.keys["events"], q.keys["stalled"], q.keys["limiter"], q.keys["delayed"],
		q.keys["paused"], q.keys["meta"], q.keys["pc"], q.keys[target], q.ToKey(j.ID), q.ToKey("metrics:" + target), q.keys["marker"],
		j.ID, timestamp, propVal, val, target, fetch, q.ToKey(""),
		pack(NewObj("token", token, "keepJobs", getKeepJobs(shouldRemove, workerKeep), "limiter", nil, "lockDuration", w.opts.LockDuration,
			"attempts", attempts, "maxMetricsSize", "", "fpof", truthy(mustGet(j.Opts, "failParentOnFailure")),
			"idof", truthy(mustGet(j.Opts, "ignoreDependencyOnFailure")), "rdof", truthy(mustGet(j.Opts, "removeDependencyOnFailure")))),
	}
}

// moveToCompleted mirrors job.moveToCompleted(returnValue, token, false).
func (w *Worker) moveToCompleted(ctx context.Context, j *Job, returnValue any) error {
	val := "null"
	if returnValue != nil {
		b, err := jsonStringify(returnValue)
		if err != nil {
			return err
		}
		val = string(b)
	}
	removeOnComplete, _ := j.Opts.Get("removeOnComplete")
	args := w.moveToFinishedArgs(j, val, "returnvalue", removeOnComplete, "completed", j.Token, time.Now().UnixMilli(), false)
	v, err := runCommand(ctx, j.queue.client, "moveToFinished", args...).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	if n, ok := v.(int64); ok && n < 0 {
		return finishedError(n, j.ID, "", "moveToFinished", "active")
	}
	j.AttemptsMade++
	return nil
}

// moveToFailed mirrors job.moveToFailed(err, token, false): an automatic
// retry (immediate or delayed by the backoff) while attempts remain, else
// the failed set.
func (w *Worker) moveToFailed(ctx context.Context, j *Job, jobErr error) error {
	q := j.queue
	message := jobErr.Error()
	j.FailedReason = message
	pipe := q.client.TxPipeline()
	j.Stacktrace = append(j.Stacktrace, "Error: "+message)
	if limit, ok := j.Opts.Get("stackTraceLimit"); ok && limit != nil {
		n := int(toNumber(limit))
		if n == 0 {
			j.Stacktrace = []string{}
		} else if len(j.Stacktrace) > n {
			j.Stacktrace = j.Stacktrace[len(j.Stacktrace)-n:]
		}
	}
	trace, _ := jsonStringify(j.Stacktrace)
	queueCommand(ctx, pipe, "saveStacktrace", q.ToKey(j.ID), string(trace), message)
	moveToFailed := false
	command := ""
	var last *redis.Cmd
	var unrecoverable *UnrecoverableError
	if float64(j.AttemptsMade+1) < numOr(j.Opts, "attempts", 0) && !errors.As(jobErr, &unrecoverable) {
		backoff, _ := j.Opts.Get("backoff")
		delay, hasDelay, err := calculateBackoff(backoff, j.AttemptsMade+1)
		if err != nil {
			return err
		}
		switch {
		case hasDelay && delay == -1:
			moveToFailed = true
		case hasDelay && delay != 0:
			last = queueCommand(ctx, pipe, "moveToDelayed", q.keys["marker"], q.keys["active"], q.keys["prioritized"], q.keys["delayed"], q.ToKey(j.ID),
				q.keys["events"], q.keys["meta"], q.keys["stalled"], q.ToKey(""), time.Now().UnixMilli(), j.ID, j.Token, delay, "0")
			command = "moveToDelayed"
		default:
			last = queueCommand(ctx, pipe, "retryJob", q.keys["active"], q.keys["wait"], q.keys["paused"], q.ToKey(j.ID), q.keys["meta"], q.keys["events"],
				q.keys["delayed"], q.keys["prioritized"], q.keys["pc"], q.keys["marker"], q.keys["stalled"],
				q.ToKey(""), time.Now().UnixMilli(), pushCmd(j.Opts), j.ID, j.Token)
			command = "retryJob"
		}
	} else {
		moveToFailed = true
	}
	if moveToFailed {
		removeOnFail, _ := j.Opts.Get("removeOnFail")
		args := w.moveToFinishedArgs(j, message, "failedReason", removeOnFail, "failed", j.Token, time.Now().UnixMilli(), false)
		last = queueCommand(ctx, pipe, "moveToFinished", args...)
		command = "moveToFinished"
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("Error \"moveToFailed\" with command %s: %v", command, err)
	}
	if v, _ := last.Result(); v != nil {
		if n, ok := v.(int64); ok && n < 0 {
			return finishedError(n, j.ID, "", command, "active")
		}
	}
	j.AttemptsMade++
	return nil
}

func pushCmd(opts *Obj) string {
	if truthy(mustGet(opts, "lifo")) {
		return "RPUSH"
	}
	return "LPUSH"
}

// UnrecoverableError mirrors UnrecoverableError: the job fails without
// further attempts.
type UnrecoverableError struct{ Message string }

func (e *UnrecoverableError) Error() string { return e.Message }
