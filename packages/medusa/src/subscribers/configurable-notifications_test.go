package subscribers

import (
	"context"
	"errors"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

type fakeNotifications struct {
	got []*dal.Record
	err error
}

func (f *fakeNotifications) CreateNotifications(_ context.Context, data []*dal.Record) ([]*dal.Record, error) {
	f.got = append(f.got, data...)
	return data, f.err
}

type fakeContainer map[string]any

func (c fakeContainer) Resolve(key string) any { return c[key] }

func TestConfigurableNotifications(t *testing.T) {
	if got := ConfigurableNotificationsEvents(); len(got) != 1 || got[0] != "order.created" {
		t.Fatalf("events = %v", got)
	}
	svc := &fakeNotifications{}
	c := fakeContainer{utilsmodulessdk.NOTIFICATION: svc}

	order := dal.NewRecord()
	order.Set("id", "order_1")
	order.Set("email", "buyer@example.test")
	payload := dal.NewRecord()
	payload.Set("order", order)
	if err := ConfigurableNotificationsHandler(context.Background(), c, "order.created", payload); err != nil {
		t.Fatal(err)
	}
	if len(svc.got) != 1 {
		t.Fatalf("notifications = %d", len(svc.got))
	}
	raw, _ := svc.got[0].MarshalJSON()
	want := `{"template":"order-created-template","channel":"email","to":"buyer@example.test","trigger_type":"order.created","resource_id":"order_1","data":{"order_id":"order_1"}}`
	if string(raw) != want {
		t.Fatalf("notification = %s", raw)
	}

	// undefined values are left out ({ id } payload of a module event)
	svc.got = nil
	p := dal.NewRecord()
	p.Set("id", "order_1")
	if err := ConfigurableNotificationsHandler(context.Background(), c, "order.created", p); err != nil {
		t.Fatal(err)
	}
	raw, _ = svc.got[0].MarshalJSON()
	if want := `{"template":"order-created-template","channel":"email","trigger_type":"order.created","data":{}}`; string(raw) != want {
		t.Fatalf("notification = %s", raw)
	}

	// failures are logged only; other events are ignored
	svc.err = errors.New("no provider")
	svc.got = nil
	if err := ConfigurableNotificationsHandler(context.Background(), c, "order.created", p); err != nil {
		t.Fatal(err)
	}
	if err := ConfigurableNotificationsHandler(context.Background(), c, "order.placed", p); err != nil || len(svc.got) != 1 {
		t.Fatalf("err = %v, notifications = %d", err, len(svc.got))
	}
}
