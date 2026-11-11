// Package services mirrors @medusajs/analytics-local/src/services.
package services

import (
	"context"
	"fmt"
	"log/slog"

	analyticstypes "github.com/hitkey-io/medusago/packages/core/types/src/analytics"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Identifier is LocalAnalyticsService.identifier.
const Identifier = "analytics-local"

var _ analyticstypes.IAnalyticsProvider = (*LocalAnalyticsService)(nil)

// LocalAnalyticsService mirrors LocalAnalyticsService: events are logged
// at debug level.
type LocalAnalyticsService struct {
	options map[string]any
	logger  *slog.Logger
}

// NewLocalAnalyticsService mirrors the constructor.
func NewLocalAnalyticsService(options map[string]any, logger *slog.Logger) *LocalAnalyticsService {
	if logger == nil {
		logger = slog.Default()
	}
	return &LocalAnalyticsService{options: options, logger: logger}
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

// orDash mirrors `${value ?? "-"}`.
func orDash(v any, ok bool) string {
	if !ok || v == nil {
		return "-"
	}
	if f, isNum := v.(float64); isNum && f == float64(int64(f)) {
		return fmt.Sprint(int64(f))
	}
	return fmt.Sprint(v)
}

// stringify mirrors JSON.stringify (undefined: "undefined").
func stringify(v any, ok bool) string {
	if !ok {
		return "undefined"
	}
	b, err := dal.MarshalValue(v)
	if err != nil {
		return "undefined"
	}
	return string(b)
}

// Track mirrors track.
func (s *LocalAnalyticsService) Track(ctx context.Context, data *dal.Record) error {
	event, _ := get(data, "event")
	actor, aok := get(data, "actor_id")
	group, gok := get(data, "group", "id")
	props, pok := get(data, "properties")
	s.logger.Debug(fmt.Sprintf("Tracking event: '%v', actor_id: '%s', group: '%s', properties: '%s'", event, orDash(actor, aok), orDash(group, gok), stringify(props, pok)))
	return nil
}

// Identify mirrors identify.
func (s *LocalAnalyticsService) Identify(ctx context.Context, data *dal.Record) error {
	actor, aok := get(data, "actor_id")
	group := "-"
	if _, hasGroup := get(data, "group"); hasGroup {
		id, ok := get(data, "group", "id")
		group = fmt.Sprint(id)
		if !ok {
			group = "undefined"
		}
	}
	props, pok := get(data, "properties")
	s.logger.Debug(fmt.Sprintf("Identifying user: '%s', group: '%s', properties: '%s'", orDash(actor, aok), group, stringify(props, pok)))
	return nil
}
