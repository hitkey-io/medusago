package httpdiff

import (
	"context"
	"sync"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	eventbuslocal "github.com/hitkey-io/medusago/packages/modules/event-bus-local/src/services"
)

// registerEventsCapture adds the test-only /events-capture route of the
// reference application (src/api/events-capture/route.js of its copy): an
// event bus interceptor records every published event in order; POST
// starts capturing (clearing what was captured), GET returns the events as
// { name, data, metadata (without created_at / published_at), options }.
func registerEventsCapture(_ context.Context, app *loaders.MedusaApp) error {
	var (
		mu        sync.Mutex
		captured  = []any{}
		installed bool
	)
	install := func() {
		mu.Lock()
		defer mu.Unlock()
		if installed {
			return
		}
		installed = true
		bus, ok := app.Container.Resolve(utilsmodulessdk.EVENT_BUS).(*eventbuslocal.LocalEventBusService)
		if !ok {
			return
		}
		bus.AddInterceptor(func(m composer.Message, _ bool, _ string) {
			metadata := dal.NewRecord()
			if m.Metadata != nil {
				for _, k := range m.Metadata.Keys() {
					if k == "created_at" || k == "published_at" {
						continue
					}
					v, _ := m.Metadata.Get(k)
					metadata.Set(k, v)
				}
			}
			ev := dal.NewRecord()
			ev.Set("name", m.Name)
			ev.Set("data", m.Data)
			ev.Set("metadata", metadata)
			if m.Options != nil {
				ev.Set("options", m.Options)
			} else {
				ev.Set("options", dal.NewRecord())
			}
			// the payload is copied like a JSON round trip (subscribers may
			// keep the objects)
			raw, err := dal.MarshalValue(ev)
			if err != nil {
				return
			}
			v, err := dal.DecodeOrderedJSON(raw)
			if err != nil {
				return
			}
			mu.Lock()
			captured = append(captured, v)
			mu.Unlock()
		})
	}
	respond := func(res *medusahttp.MedusaResponse, events []any) error {
		body := dal.NewRecord()
		body.Set("events", events)
		return res.JSON(body)
	}
	app.App.Handle("POST", "/events-capture", func(_ *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, _ medusahttp.NextFunction) error {
		install()
		mu.Lock()
		captured = []any{}
		mu.Unlock()
		return respond(res, []any{})
	})
	app.App.Handle("GET", "/events-capture", func(_ *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, _ medusahttp.NextFunction) error {
		install()
		mu.Lock()
		events := append([]any{}, captured...)
		mu.Unlock()
		return respond(res, events)
	})
	return nil
}
