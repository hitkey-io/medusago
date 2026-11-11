package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// A port of the posthog-node client parts the provider uses (capture,
// identify, groupIdentify, shutdown): events are queued and sent in
// gzipped batches to <host>/batch/ when 20 are queued, every 10 seconds and
// on shutdown.

const (
	posthogLibVersion    = "5.57.0"
	posthogFlushAt       = 20
	posthogFlushInterval = 10 * time.Second
	posthogMaxQueueSize  = 1000
	posthogMaxBatchSize  = 100
	posthogRetries       = 3
	posthogRetryDelay    = 3 * time.Second
)

type posthogClient struct {
	apiKey     string
	host       string
	httpClient *http.Client
	logger     *slog.Logger
	now        func() time.Time

	mu      sync.Mutex
	queue   []*dal.Record
	timer   *time.Timer
	flushMu sync.Mutex
}

func newPosthogClient(apiKey, host string, logger *slog.Logger) *posthogClient {
	return &posthogClient{apiKey: apiKey, host: strings.TrimRight(host, "/"), httpClient: http.DefaultClient, logger: logger, now: time.Now}
}

// uuidv7 mirrors the core uuidv7 (48-bit millisecond timestamp, version 7,
// variant 10).
func uuidv7(t time.Time) string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	ms := uint64(t.UnixMilli())
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], ms)
	copy(b[0:6], ts[2:8])
	b[6] = 0x70 | (b[6] & 0x0f)
	b[8] = 0x80 | (b[8] & 0x3f)
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

func isoTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// commonProperties mirrors getCommonEventProperties of posthog-node.
func commonProperties(props *dal.Record) {
	props.Set("$lib", "posthog-node")
	props.Set("$lib_version", posthogLibVersion)
	props.Set("$is_server", true)
}

// enqueue mirrors prepareMessage + enqueue: { event, properties, timestamp,
// uuid, distinct_id } with $geoip_disable (disableGeoip defaults to true).
func (c *posthogClient) enqueue(distinctID, event string, properties *dal.Record) {
	now := c.now()
	properties.Set("$geoip_disable", true)
	msg := dal.NewRecord()
	msg.Set("event", event)
	msg.Set("properties", properties)
	msg.Set("timestamp", isoTime(now))
	msg.Set("uuid", uuidv7(now))
	msg.Set("distinct_id", distinctID)

	c.mu.Lock()
	if len(c.queue) >= posthogMaxQueueSize {
		c.queue = c.queue[1:]
		c.logger.Warn("Queue is full, the oldest event is dropped.")
	}
	c.queue = append(c.queue, msg)
	full := len(c.queue) >= posthogFlushAt
	if c.timer == nil {
		c.timer = time.AfterFunc(posthogFlushInterval, func() { c.flushBackground() })
	}
	c.mu.Unlock()
	if full {
		go c.flushBackground()
	}
}

// mergedProperties mirrors { ...properties || {} } (a copy).
func mergedProperties(properties any) *dal.Record {
	out := dal.NewRecord()
	if r, ok := properties.(*dal.Record); ok && r != nil {
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

// capture mirrors capture({ event, distinctId, properties, groups }).
func (c *posthogClient) capture(event, distinctID string, properties any, groups *dal.Record) {
	props := mergedProperties(properties)
	if distinctID == "" {
		distinctID = uuidv7(c.now())
		props.Set("$process_person_profile", false)
	}
	if groups != nil && len(groups.Keys()) > 0 {
		props.Set("$groups", groups)
	}
	commonProperties(props)
	c.enqueue(distinctID, event, props)
}

// identify mirrors identify({ distinctId, properties }).
func (c *posthogClient) identify(distinctID string, properties any) {
	in := mergedProperties(properties)
	setProps, hasSet := in.Get("$set")
	setOnce, hasSetOnce := in.Get("$set_once")
	anon, hasAnon := in.Get("$anon_distinct_id")
	rest := dal.NewRecord()
	for _, k := range in.Keys() {
		if k == "$set" || k == "$set_once" || k == "$anon_distinct_id" {
			continue
		}
		v, _ := in.Get(k)
		rest.Set(k, v)
	}
	if !hasSet || !truthy(setProps) {
		setProps = rest
	}
	if !hasSetOnce || !truthy(setOnce) {
		setOnce = dal.NewRecord()
	}
	props := dal.NewRecord()
	props.Set("$set", setProps)
	props.Set("$set_once", setOnce)
	if hasAnon && anon != nil {
		props.Set("$anon_distinct_id", anon)
	}
	if distinctID == "" {
		distinctID = uuidv7(c.now())
		props.Set("$process_person_profile", false)
	}
	commonProperties(props)
	c.enqueue(distinctID, "$identify", props)
}

// groupIdentify mirrors groupIdentify({ groupType, groupKey, properties,
// distinctId }).
func (c *posthogClient) groupIdentify(groupType, groupKey string, properties any, distinctID string) {
	if distinctID == "" {
		distinctID = fmt.Sprintf("$%s_%s", groupType, groupKey)
	}
	props := dal.NewRecord()
	props.Set("$group_type", groupType)
	props.Set("$group_key", groupKey)
	if properties == nil || !truthy(properties) {
		properties = dal.NewRecord()
	}
	props.Set("$group_set", properties)
	commonProperties(props)
	c.enqueue(distinctID, "$groupidentify", props)
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

func (c *posthogClient) flushBackground() {
	if err := c.flush(context.Background()); err != nil {
		c.logger.Error("Error while flushing PostHog: " + err.Error())
	}
}

// flush sends the queued events in batches.
func (c *posthogClient) flush(ctx context.Context) error {
	c.flushMu.Lock()
	defer c.flushMu.Unlock()
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
	for {
		c.mu.Lock()
		n := min(len(c.queue), posthogMaxBatchSize)
		batch := append([]*dal.Record(nil), c.queue[:n]...)
		c.mu.Unlock()
		if n == 0 {
			return nil
		}
		err := c.sendBatch(ctx, batch)
		c.mu.Lock()
		c.queue = c.queue[n:]
		c.mu.Unlock()
		if err != nil {
			return err
		}
	}
}

func (c *posthogClient) sendBatch(ctx context.Context, batch []*dal.Record) error {
	data := dal.NewRecord()
	data.Set("api_key", c.apiKey)
	items := make([]any, len(batch))
	for i, m := range batch {
		items[i] = m
	}
	data.Set("batch", items)
	data.Set("sent_at", isoTime(c.now()))
	payload, err := dal.MarshalValue(data)
	if err != nil {
		return err
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	_, _ = w.Write(payload)
	_ = w.Close()

	var lastErr error
	for attempt := 0; attempt <= posthogRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(posthogRetryDelay):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/batch/", bytes.NewReader(gz.Bytes()))
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "posthog-node/"+posthogLibVersion)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		res, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_, _ = io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode >= 200 && res.StatusCode < 400 {
			return nil
		}
		lastErr = fmt.Errorf("HTTP error while fetching PostHog: status=%d", res.StatusCode)
		if res.StatusCode < 500 && res.StatusCode != 408 && res.StatusCode != 429 {
			return lastErr
		}
	}
	return lastErr
}

// shutdown mirrors shutdown: the queue is flushed.
func (c *posthogClient) shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.flush(ctx)
}
