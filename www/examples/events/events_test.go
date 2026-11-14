package events

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/container"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	eventbuslocal "github.com/hitkey-io/medusago/packages/modules/event-bus-local/src/services"
)

func newApp() (*loaders.MedusaApp, *eventbuslocal.LocalEventBusService) {
	bus := eventbuslocal.NewLocalEventBusService(nil)
	c := container.New()
	c.Register(utilsmodulessdk.EVENT_BUS, bus)
	c.Register(common.ContainerRegistrationKeys.LOGGER, slog.Default())
	return &loaders.MedusaApp{Container: c}, bus
}

func payload(id string) *dal.Record {
	r := dal.NewRecord()
	r.Set("id", id)
	return r
}

func TestRegisterSubscribers(t *testing.T) {
	app, bus := newApp()
	if err := RegisterSubscribers(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	// the subscribers run asynchronously, and they log without failing
	_ = bus.Emit(context.Background(), []eventbus.Message{{Name: "product.created", Data: payload("prod_1")}})
	_ = bus.Emit(context.Background(), []eventbus.Message{{Name: "order.placed", Data: payload("order_1")}})
	bus.Wait()
}

func TestWorkflowEmitsAfterItFinishes(t *testing.T) {
	app, bus := newApp()
	received := make(chan eventbus.Event, 1)
	bus.Subscribe("custom.created", func(ctx context.Context, ev eventbus.Event) error {
		received <- ev
		return nil
	})

	if _, err := HelloWorldWorkflow.Run(context.Background(), app.Container, struct{}{}); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-received:
		data, _ := ev.Data.(*dal.Record)
		if id, _ := data.Get("id"); id != "123" {
			t.Fatalf("unexpected payload: %v", ev.Data)
		}
		name, _, createdAt := describeEvent(ev)
		if name != "custom.created" || createdAt == nil {
			t.Fatalf("unexpected event: %s %v", name, createdAt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the event wasn't delivered")
	}
}

func TestServiceEvent(t *testing.T) {
	app, bus := newApp()
	received := make(chan eventbus.Event, 1)
	bus.Subscribe("custom.event", func(ctx context.Context, ev eventbus.Event) error {
		received <- ev
		return nil
	})

	svc := &BlogModuleService{container: app.Container}
	if err := svc.PerformAction(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("the event wasn't delivered")
	}
}

func TestWorkflowDoesNotEmitWhenItFails(t *testing.T) {
	app, bus := newApp()
	received := make(chan eventbus.Event, 1)
	bus.Subscribe("custom.created", func(ctx context.Context, ev eventbus.Event) error {
		received <- ev
		return nil
	})

	failing := composer.CreateWorkflow("failing-hello-world", func(e *composer.Execution, input struct{}) (any, error) {
		if _, err := HelloWorldWorkflow.RunAsStep(e, struct{}{}); err != nil {
			return nil, err
		}
		return nil, errors.New("something went wrong")
	})
	if _, err := failing.Run(context.Background(), app.Container, struct{}{}); err == nil {
		t.Fatal("the workflow must fail")
	}

	select {
	case <-received:
		t.Fatal("the event of a failed workflow must not be emitted")
	case <-time.After(300 * time.Millisecond):
	}
}
