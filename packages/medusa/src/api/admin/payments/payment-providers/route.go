// Package paymentproviders mirrors api/admin/payments/payment-providers.
package paymentproviders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments"
)

// Route is route.ts.
var Route = medusahttp.Route{GET: payments.ListProviders}
