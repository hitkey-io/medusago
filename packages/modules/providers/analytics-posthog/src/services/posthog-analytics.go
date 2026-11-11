// Package services mirrors @medusajs/analytics-posthog/src/services.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	analyticstypes "github.com/hitkey-io/medusago/packages/core/types/src/analytics"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Identifier is PosthogAnalyticsService.identifier.
const Identifier = "analytics-posthog"

var (
	_ analyticstypes.IAnalyticsProvider = (*PosthogAnalyticsService)(nil)
	_ analyticstypes.ShutdownProvider   = (*PosthogAnalyticsService)(nil)
)

// PosthogAnalyticsService mirrors PosthogAnalyticsService.
type PosthogAnalyticsService struct {
	options map[string]any
	logger  *slog.Logger
	client  *posthogClient
}

// NewPosthogAnalyticsService mirrors the constructor (options
// posthogEventsKey, posthogHost).
func NewPosthogAnalyticsService(options map[string]any, logger *slog.Logger) (*PosthogAnalyticsService, error) {
	if logger == nil {
		logger = slog.Default()
	}
	key, _ := options["posthogEventsKey"].(string)
	if key == "" {
		return nil, errors.New("Posthog API key is not set, but is required")
	}
	host, _ := options["posthogHost"].(string)
	if host == "" {
		host = "https://eu.i.posthog.com"
	}
	return &PosthogAnalyticsService{options: options, logger: logger, client: newPosthogClient(key, host, logger)}, nil
}

func get(r *dal.Record, path ...string) (any, bool) {
	var cur any = r
	for _, k := range path {
		rec, ok := cur.(*dal.Record)
		if !ok || rec == nil {
			return nil, false
		}
		if cur, ok = rec.Get(k); !ok {
			return nil, false
		}
	}
	return cur, true
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if f, ok := v.(float64); ok && f == float64(int64(f)) {
		return fmt.Sprint(int64(f))
	}
	return fmt.Sprint(v)
}

// Track mirrors track.
func (s *PosthogAnalyticsService) Track(ctx context.Context, data *dal.Record) error {
	event, _ := get(data, "event")
	if !truthy(event) {
		return errors.New("Event name is required when tracking an event with Posthog")
	}
	actor, _ := get(data, "actor_id")
	if !truthy(actor) {
		return errors.New("Actor ID is required when tracking an event with Posthog")
	}
	groupID, _ := get(data, "group", "id")
	groupType, _ := get(data, "group", "type")
	if truthy(groupID) && !truthy(groupType) {
		return errors.New("Group type is required if passing group id when tracking an event with Posthog")
	}
	var groups *dal.Record
	if truthy(groupID) {
		groups = dal.NewRecord()
		groups.Set(str(groupType), groupID)
	}
	props, _ := get(data, "properties")
	s.client.capture(str(event), str(actor), props, groups)
	return nil
}

// Identify mirrors identify.
func (s *PosthogAnalyticsService) Identify(ctx context.Context, data *dal.Record) error {
	props, _ := get(data, "properties")
	actor, _ := get(data, "actor_id")
	if _, hasGroup := get(data, "group"); hasGroup {
		groupID, _ := get(data, "group", "id")
		groupType, _ := get(data, "group", "type")
		s.client.groupIdentify(str(groupType), str(groupID), props, str(actor))
		return nil
	}
	if truthy(actor) {
		s.client.identify(str(actor), props)
		return nil
	}
	return errors.New("Actor or group is required when identifying an entity with Posthog")
}

// Shutdown mirrors shutdown (the queued events are flushed).
func (s *PosthogAnalyticsService) Shutdown(ctx context.Context) error {
	return s.client.shutdown(ctx)
}
