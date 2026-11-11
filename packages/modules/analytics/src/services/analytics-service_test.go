package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	analyticstypes "github.com/hitkey-io/medusago/packages/core/types/src/analytics"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/analytics/src/services"
	posthog "github.com/hitkey-io/medusago/packages/modules/providers/analytics-posthog/src/services"
)

// The module's error wrapping, against the upstream AnalyticsService over
// the PostHog provider (vectors of scripts/extract-provider-vectors.js).
func TestAnalyticsServiceMatchesUpstream(t *testing.T) {
	raw, err := os.ReadFile("../../../providers/analytics-posthog/src/services/testdata/upstream-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		ModuleCalls []struct {
			Method string          `json:"method"`
			Data   json.RawMessage `json:"data"`
			Error  struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"module_calls"`
		ProviderServiceError struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"provider_service_error"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	check := func(name string, err error, typ, msg string) {
		var me *common.MedusaError
		if !errors.As(err, &me) || me.Type != typ || me.Message != msg {
			t.Errorf("%s: %v, want %s %q", name, err, typ, msg)
		}
	}
	_, err = services.NewAnalyticsProviderService(nil)
	check("provider service", err, v.ProviderServiceError.Type, v.ProviderServiceError.Message)

	p, err := posthog.NewPosthogAnalyticsService(map[string]any{"posthogEventsKey": "phc_test", "posthogHost": "http://127.0.0.1:9"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ps, err := services.NewAnalyticsProviderService([]analyticstypes.IAnalyticsProvider{p})
	if err != nil {
		t.Fatal(err)
	}
	s := services.NewAnalyticsService(ps)
	for _, c := range v.ModuleCalls {
		dv, _ := dal.DecodeOrderedJSON(c.Data)
		data := dv.(*dal.Record)
		if c.Method == "track" {
			err = s.Track(context.Background(), data)
		} else {
			err = s.Identify(context.Background(), data)
		}
		check(c.Method+" "+string(c.Data), err, c.Error.Type, c.Error.Message)
	}
}
