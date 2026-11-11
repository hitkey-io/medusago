package bullmq

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	cronparser "github.com/hitkey-io/medusago/packages/core/utils/src/cron-parser"
)

// md5Hex mirrors createHash("md5").update(s).digest("hex").
func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// getRepeatConcatOptions mirrors getRepeatConcatOptions.
func getRepeatConcatOptions(name string, repeat *Obj) string {
	endDate := ""
	if v, ok := repeat.Get("endDate"); ok && truthy(v) {
		if ms, ok := dateMillis(v); ok {
			endDate = strconv.FormatInt(ms, 10)
		}
	}
	tz := ""
	if v, ok := repeat.Get("tz"); ok && truthy(v) {
		tz = jsString(v)
	}
	suffix := ""
	if p, ok := repeat.Get("pattern"); ok && truthy(p) {
		suffix = jsString(p)
	} else {
		every, _ := repeat.Get("every")
		suffix = jsString(every)
	}
	jobID := ""
	if v, ok := repeat.Get("jobId"); ok && truthy(v) {
		jobID = jsString(v)
	}
	return name + ":" + jobID + ":" + endDate + ":" + tz + ":" + suffix
}

// GetNextMillis mirrors getNextMillis(millis, opts); ok false when there is
// no next iteration.
func GetNextMillis(millis int64, opts *Obj) (int64, bool, error) {
	pattern, _ := opts.Get("pattern")
	every := numOr(opts, "every", 0)
	immediately := truthy(mustGet(opts, "immediately"))
	if truthy(pattern) && every != 0 {
		return 0, false, errors.New("Both .pattern and .every options are defined for this repeatable job")
	}
	if every != 0 {
		next := math.Floor(float64(millis)/every) * every
		if !immediately {
			next += every
		}
		return int64(next), true, nil
	}
	current := time.UnixMilli(millis)
	if sd, ok := opts.Get("startDate"); ok && truthy(sd) {
		if ms, ok := dateMillis(sd); ok && ms > millis {
			current = time.UnixMilli(ms)
		}
	}
	po := cronparser.Options{CurrentDate: current}
	if tz, ok := opts.Get("tz"); ok && truthy(tz) {
		po.TZ = jsString(tz)
	}
	if ed, ok := opts.Get("endDate"); ok && truthy(ed) {
		if ms, ok := dateMillis(ed); ok {
			po.EndDate = time.UnixMilli(ms)
		}
	}
	expr, err := cronparser.Parse(jsString(pattern), po)
	if err != nil {
		return 0, false, err
	}
	if immediately {
		return time.Now().UnixMilli(), true, nil
	}
	next, err := expr.Next()
	if err != nil {
		return 0, false, nil
	}
	return next.UnixMilli(), true, nil
}

// updateRepeatableJob mirrors Repeat.updateRepeatableJob: it stores the
// repeatable job (override) or its next millis, and adds the delayed job of
// the next iteration.
func (q *Queue) updateRepeatableJob(ctx context.Context, name string, data any, opts *Obj, override bool) (*Job, error) {
	repeatOpts := ObjFromMap(mustGet(opts, "repeat")).Clone()
	// repeatOpts.pattern ??= repeatOpts.cron (the key exists afterwards:
	// cmsgpack presizes the Lua table with every packed key, which decides
	// the key order of the stored JSON)
	if !repeatOpts.Has("pattern") {
		cron, _ := repeatOpts.Get("cron")
		repeatOpts.Set("pattern", cron)
	}
	repeatOpts.Delete("cron")
	iterationCount := 1.0
	if c := numOr(repeatOpts, "count", 0); c != 0 {
		iterationCount = c + 1
	}
	if limit, ok := repeatOpts.Get("limit"); ok && limit != nil && iterationCount > toNumber(limit) {
		return nil, nil
	}
	now := time.Now().UnixMilli()
	endDate, _ := repeatOpts.Get("endDate")
	if endDate != nil {
		if ms, ok := dateMillis(endDate); ok && now > ms {
			return nil, nil
		}
	}
	prevMillis := int64(numOr(opts, "prevMillis", 0))
	if prevMillis > now {
		now = prevMillis
	}
	nextMillis, ok, err := GetNextMillis(now, repeatOpts)
	if err != nil {
		return nil, err
	}
	every, _ := repeatOpts.Get("every")
	pattern, _ := repeatOpts.Get("pattern")
	hasImmediately := (truthy(every) || truthy(pattern)) && truthy(mustGet(repeatOpts, "immediately"))
	var offset any
	if hasImmediately && truthy(every) {
		offset = now - nextMillis
	}
	if !ok || nextMillis == 0 {
		return nil, nil
	}
	if prevMillis == 0 {
		if jobID, ok := opts.Get("jobId"); ok && truthy(jobID) {
			repeatOpts.Set("jobId", jobID)
		}
	}
	concat := getRepeatConcatOptions(name, repeatOpts)
	customKey := concat
	origRepeat := ObjFromMap(mustGet(opts, "repeat"))
	if k, ok := origRepeat.Get("key"); ok && k != nil {
		customKey = jsString(k)
	} else {
		customKey = md5Hex(concat)
	}
	var repeatJobKey string
	if override {
		var endMillis any
		if endDate != nil {
			if ms, ok := dateMillis(endDate); ok {
				endMillis = ms
			}
		}
		tz, _ := repeatOpts.Get("tz")
		v, err := runCommand(ctx, q.client, "addRepeatableJob", q.keys["repeat"], q.keys["delayed"],
			nextMillis, pack(NewObj("name", name, "endDate", endMillis, "tz", tz, "pattern", pattern, "every", every)), concat, customKey, q.ToKey("")).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
		if v != nil {
			repeatJobKey = argString(v)
		}
	} else {
		v, err := runCommand(ctx, q.client, "updateRepeatableJobMillis", q.keys["repeat"], nextMillis, customKey, concat).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
		if v != nil {
			repeatJobKey = argString(v)
		}
	}
	if repeatJobKey == "" {
		return nil, nil
	}
	filtered := repeatOpts.Clone()
	filtered.Delete("immediately")
	repeat := NewObj("offset", offset).Assign(filtered)
	return q.createNextJob(ctx, name, nextMillis, repeatJobKey, opts.Clone().Set("repeat", repeat), data, iterationCount, hasImmediately)
}

// createNextJob mirrors Repeat.createNextJob.
func (q *Queue) createNextJob(ctx context.Context, name string, nextMillis int64, repeatJobKey string, opts *Obj, data any, currentCount float64, hasImmediately bool) (*Job, error) {
	jobID := q.getRepeatJobKey(name, nextMillis, repeatJobKey, data)
	now := time.Now().UnixMilli()
	repeat := ObjFromMap(mustGet(opts, "repeat"))
	delay := nextMillis + int64(numOr(repeat, "offset", 0)) - now
	if delay < 0 || hasImmediately {
		delay = 0
	}
	merged := opts.Clone().Set("jobId", jobID).Set("delay", delay).Set("timestamp", now).Set("prevMillis", nextMillis).Set("repeatJobKey", repeatJobKey)
	merged.Set("repeat", repeat.Clone().Set("count", currentCount))
	return q.createJob(ctx, name, data, merged)
}

// getRepeatJobKey mirrors Repeat.getRepeatJobKey.
func (q *Queue) getRepeatJobKey(name string, nextMillis int64, repeatJobKey string, data any) string {
	if len(strings.Split(repeatJobKey, ":")) > 2 {
		var id any
		if o := ObjFromMap(data); o != nil {
			id, _ = o.Get("id")
		}
		return getRepeatJobID(name, strconv.FormatInt(nextMillis, 10), md5Hex(repeatJobKey), id, nil)
	}
	return "repeat:" + repeatJobKey + ":" + strconv.FormatInt(nextMillis, 10)
}

// getRepeatJobID mirrors Repeat.getRepeatJobId.
func getRepeatJobID(name, nextMillis, namespace string, jobID any, key any) string {
	checksum := ""
	if key != nil {
		checksum = jsString(key)
	} else {
		id := ""
		if truthy(jobID) {
			id = jsString(jobID)
		}
		checksum = md5Hex(name + id + namespace)
	}
	return "repeat:" + checksum + ":" + nextMillis
}

// RepeatableJob mirrors an entry of getRepeatableJobs.
type RepeatableJob struct {
	Key     string
	Name    string
	Pattern string
	Every   string
	Next    int64
}

// GetRepeatableJobs mirrors queue.getRepeatableJobs() (descending).
func (q *Queue) GetRepeatableJobs(ctx context.Context) ([]RepeatableJob, error) {
	res, err := q.client.ZRevRangeWithScores(ctx, q.keys["repeat"], 0, -1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]RepeatableJob, 0, len(res))
	for _, z := range res {
		key := argString(z.Member)
		data, err := q.client.HGetAll(ctx, q.ToKey("repeat:"+key)).Result()
		if err != nil {
			return nil, err
		}
		out = append(out, RepeatableJob{Key: key, Name: data["name"], Pattern: data["pattern"], Every: data["every"], Next: int64(z.Score)})
	}
	return out, nil
}

// RemoveRepeatableByKey mirrors queue.removeRepeatableByKey(key): true when
// it was removed.
func (q *Queue) RemoveRepeatableByKey(ctx context.Context, key string) (bool, error) {
	parts := strings.Split(key, ":")
	var id any
	if len(parts) > 1 && parts[1] != "" {
		id = parts[1]
	}
	legacy := getRepeatJobID(parts[0], "", md5Hex(key), id, nil)
	concat := ""
	if len(parts) > 2 {
		concat = key
	}
	v, err := runCommand(ctx, q.client, "removeRepeatable", q.keys["repeat"], q.keys["delayed"], q.keys["events"], legacy, concat, key, q.ToKey("")).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err
	}
	n, _ := replyInt(v)
	return n == 0, nil
}
