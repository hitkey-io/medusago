package httpdiff

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	adyenclient "github.com/hitkey-io/medusago/packages/modules/providers/payment-adyen/src/client"
	stripecore "github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
)

// serviceURLs are the base URLs of the fake third-party services of the
// running scenario (steps with "service" are sent there).
var serviceURLs = map[string]string{}

// startFakes starts the fake services a scenario environment names and
// points the Go providers at them (the reference application's
// medusa-config.js does the same for upstream).
func startFakes(t *testing.T, env map[string]string) {
	t.Helper()
	serviceURLs = map[string]string{}
	if u := env["MEDUSA_TEST_STRIPE_URL"]; u != "" {
		startFakeStripe(t, u)
		serviceURLs["stripe"] = strings.TrimSuffix(u, "/")
		pu, _ := url.Parse(u)
		host, port, protocol := stripecore.DefaultHost, stripecore.DefaultPort, stripecore.DefaultProtocol
		stripecore.DefaultHost, stripecore.DefaultPort, stripecore.DefaultProtocol = pu.Hostname(), pu.Port(), pu.Scheme
		t.Cleanup(func() {
			stripecore.DefaultHost, stripecore.DefaultPort, stripecore.DefaultProtocol = host, port, protocol
		})
	}
	if u := env["MEDUSA_TEST_ADYEN_URL"]; u != "" {
		startFakeAdyen(t, u)
		serviceURLs["adyen"] = strings.TrimSuffix(u, "/")
		// the fake's certificate is self-signed (upstream runs with
		// NODE_TLS_REJECT_UNAUTHORIZED=0)
		insecure := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		prevClient, prevOverride := adyenclient.HTTPClient, adyenclient.EndpointOverride
		adyenclient.HTTPClient = &http.Client{Transport: insecure}
		adyenclient.EndpointOverride = u
		prevDefault := http.DefaultTransport
		http.DefaultTransport = insecure
		t.Cleanup(func() {
			adyenclient.HTTPClient, adyenclient.EndpointOverride = prevClient, prevOverride
			http.DefaultTransport = prevDefault
		})
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// paymentModule mirrors the payment module entry of the reference
// application's medusa-config.js.
func paymentModule() (config.ModuleConfig, bool) {
	var providers []config.ModuleProviderConfig
	if os.Getenv("MEDUSA_TEST_STRIPE_URL") != "" {
		options := map[string]any{
			"apiKey":        "sk_test_fake",
			"webhookSecret": envOr("STRIPE_WEBHOOK_SECRET", "whsec_test_fake"),
			"capture":       os.Getenv("STRIPE_CAPTURE") == "true",
		}
		if m := os.Getenv("STRIPE_ASYNC_METHODS"); m != "" {
			var l []any
			for _, s := range strings.Split(m, ",") {
				l = append(l, s)
			}
			options["asyncPaymentMethodTypes"] = l
		}
		providers = append(providers, config.ModuleProviderConfig{Resolve: "@medusajs/medusa/payment-stripe", ID: "stripe", Options: options})
	}
	if os.Getenv("MEDUSA_TEST_ADYEN_URL") != "" {
		providers = append(providers, config.ModuleProviderConfig{Resolve: "@medusajs/medusa/payment-adyen", ID: "adyen", Options: map[string]any{
			"apiKey":          "adyen_test_fake",
			"merchantAccount": "MedusaTestAccount",
			"hmacKey":         envOr("ADYEN_HMAC_KEY", "44782DEF547AAA06C910C43932B1EB0C71FC68D9D0C057550C48EC2ACF6BA056"),
			"capture":         os.Getenv("ADYEN_CAPTURE") == "true",
			"returnUrl":       "http://localhost:8000/checkout",
		}})
	}
	if len(providers) == 0 {
		return config.ModuleConfig{}, false
	}
	return config.ModuleConfig{Key: "payment", Resolve: "@medusajs/medusa/payment", Providers: providers}, true
}

// fulfillmentModule mirrors the fulfillment module entry of the reference
// application's medusa-config.js.
func fulfillmentModule() (config.ModuleConfig, bool) {
	if os.Getenv("MEDUSA_TEST_CALCULATED_FULFILLMENT") != "true" {
		return config.ModuleConfig{}, false
	}
	return config.ModuleConfig{Key: "fulfillment", Resolve: "@medusajs/medusa/fulfillment", Providers: []config.ModuleProviderConfig{
		{Resolve: "@medusajs/medusa/fulfillment-manual", ID: "manual"},
		{Resolve: testCalculatedResolve, ID: "calc"},
	}}, true
}

// signBody signs a webhook body for a provider: it returns the body to send
// and the headers to add.
func signBody(scheme, body string) (string, map[string]string) {
	switch scheme {
	case "stripe":
		return body, map[string]string{"stripe-signature": stripeSignature(body)}
	case "adyen":
		return adyenSign(body), nil
	}
	return body, nil
}

// stripeSignature signs a webhook body like Stripe (stripe-signature).
func stripeSignature(body string) string {
	return stripecore.GenerateTestHeaderString([]byte(body), envOr("STRIPE_WEBHOOK_SECRET", "whsec_test_fake"), time.Now().Unix())
}
