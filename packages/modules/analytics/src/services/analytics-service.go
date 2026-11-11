// Package services mirrors @medusajs/analytics/src/services.
package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// AnalyticsService mirrors AnalyticsService.
type AnalyticsService struct {
	provider *AnalyticsProviderService
}

// NewAnalyticsService builds the service.
func NewAnalyticsService(provider *AnalyticsProviderService) *AnalyticsService {
	return &AnalyticsService{provider: provider}
}

// GetProvider mirrors getProvider.
func (s *AnalyticsService) GetProvider() *AnalyticsProviderService { return s.provider }

// OnApplicationShutdown mirrors __hooks.onApplicationShutdown.
func (s *AnalyticsService) OnApplicationShutdown(ctx context.Context) error {
	return s.provider.Shutdown(ctx)
}

func field(r *dal.Record, path ...string) (any, bool) {
	var cur any = r
	ok := false
	for _, k := range path {
		rec, isRec := cur.(*dal.Record)
		if !isRec || rec == nil {
			return nil, false
		}
		cur, ok = rec.Get(k)
		if !ok {
			return nil, false
		}
	}
	return cur, ok
}

func jsString(v any, ok bool) string {
	if !ok {
		return "undefined"
	}
	if v == nil {
		return "null"
	}
	if f, isNum := v.(float64); isNum && f == float64(int64(f)) {
		return fmt.Sprint(int64(f))
	}
	return fmt.Sprint(v)
}

// Track mirrors track: provider errors become UNEXPECTED_STATE errors.
func (s *AnalyticsService) Track(ctx context.Context, data *dal.Record) error {
	if err := s.provider.Track(ctx, data); err != nil {
		event, ok := field(data, "event")
		return common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE, fmt.Sprintf("Error tracking event for %s: %s", jsString(event, ok), err.Error()))
	}
	return nil
}

// Identify mirrors identify.
func (s *AnalyticsService) Identify(ctx context.Context, data *dal.Record) error {
	if err := s.provider.Identify(ctx, data); err != nil {
		var who any
		var ok bool
		if _, hasGroup := field(data, "group"); hasGroup {
			who, ok = field(data, "group", "id")
		} else {
			who, ok = field(data, "actor_id")
		}
		return common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE, fmt.Sprintf("Error identifying event for %s: %s", jsString(who, ok), err.Error()))
	}
	return nil
}
