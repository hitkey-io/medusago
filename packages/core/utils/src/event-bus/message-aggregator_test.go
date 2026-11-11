package eventbus

import (
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func TestMessageAggregatorDeduplicates(t *testing.T) {
	a := NewMessageAggregator()
	data := func(id string) *dal.Record {
		r := dal.NewRecord()
		r.Set("id", id)
		return r
	}
	a.SaveRawMessageData(
		RawMessage{EventName: "product.product.created", Source: "product", Object: "product", Action: "created", Data: data("prod_1"), EventGroupID: "g"},
		RawMessage{EventName: "product.product.created", Source: "product", Object: "product", Action: "created", Data: data("prod_1"), EventGroupID: "g"},
		RawMessage{EventName: "product.product.created", Source: "product", Object: "product", Action: "created", Data: data("prod_2"), EventGroupID: "g"},
	)
	msgs := a.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	raw, _ := dal.MarshalValue(msgs[0].Metadata)
	if string(raw) != `{"source":"product","object":"product","action":"created","eventGroupId":"g"}` {
		t.Fatalf("unexpected metadata %s", raw)
	}
}

func TestBuildEventNamesFromEntityName(t *testing.T) {
	names := BuildEventNamesFromEntityName([]string{"serviceZone"}, "fulfillment")
	if names["SERVICE_ZONE_CREATED"] != "fulfillment.service-zone.created" || names["SERVICE_ZONE_DETACHED"] != "fulfillment.service-zone.detached" {
		t.Fatalf("unexpected names %v", names)
	}
	if got := BuildModuleResourceEventName("product", "product_variant", "updated"); got != "product.product-variant.updated" {
		t.Fatalf("unexpected name %s", got)
	}
}
