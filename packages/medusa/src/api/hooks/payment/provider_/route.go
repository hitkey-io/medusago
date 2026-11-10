// Package provider_ mirrors api/hooks/payment/[provider].
package provider_

import (
	"context"
	"fmt"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type providerLister interface {
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
}

// PaymentWebhookReceived mirrors PaymentWebhookEvents.WebhookReceived.
const PaymentWebhookReceived = "payment.webhook_received"

func handle(req *medusahttp.MedusaRequest) error {
	provider := req.Params["provider"]
	providerID := provider
	if !strings.HasPrefix(providerID, "pp_") {
		providerID = "pp_" + provider
	}
	svc, _ := req.Scope.Resolve(utilsmodulessdk.PAYMENT).(providerLister)
	f := dal.NewRecord()
	f.Set("id", providerID)
	f.Set("is_enabled", true)
	providers, err := svc.ListBySuffix(req.Ctx(), "PaymentProviders", f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Payment provider %s is not configured", providerID))
	}
	headers := dal.NewRecord()
	for k, v := range req.Header {
		headers.Set(strings.ToLower(k), strings.Join(v, ", "))
	}
	payload := dal.NewRecord()
	payload.Set("data", req.Body)
	payload.Set("rawData", req.RawBody)
	payload.Set("headers", headers)
	event := dal.NewRecord()
	event.Set("provider", provider)
	event.Set("payload", payload)
	bus, _ := req.Scope.Resolve(utilsmodulessdk.EVENT_BUS).(composer.EventBus)
	if bus == nil {
		return nil
	}
	options := dal.NewRecord()
	options.Set("delay", float64(5000))
	options.Set("attempts", float64(3))
	return bus.Emit(req.Ctx(), []composer.Message{{Name: PaymentWebhookReceived, Data: event, Options: options}})
}

// POST mirrors POST: the webhook is processed by a subscriber.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if err := handle(req); err != nil {
		res.Status(400).Send("Webhook Error: " + err.Error())
		return nil
	}
	res.SendStatus(200)
	return nil
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}

// Middlewares mirrors hooksRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/hooks/payment/:provider", BodyParser: &medusahttp.BodyParserConfig{PreserveRawBody: true}},
}
