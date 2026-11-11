package httpdiff

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/framework/src/jobs"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// Scenario environments for the Redis modules, scheduled jobs and
// subscribers. The reference application (medusa-config.js, src/jobs,
// src/subscribers of the recording copy) reads the same variables:
//
//   HTTP_DIFF_REDIS=true        event-bus-redis (jobOptions { attempts: 2 })
//                               and workflow-engine-redis on
//                               MEDUSA_TEST_REDIS_URL (taken from the
//                               environment of the test run; the scenario
//                               is skipped without it)
//   HTTP_DIFF_CACHING=true      the caching module with the caching-redis
//                               provider on MEDUSA_TEST_REDIS_URL (with
//                               MEDUSA_FF_CACHING=true); "memory": with its
//                               built in memory provider
//   HTTP_DIFF_SUBSCRIBERS=true  a product.created subscriber recording each
//                               call (numbered per process) as a product
//                               tag; with
//                               HTTP_DIFF_SUBSCRIBER_FAIL_ONCE=true its first
//                               call fails (retried by the event bus)
//   HTTP_DIFF_JOBS=true         the scheduled job "http-diff-scheduled"
//                               (interval 1000 ms, one execution) creating a
//                               product tag

func redisScenario() bool {
	return os.Getenv("HTTP_DIFF_REDIS") == "true" || os.Getenv("HTTP_DIFF_CACHING") == "true"
}

func redisScenarioModules() []config.ModuleConfig {
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	var out []config.ModuleConfig
	if os.Getenv("HTTP_DIFF_CACHING") == "memory" {
		// the caching module with its built in memory provider
		out = append(out, config.ModuleConfig{Key: "caching", Resolve: "@medusajs/medusa/caching", Options: map[string]any{
			"in_memory": map[string]any{"enable": true},
		}})
	}
	if os.Getenv("HTTP_DIFF_CACHING") == "true" {
		// the caching module with the Redis provider
		out = append(out, config.ModuleConfig{Key: "caching", Resolve: "@medusajs/medusa/caching", Providers: []config.ModuleProviderConfig{
			{Resolve: "@medusajs/medusa/caching-redis", ID: "caching-redis", IsDefault: true, Options: map[string]any{"redisUrl": url}},
		}})
	}
	if os.Getenv("HTTP_DIFF_REDIS") != "true" {
		return out
	}
	return append(out, []config.ModuleConfig{
		{Resolve: "@medusajs/medusa/event-bus-redis", Options: map[string]any{
			"redisUrl": url, "jobOptions": map[string]any{"attempts": 2},
		}},
		{Resolve: "@medusajs/medusa/workflow-engine-redis", Options: map[string]any{
			"redis": map[string]any{"redisUrl": url},
		}},
	}...)
}

// prepareRedis skips a Redis scenario without MEDUSA_TEST_REDIS_URL and
// empties the database otherwise (record.sh starts upstream on a fresh one).
func prepareRedis(t *testing.T) {
	t.Helper()
	if !redisScenario() {
		return
	}
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL not set")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	c := redis.NewClient(opt)
	defer c.Close()
	if err := c.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
}

type tagCreator interface {
	CreateProductTags(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

func createTag(ctx context.Context, c composer.Container, value string, metadata *dal.Record) error {
	svc, ok := c.Resolve(utilsmodulessdk.PRODUCT).(tagCreator)
	if !ok {
		return errors.New("product module not found")
	}
	tag := dal.NewRecord()
	tag.Set("value", value)
	tag.Set("metadata", metadata)
	_, err := svc.CreateProductTags(ctx, []*dal.Record{tag})
	return err
}

func sortedKeys(v any) string {
	r, ok := v.(*dal.Record)
	if !ok || r == nil {
		return ""
	}
	keys := r.Keys()
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

// scenarioHooks registers the scenario's subscribers and scheduled jobs
// (mirroring src/subscribers/http-diff-events.js and
// src/jobs/http-diff-scheduled.js of the reference application).
func scenarioHooks(t *testing.T) []func(context.Context, *loaders.MedusaApp) error {
	var hooks []func(context.Context, *loaders.MedusaApp) error
	if os.Getenv("HTTP_DIFF_SUBSCRIBERS") == "true" {
		hooks = append(hooks, func(_ context.Context, app *loaders.MedusaApp) error {
			bus, ok := app.Container.Resolve(utilsmodulessdk.EVENT_BUS).(eventbus.Subscribable)
			if !ok {
				return errors.New("event bus not found")
			}
			var mu sync.Mutex
			calls := 0
			bus.Subscribe("product.created", func(ctx context.Context, ev eventbus.Event) error {
				data, _ := ev.Data.(*dal.Record)
				var id any
				if data != nil {
					id, _ = data.Get("id")
				}
				mu.Lock()
				calls++
				n := calls
				mu.Unlock()
				md := dal.NewRecord()
				md.Set("product_id", id)
				md.Set("data_keys", sortedKeys(ev.Data))
				md.Set("metadata_keys", sortedKeys(ev.Metadata))
				if err := createTag(ctx, app.Container, fmt.Sprintf("event-%s-call-%d", ev.Name, n), md); err != nil {
					return err
				}
				if n == 1 && os.Getenv("HTTP_DIFF_SUBSCRIBER_FAIL_ONCE") == "true" {
					return errors.New("http-diff subscriber fails once")
				}
				return nil
			})
			return nil
		})
	}
	if os.Getenv("HTTP_DIFF_JOBS") == "true" {
		hooks = append(hooks, func(context.Context, *loaders.MedusaApp) error {
			one := 1
			_, err := jobs.Register(&jobs.CronJobConfig{
				Name:            "http-diff-scheduled",
				ScheduleOptions: &composer.SchedulerOptions{Interval: 1000, NumberOfExecutions: &one},
			}, func(ctx context.Context, c composer.Container, jc jobs.ScheduledJobContext) (any, error) {
				md := dal.NewRecord()
				md.Set("has_scheduled_for", !jc.ScheduledFor.IsZero())
				return nil, createTag(ctx, c, "http-diff-scheduled-job", md)
			})
			t.Cleanup(func() { jobs.Unregister("http-diff-scheduled") })
			return err
		})
	}
	return hooks
}
