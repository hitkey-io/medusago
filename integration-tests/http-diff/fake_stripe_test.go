package httpdiff

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// fakeStripe is a small stateful stand-in for the Stripe API, served on the
// port of MEDUSA_TEST_STRIPE_URL to both the upstream server (record.sh)
// and the Go server. It keeps payment intents, refunds and customers in
// memory, records every API request (GET /__requests) and lets scenarios
// play the storefront (confirming an intent with a test payment method) and
// Stripe itself (POST /__simulate/<intent>/<event type> changes the intent
// and returns the webhook event).
type fakeStripe struct {
	mu        sync.Mutex
	counters  map[string]int
	intents   map[string]*dal.Record
	customers map[string]*dal.Record
	refunded  map[string]float64
	requests  []any
}

// fakeCreated is the creation timestamp of every fake object.
const fakeCreated = float64(1767225600)

func newFakeStripe() *fakeStripe {
	return &fakeStripe{counters: map[string]int{}, intents: map[string]*dal.Record{}, customers: map[string]*dal.Record{}, refunded: map[string]float64{}}
}

// startFakeStripe serves the fake on the address of rawURL until the test ends.
func startFakeStripe(t *testing.T, rawURL string) *fakeStripe {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", u.Host)
	if err != nil {
		t.Fatalf("fake stripe: %v", err)
	}
	f := newFakeStripe()
	srv := &http.Server{Handler: f}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return f
}

func (f *fakeStripe) nextID(prefix string) string {
	f.counters[prefix]++
	return fmt.Sprintf("%s_fake%04d", prefix, f.counters[prefix])
}

var retryKeyRe = regexp.MustCompile(`^stripe-node-retry-[0-9a-f-]{36}$`)

// parseForm decodes a stripe-node form body (key[child]=v, key[0]=v) into
// an ordered record; values stay strings.
func parseForm(raw string) *dal.Record {
	out := dal.NewRecord()
	if raw == "" {
		return out
	}
	for _, pair := range strings.Split(raw, "&") {
		k, v, _ := strings.Cut(pair, "=")
		key, _ := url.QueryUnescape(k)
		val, _ := url.QueryUnescape(v)
		var path []string
		head, rest, _ := strings.Cut(key, "[")
		path = append(path, head)
		if rest != "" {
			for _, seg := range strings.Split(strings.TrimSuffix(rest, "]"), "][") {
				path = append(path, seg)
			}
		}
		cur := out
		for i, seg := range path {
			if i == len(path)-1 {
				cur.Set(seg, val)
				break
			}
			next, _ := cur.Get(seg)
			nr, ok := next.(*dal.Record)
			if !ok {
				nr = dal.NewRecord()
				cur.Set(seg, nr)
			}
			cur = nr
		}
	}
	return toArrays(out).(*dal.Record)
}

// toArrays turns records keyed 0..n-1 into arrays.
func toArrays(v any) any {
	r, ok := v.(*dal.Record)
	if !ok {
		return v
	}
	keys := r.Keys()
	isArray := len(keys) > 0
	for i, k := range keys {
		if k != strconv.Itoa(i) {
			isArray = false
		}
	}
	if isArray {
		out := make([]any, len(keys))
		for i, k := range keys {
			x, _ := r.Get(k)
			out[i] = toArrays(x)
		}
		return out
	}
	for _, k := range keys {
		x, _ := r.Get(k)
		r.Set(k, toArrays(x))
	}
	return r
}

func num(s any) float64 {
	f, _ := strconv.ParseFloat(fmt.Sprint(s), 64)
	return f
}

func str(r *dal.Record, k string) string {
	v, _ := r.Get(k)
	s, _ := v.(string)
	return s
}

func stripeError(w http.ResponseWriter, status int, fields *dal.Record) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Request-Id", "req_fake")
	w.WriteHeader(status)
	body := dal.NewRecord()
	body.Set("error", fields)
	raw, _ := dal.MarshalValue(body)
	_, _ = w.Write(raw)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Request-Id", "req_fake")
	raw, _ := dal.MarshalValue(v)
	_, _ = w.Write(raw)
}

func clone(r *dal.Record) *dal.Record {
	raw, _ := dal.MarshalValue(r)
	v, _ := dal.DecodeOrderedJSON(raw)
	return v.(*dal.Record)
}

// paymentMethod builds the test payment methods: pm_card_visa,
// pm_card_chargeDeclined and pm_sepa_debit (an asynchronous method).
func paymentMethod(id string) *dal.Record {
	pm := dal.NewRecord()
	pm.Set("id", id)
	pm.Set("object", "payment_method")
	pm.Set("created", fakeCreated)
	pm.Set("customer", nil)
	pm.Set("livemode", false)
	pm.Set("metadata", dal.NewRecord())
	if id == "pm_sepa_debit" {
		pm.Set("type", "sepa_debit")
		sepa := dal.NewRecord()
		sepa.Set("country", "DE")
		sepa.Set("last4", "3000")
		pm.Set("sepa_debit", sepa)
		return pm
	}
	pm.Set("type", "card")
	card := dal.NewRecord()
	card.Set("brand", "visa")
	card.Set("exp_month", float64(12))
	card.Set("exp_year", float64(2034))
	card.Set("last4", "4242")
	pm.Set("card", card)
	return pm
}

func (f *fakeStripe) view(intent *dal.Record, expand []any) *dal.Record {
	out := clone(intent)
	for _, e := range expand {
		if e == "payment_method" {
			if id := str(out, "payment_method"); id != "" {
				out.Set("payment_method", paymentMethod(id))
			}
		}
	}
	return out
}

func expandOf(params *dal.Record) []any {
	v, _ := params.Get("expand")
	l, _ := v.([]any)
	return l
}

func (f *fakeStripe) createIntent(params *dal.Record) *dal.Record {
	id := f.nextID("pi")
	pi := dal.NewRecord()
	pi.Set("id", id)
	pi.Set("object", "payment_intent")
	pi.Set("amount", num(str(params, "amount")))
	pi.Set("amount_capturable", float64(0))
	pi.Set("amount_received", float64(0))
	pi.Set("canceled_at", nil)
	pi.Set("cancellation_reason", nil)
	captureMethod := str(params, "capture_method")
	if captureMethod == "" {
		captureMethod = "automatic_async"
	}
	pi.Set("capture_method", captureMethod)
	pi.Set("client_secret", id+"_secret_fake")
	pi.Set("confirmation_method", "automatic")
	pi.Set("created", fakeCreated)
	pi.Set("currency", str(params, "currency"))
	if c := str(params, "customer"); c != "" {
		pi.Set("customer", c)
	} else {
		pi.Set("customer", nil)
	}
	if d := str(params, "description"); d != "" {
		pi.Set("description", d)
	} else {
		pi.Set("description", nil)
	}
	pi.Set("last_payment_error", nil)
	pi.Set("latest_charge", nil)
	pi.Set("livemode", false)
	md, _ := params.Get("metadata")
	if m, ok := md.(*dal.Record); ok {
		pi.Set("metadata", m)
	} else {
		pi.Set("metadata", dal.NewRecord())
	}
	pi.Set("next_action", nil)
	pi.Set("payment_method", nil)
	if o, ok := params.Get("payment_method_options"); ok {
		pi.Set("payment_method_options", o)
	} else {
		pi.Set("payment_method_options", dal.NewRecord())
	}
	types, ok := params.Get("payment_method_types")
	if !ok {
		types = []any{"card"}
	}
	pi.Set("payment_method_types", types)
	if s := str(params, "setup_future_usage"); s != "" {
		pi.Set("setup_future_usage", s)
	} else {
		pi.Set("setup_future_usage", nil)
	}
	pi.Set("status", "requires_payment_method")
	f.intents[id] = pi
	return pi
}

// confirm applies a test payment method: declined cards fail (card error),
// asynchronous methods are processing, others succeed or wait for capture.
func (f *fakeStripe) confirm(pi *dal.Record, pm string) *dal.Record {
	pi.Set("payment_method", pm)
	switch pm {
	case "pm_card_chargeDeclined":
		e := dal.NewRecord()
		e.Set("code", "card_declined")
		e.Set("decline_code", "generic_decline")
		e.Set("message", "Your card was declined.")
		e.Set("type", "card_error")
		pi.Set("last_payment_error", e)
		pi.Set("status", "requires_payment_method")
		pi.Set("payment_method", nil)
		ef := dal.NewRecord()
		ef.Set("code", "card_declined")
		ef.Set("decline_code", "generic_decline")
		ef.Set("message", "Your card was declined.")
		ef.Set("payment_intent", clone(pi))
		ef.Set("type", "card_error")
		return ef
	case "pm_sepa_debit":
		pi.Set("status", "processing")
		return nil
	}
	f.succeedOrAuthorize(pi)
	return nil
}

func (f *fakeStripe) succeedOrAuthorize(pi *dal.Record) {
	amount, _ := pi.Get("amount")
	if str(pi, "capture_method") == "manual" {
		pi.Set("amount_capturable", amount)
		pi.Set("status", "requires_capture")
	} else {
		pi.Set("amount_received", amount)
		pi.Set("latest_charge", f.nextID("ch"))
		pi.Set("status", "succeeded")
	}
}

func unexpectedState(w http.ResponseWriter, pi *dal.Record, msg string) {
	e := dal.NewRecord()
	e.Set("code", "payment_intent_unexpected_state")
	e.Set("message", msg)
	e.Set("payment_intent", clone(pi))
	e.Set("type", "invalid_request_error")
	stripeError(w, 400, e)
}

func notFound(w http.ResponseWriter, kind, id string) {
	e := dal.NewRecord()
	e.Set("code", "resource_missing")
	e.Set("message", fmt.Sprintf("No such %s: '%s'", kind, id))
	e.Set("type", "invalid_request_error")
	stripeError(w, 404, e)
}

func (f *fakeStripe) event(eventType string, pi *dal.Record) *dal.Record {
	ev := dal.NewRecord()
	ev.Set("id", f.nextID("evt"))
	ev.Set("object", "event")
	ev.Set("api_version", "2024-04-10")
	ev.Set("created", fakeCreated)
	data := dal.NewRecord()
	data.Set("object", clone(pi))
	ev.Set("data", data)
	ev.Set("livemode", false)
	ev.Set("pending_webhooks", float64(1))
	ev.Set("type", eventType)
	out := dal.NewRecord()
	out.Set("event", ev)
	return out
}

func (f *fakeStripe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	raw, _ := io.ReadAll(r.Body)
	body := string(raw)
	path := r.URL.Path

	if path == "/__requests" {
		writeJSON(w, f.requests)
		return
	}
	if strings.HasPrefix(path, "/__simulate/") {
		// /__simulate/<intent>/<event type>
		parts := strings.SplitN(strings.TrimPrefix(path, "/__simulate/"), "/", 2)
		pi := f.intents[parts[0]]
		if pi == nil || len(parts) < 2 {
			notFound(w, "payment_intent", parts[0])
			return
		}
		switch parts[1] {
		case "payment_intent.succeeded":
			amount, _ := pi.Get("amount")
			pi.Set("amount_received", amount)
			pi.Set("amount_capturable", float64(0))
			pi.Set("latest_charge", f.nextID("ch"))
			pi.Set("status", "succeeded")
		case "payment_intent.amount_capturable_updated":
			amount, _ := pi.Get("amount")
			pi.Set("amount_capturable", amount)
			pi.Set("status", "requires_capture")
		case "payment_intent.payment_failed":
			pi.Set("status", "requires_payment_method")
		case "payment_intent.canceled":
			pi.Set("status", "canceled")
			pi.Set("canceled_at", fakeCreated)
		}
		writeJSON(w, f.event(parts[1], pi))
		return
	}

	// Requests without credentials are the storefront's (Stripe.js
	// confirming an intent): not recorded, not authenticated.
	storefront := r.Header.Get("Authorization") == ""
	entry := dal.NewRecord()
	entry.Set("method", r.Method)
	entry.Set("path", path)
	entry.Set("query", r.URL.RawQuery)
	entry.Set("body", body)
	key := r.Header.Get("Idempotency-Key")
	if retryKeyRe.MatchString(key) {
		key = "stripe-node-retry-<uuid>"
	}
	entry.Set("idempotency_key", key)
	entry.Set("stripe_version", r.Header.Get("Stripe-Version"))
	entry.Set("content_type", r.Header.Get("Content-Type"))
	if !storefront {
		f.requests = append(f.requests, entry)
	}

	if !storefront && !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer sk_test_") {
		e := dal.NewRecord()
		e.Set("message", "Invalid API Key provided")
		e.Set("type", "invalid_request_error")
		stripeError(w, 401, e)
		return
	}
	params := parseForm(body)
	// query parameters (GET, and the storefront steps of the scenarios)
	q := parseForm(r.URL.RawQuery)
	for _, k := range q.Keys() {
		v, _ := q.Get(k)
		params.Set(k, v)
	}
	segs := strings.Split(strings.TrimPrefix(path, "/v1/"), "/")
	switch {
	case r.Method == http.MethodPost && path == "/v1/payment_intents":
		// a description "fake_failure" fails the creation (scenarios
		// reverting the workflow after the account holder was created)
		if str(params, "description") == "fake_failure" {
			e := dal.NewRecord()
			e.Set("message", "Fake failure")
			e.Set("type", "invalid_request_error")
			stripeError(w, 400, e)
			return
		}
		pi := f.createIntent(params)
		if str(params, "confirm") == "true" && str(params, "payment_method") != "" {
			if e := f.confirm(pi, str(params, "payment_method")); e != nil {
				stripeError(w, 402, e)
				return
			}
		}
		writeJSON(w, f.view(pi, expandOf(params)))
	case segs[0] == "payment_intents" && len(segs) >= 2:
		pi := f.intents[segs[1]]
		if pi == nil {
			notFound(w, "payment_intent", segs[1])
			return
		}
		action := ""
		if len(segs) > 2 {
			action = segs[2]
		}
		switch {
		case r.Method == http.MethodGet && action == "":
			writeJSON(w, f.view(pi, expandOf(params)))
		case r.Method == http.MethodPost && action == "":
			if a, ok := params.Get("amount"); ok {
				pi.Set("amount", num(a))
			}
			writeJSON(w, f.view(pi, expandOf(params)))
		case action == "confirm":
			if st := str(pi, "status"); st != "requires_payment_method" && st != "requires_confirmation" {
				unexpectedState(w, pi, fmt.Sprintf("This PaymentIntent's payment_method could not be updated because it has a status of %s. You may only update the payment_method of a PaymentIntent with one of the following statuses: requires_payment_method, requires_confirmation, requires_action.", st))
				return
			}
			if e := f.confirm(pi, str(params, "payment_method")); e != nil {
				stripeError(w, 402, e)
				return
			}
			writeJSON(w, f.view(pi, expandOf(params)))
		case action == "capture":
			if st := str(pi, "status"); st != "requires_capture" {
				unexpectedState(w, pi, fmt.Sprintf("This PaymentIntent could not be captured because it has a status of %s. Only a PaymentIntent with one of the following statuses may be captured: requires_capture.", st))
				return
			}
			amount, _ := pi.Get("amount_capturable")
			pi.Set("amount_received", amount)
			pi.Set("amount_capturable", float64(0))
			pi.Set("latest_charge", f.nextID("ch"))
			pi.Set("status", "succeeded")
			writeJSON(w, f.view(pi, expandOf(params)))
		case action == "cancel":
			switch str(pi, "status") {
			case "requires_payment_method", "requires_capture", "requires_confirmation", "requires_action", "processing":
				pi.Set("amount_capturable", float64(0))
				pi.Set("canceled_at", fakeCreated)
				pi.Set("status", "canceled")
				writeJSON(w, f.view(pi, expandOf(params)))
			default:
				unexpectedState(w, pi, fmt.Sprintf("You cannot cancel this PaymentIntent because it has a status of %s. Only a PaymentIntent with one of the following statuses may be canceled: requires_payment_method, requires_capture, requires_confirmation, requires_action, processing.", str(pi, "status")))
			}
		default:
			notFound(w, "payment_intent", segs[1])
		}
	case r.Method == http.MethodPost && path == "/v1/refunds":
		piID := str(params, "payment_intent")
		pi := f.intents[piID]
		if pi == nil {
			notFound(w, "payment_intent", piID)
			return
		}
		if str(pi, "status") != "succeeded" {
			e := dal.NewRecord()
			e.Set("message", fmt.Sprintf("This PaymentIntent (%s) does not have a successful charge to refund.", piID))
			e.Set("type", "invalid_request_error")
			stripeError(w, 400, e)
			return
		}
		received, _ := pi.Get("amount_received")
		amount := num(str(params, "amount"))
		if f.refunded[piID] >= received.(float64) {
			e := dal.NewRecord()
			e.Set("code", "charge_already_refunded")
			e.Set("message", fmt.Sprintf("Charge %s has already been refunded.", str(pi, "latest_charge")))
			e.Set("type", "invalid_request_error")
			stripeError(w, 400, e)
			return
		}
		f.refunded[piID] += amount
		re := dal.NewRecord()
		re.Set("id", f.nextID("re"))
		re.Set("object", "refund")
		re.Set("amount", amount)
		re.Set("charge", str(pi, "latest_charge"))
		re.Set("created", fakeCreated)
		re.Set("currency", str(pi, "currency"))
		re.Set("payment_intent", piID)
		re.Set("status", "succeeded")
		writeJSON(w, re)
	case r.Method == http.MethodPost && path == "/v1/customers":
		c := dal.NewRecord()
		id := f.nextID("cus")
		c.Set("id", id)
		c.Set("object", "customer")
		addr, _ := params.Get("address")
		c.Set("address", addr)
		c.Set("created", fakeCreated)
		for _, k := range []string{"email", "name", "phone"} {
			v, ok := params.Get(k)
			if !ok || v == "" {
				v = nil
			}
			c.Set(k, v)
		}
		c.Set("livemode", false)
		c.Set("metadata", dal.NewRecord())
		f.customers[id] = c
		writeJSON(w, c)
	case segs[0] == "customers" && len(segs) >= 2:
		c := f.customers[segs[1]]
		if c == nil {
			notFound(w, "customer", segs[1])
			return
		}
		switch {
		case r.Method == http.MethodDelete:
			delete(f.customers, segs[1])
			d := dal.NewRecord()
			d.Set("id", segs[1])
			d.Set("object", "customer")
			d.Set("deleted", true)
			writeJSON(w, d)
		case len(segs) == 3 && segs[2] == "payment_methods":
			l := dal.NewRecord()
			l.Set("object", "list")
			l.Set("data", []any{paymentMethod("pm_card_visa")})
			l.Set("has_more", false)
			l.Set("url", "/v1/customers/"+segs[1]+"/payment_methods")
			writeJSON(w, l)
		default:
			for _, k := range params.Keys() {
				v, _ := params.Get(k)
				c.Set(k, v)
			}
			writeJSON(w, c)
		}
	case segs[0] == "payment_methods" && len(segs) >= 2:
		pm := paymentMethod(segs[1])
		writeJSON(w, pm)
	case r.Method == http.MethodPost && path == "/v1/setup_intents":
		si := dal.NewRecord()
		si.Set("id", f.nextID("seti"))
		si.Set("object", "setup_intent")
		si.Set("customer", str(params, "customer"))
		types, ok := params.Get("payment_method_types")
		if !ok {
			types = []any{"card"}
		}
		si.Set("payment_method_types", types)
		si.Set("status", "requires_payment_method")
		writeJSON(w, si)
	default:
		e := dal.NewRecord()
		e.Set("message", "Unrecognized request URL ("+r.Method+": "+path+").")
		e.Set("type", "invalid_request_error")
		stripeError(w, 404, e)
	}
}

var _ = json.Marshal
var _ = sort.Strings
