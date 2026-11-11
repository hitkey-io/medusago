package httpdiff

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	adyenclient "github.com/hitkey-io/medusago/packages/modules/providers/payment-adyen/src/client"
)

// fakeAdyen stands in for the Adyen Checkout API (v72) over TLS with a
// self-signed certificate, on the port of MEDUSA_TEST_ADYEN_URL. Test card
// numbers decide the result of /payments: test_4111 is authorised,
// test_refused refused, test_3ds asks for a redirect, test_invalid answers
// a validation error. GET /__requests lists the API requests.
type fakeAdyen struct {
	mu       sync.Mutex
	counter  int
	requests []any
	stored   []any
}

func startFakeAdyen(t *testing.T, rawURL string) *fakeAdyen {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", u.Host)
	if err != nil {
		t.Fatalf("fake adyen: %v", err)
	}
	f := &fakeAdyen{}
	srv := httptest.NewUnstartedServer(f)
	srv.Listener.Close()
	srv.Listener = ln
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return f
}

func (f *fakeAdyen) psp(prefix string) string {
	f.counter++
	return fmt.Sprintf("%sFAKE%012d", prefix, f.counter)
}

func adyenJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	raw, _ := dal.MarshalValue(v)
	_, _ = w.Write(raw)
}

func rget(r *dal.Record, path ...string) any {
	var cur any = r
	for _, p := range path {
		rec, ok := cur.(*dal.Record)
		if !ok {
			return nil
		}
		cur, _ = rec.Get(p)
	}
	return cur
}

func (f *fakeAdyen) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	raw, _ := io.ReadAll(r.Body)
	path := strings.TrimPrefix(r.URL.Path, "/v72")
	if r.URL.Path == "/__requests" {
		adyenJSON(w, 200, f.requests)
		return
	}
	entry := dal.NewRecord()
	entry.Set("method", r.Method)
	entry.Set("path", r.URL.Path)
	entry.Set("query", r.URL.RawQuery)
	entry.Set("body", string(raw))
	entry.Set("idempotency_key", r.Header.Get("Idempotency-Key"))
	entry.Set("api_key", r.Header.Get("X-API-Key"))
	entry.Set("library", r.Header.Get("adyen-library-name"))
	f.requests = append(f.requests, entry)

	var body *dal.Record
	if len(raw) > 0 {
		v, _ := dal.DecodeOrderedJSON(raw)
		body, _ = v.(*dal.Record)
	}
	merchant := rget(body, "merchantAccount")
	switch {
	case r.Method == http.MethodPost && path == "/paymentMethods":
		card := dal.NewRecord()
		card.Set("brands", []any{"visa", "mc"})
		card.Set("name", "Cards")
		card.Set("type", "scheme")
		card.Set("unknownField", "dropped")
		gp := dal.NewRecord()
		gp.Set("configuration", dal.NewRecord())
		gp.Set("name", "Google Pay")
		gp.Set("type", "googlepay")
		out := dal.NewRecord()
		out.Set("paymentMethods", []any{card, gp})
		if rget(body, "shopperReference") != nil {
			out.Set("storedPaymentMethods", f.stored)
		}
		out.Set("unknownTopLevel", true)
		adyenJSON(w, 200, out)
	case r.Method == http.MethodPost && path == "/payments":
		number, _ := rget(body, "paymentMethod", "encryptedCardNumber").(string)
		out := dal.NewRecord()
		amount := rget(body, "amount")
		switch number {
		case "test_invalid":
			e := dal.NewRecord()
			e.Set("status", float64(422))
			e.Set("errorCode", "101")
			e.Set("message", "Invalid card number")
			e.Set("errorType", "validation")
			e.Set("pspReference", f.psp("ERR"))
			adyenJSON(w, 422, e)
			return
		case "test_refused":
			out.Set("pspReference", f.psp("PSP"))
			out.Set("refusalReason", "Refused")
			out.Set("refusalReasonCode", "2")
			out.Set("resultCode", "Refused")
		case "test_3ds":
			action := dal.NewRecord()
			action.Set("paymentData", "fake-payment-data")
			action.Set("paymentMethodType", "scheme")
			action.Set("url", "https://checkout-test.adyen.com/3ds")
			action.Set("method", "GET")
			action.Set("type", "redirect")
			out.Set("action", action)
			out.Set("resultCode", "RedirectShopper")
		default:
			out.Set("additionalData", dal.NewRecord())
			out.Set("amount", amount)
			out.Set("merchantReference", rget(body, "reference"))
			out.Set("pspReference", f.psp("PSP"))
			out.Set("resultCode", "Authorised")
			out.Set("unknownField", "dropped")
		}
		adyenJSON(w, 200, out)
	case r.Method == http.MethodPost && path == "/payments/details":
		out := dal.NewRecord()
		out.Set("merchantReference", "details")
		out.Set("pspReference", f.psp("PSP"))
		out.Set("resultCode", "Authorised")
		adyenJSON(w, 200, out)
	case r.Method == http.MethodPost && strings.HasPrefix(path, "/payments/"):
		parts := strings.Split(strings.TrimPrefix(path, "/payments/"), "/")
		if len(parts) != 2 {
			adyenJSON(w, 404, nil)
			return
		}
		out := dal.NewRecord()
		if a := rget(body, "amount"); a != nil {
			out.Set("amount", a)
		}
		out.Set("merchantAccount", merchant)
		out.Set("paymentPspReference", parts[0])
		out.Set("pspReference", f.psp(strings.ToUpper(parts[1][:3])))
		out.Set("reference", rget(body, "reference"))
		out.Set("status", "received")
		adyenJSON(w, 201, out)
	case path == "/storedPaymentMethods" && r.Method == http.MethodGet:
		out := dal.NewRecord()
		out.Set("merchantAccount", r.URL.Query().Get("merchantAccount"))
		out.Set("shopperReference", r.URL.Query().Get("shopperReference"))
		out.Set("storedPaymentMethods", f.stored)
		adyenJSON(w, 200, out)
	case path == "/storedPaymentMethods" && r.Method == http.MethodPost:
		pm := dal.NewRecord()
		pm.Set("brand", "visa")
		pm.Set("expiryMonth", "03")
		pm.Set("expiryYear", "2030")
		pm.Set("id", f.psp("SPM"))
		pm.Set("lastFour", "1111")
		pm.Set("shopperReference", rget(body, "shopperReference"))
		pm.Set("type", "scheme")
		f.stored = append(f.stored, pm)
		adyenJSON(w, 200, pm)
	case strings.HasPrefix(path, "/storedPaymentMethods/") && r.Method == http.MethodDelete:
		w.WriteHeader(204)
	default:
		adyenJSON(w, 404, nil)
	}
}

// adyenSign sets the HMAC signature of every notification item of a
// standard webhook body.
func adyenSign(body string) string {
	v, err := dal.DecodeOrderedJSON([]byte(body))
	if err != nil {
		return body
	}
	r, _ := v.(*dal.Record)
	items, _ := rget(r, "notificationItems").([]any)
	key := envOr("ADYEN_HMAC_KEY", "44782DEF547AAA06C910C43932B1EB0C71FC68D9D0C057550C48EC2ACF6BA056")
	for _, x := range items {
		item, _ := rget(x.(*dal.Record), "NotificationRequestItem").(*dal.Record)
		if item == nil {
			continue
		}
		add, _ := rget(item, "additionalData").(*dal.Record)
		if add == nil {
			add = dal.NewRecord()
			item.Set("additionalData", add)
		}
		sig, _ := adyenclient.CalculateHmac(adyenclient.DataToSign(item), key)
		add.Set(adyenclient.HmacSignatureKey, sig)
	}
	raw, _ := dal.MarshalValue(r)
	return string(raw)
}
