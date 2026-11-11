package core

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	mrand "math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// The API location of the Stripe client, like stripe-node's DEFAULT_HOST,
// DEFAULT_PORT and default protocol. Tests point them at a local fake.
var (
	DefaultHost     = "api.stripe.com"
	DefaultPort     = "443"
	DefaultProtocol = "https"
	// HTTPClient sends the API requests.
	HTTPClient = &http.Client{Timeout: 80 * time.Second}
)

const (
	defaultBasePath = "/v1/"
	// stripe-node's maxNetworkRetries default on Node.
	defaultMaxNetworkRetries     = 2
	initialNetworkRetryDelaySec  = 0.5
	maxNetworkRetryDelaySec      = 5.0
	webhookDefaultToleranceSecs  = 300
	connectionErrorMessagePrefix = "An error occurred with our connection to Stripe."
)

// StripeError mirrors stripe-node's StripeError and its subclasses (Type
// is the class name, e.g. StripeCardError).
type StripeError struct {
	Type    string
	Message string
	// Raw is the error object of the response with headers, statusCode
	// and requestId, like error.raw.
	Raw        *dal.Record
	StatusCode int
}

func (e *StripeError) Error() string { return e.Message }

func (e *StripeError) rawGet(k string) any {
	if e.Raw == nil {
		return nil
	}
	v, _ := e.Raw.Get(k)
	return v
}

// Code mirrors error.code.
func (e *StripeError) Code() any { return e.rawGet("code") }

// PaymentIntent mirrors error.payment_intent.
func (e *StripeError) PaymentIntent() *dal.Record {
	r, _ := e.rawGet("payment_intent").(*dal.Record)
	return r
}

// generateV1Error mirrors generateV1Error.
func generateV1Error(raw *dal.Record, statusCode int) *StripeError {
	msg, _ := raw.Get("message")
	m, _ := msg.(string)
	code, _ := raw.Get("code")
	typ, _ := raw.Get("type")
	t := "StripeAPIError"
	switch {
	case statusCode == 429 || (statusCode == 400 && code == "rate_limit"):
		t = "StripeRateLimitError"
	case statusCode == 400 || statusCode == 404:
		if typ == "idempotency_error" {
			t = "StripeIdempotencyError"
		} else {
			t = "StripeInvalidRequestError"
		}
	case statusCode == 401:
		t = "StripeAuthenticationError"
	case statusCode == 402:
		t = "StripeCardError"
	case statusCode == 403:
		t = "StripePermissionError"
	}
	return &StripeError{Type: t, Message: m, Raw: raw, StatusCode: statusCode}
}

// Client mirrors the parts of the stripe-node client the providers use.
type Client struct {
	apiKey     string
	apiVersion string
}

// NewClient mirrors new Stripe(apiKey, { apiVersion }).
func NewClient(apiKey, apiVersion string) *Client {
	return &Client{apiKey: apiKey, apiVersion: apiVersion}
}

// RequestOptions mirrors the request options (idempotencyKey).
type RequestOptions struct {
	IdempotencyKey string
}

func uuid4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// FormatJSNumber formats a number like JavaScript's String(n).
func FormatJSNumber(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if f == 0 {
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e21 || abs < 1e-6 {
		s := strconv.FormatFloat(f, 'e', -1, 64)
		mant, exp, _ := strings.Cut(s, "e")
		if exp[0] == '+' {
			exp = exp[1:]
			return mant + "e+" + strings.TrimLeft(exp, "0")
		}
		return mant + "e-" + strings.TrimLeft(exp[1:], "0")
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func valueToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return FormatJSNumber(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case time.Time:
		return strconv.FormatInt(t.Unix(), 10)
	case fmt.Stringer:
		return t.String()
	}
	return fmt.Sprint(v)
}

// encodeQueryValue mirrors encodeQueryValue (encodeURIComponent, with
// !*()' encoded and brackets left readable).
func encodeQueryValue(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~', c == '[', c == ']':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// StringifyRequestData mirrors stringifyRequestData: nested objects as
// key[child]=value, arrays as key[0]=value, undefined (absent) skipped,
// null as an empty value.
func StringifyRequestData(data *dal.Record) string {
	var pairs []string
	var encode func(key string, value any)
	encode = func(key string, value any) {
		switch t := value.(type) {
		case *dal.Record:
			if t == nil {
				pairs = append(pairs, encodeQueryValue(key)+"=")
				return
			}
			for _, k := range t.Keys() {
				v, _ := t.Get(k)
				encode(key+"["+k+"]", v)
			}
		case []any:
			for i, v := range t {
				encode(key+"["+strconv.Itoa(i)+"]", v)
			}
		case []string:
			for i, v := range t {
				encode(key+"["+strconv.Itoa(i)+"]", v)
			}
		default:
			pairs = append(pairs, encodeQueryValue(key)+"="+encodeQueryValue(valueToString(value)))
		}
	}
	if data != nil {
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			encode(k, v)
		}
	}
	return strings.Join(pairs, "&")
}

func sleepTime(numRetries int) time.Duration {
	s := math.Min(initialNetworkRetryDelaySec*math.Pow(2, float64(numRetries-1)), maxNetworkRetryDelaySec)
	s *= 0.5 * (1 + mrand.Float64())
	s = math.Max(initialNetworkRetryDelaySec, s)
	return time.Duration(s * float64(time.Second))
}

func baseURL() string {
	return DefaultProtocol + "://" + DefaultHost + ":" + DefaultPort
}

// Request mirrors StripeResource requests: GET and DELETE send the params
// in the query string, POST form-encoded; every POST carries an
// idempotency key (a generated one unless given). Connection errors, 409
// and 5xx responses are retried (maxNetworkRetries 2).
func (c *Client) Request(ctx context.Context, method, path string, params *dal.Record, opts RequestOptions) (*dal.Record, error) {
	u := baseURL() + defaultBasePath + strings.TrimPrefix(path, "/")
	var body []byte
	encoded := StringifyRequestData(params)
	if method == http.MethodPost {
		body = []byte(encoded)
	} else if encoded != "" {
		u += "?" + encoded
	}
	idempotencyKey := opts.IdempotencyKey
	if idempotencyKey == "" && method == http.MethodPost {
		idempotencyKey = "stripe-node-retry-" + uuid4()
	}
	numRetries := 0
	for {
		var reader io.Reader
		if method == http.MethodPost {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, reader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "Stripe/v1 NodeBindings/23.1.0")
		req.Header.Set("Stripe-Version", c.apiVersion)
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		res, err := HTTPClient.Do(req)
		if err != nil {
			if numRetries < defaultMaxNetworkRetries {
				numRetries++
				time.Sleep(sleepTime(numRetries))
				continue
			}
			msg := connectionErrorMessagePrefix
			if numRetries > 0 {
				msg += fmt.Sprintf(" Request was retried %d times.", numRetries)
			}
			raw := dal.NewRecord()
			raw.Set("message", msg)
			raw.Set("detail", err.Error())
			return nil, &StripeError{Type: "StripeConnectionError", Message: msg, Raw: raw}
		}
		data, rerr := io.ReadAll(res.Body)
		res.Body.Close()
		if rerr == nil && numRetries < defaultMaxNetworkRetries && shouldRetry(res) {
			numRetries++
			time.Sleep(sleepTime(numRetries))
			continue
		}
		v, perr := dal.DecodeOrderedJSON(data)
		if rerr != nil || perr != nil {
			raw := dal.NewRecord()
			raw.Set("message", "Invalid JSON received from the Stripe API")
			return nil, &StripeError{Type: "StripeAPIError", Message: "Invalid JSON received from the Stripe API", Raw: raw}
		}
		rec, _ := v.(*dal.Record)
		if rec == nil {
			rec = dal.NewRecord()
		}
		if e, ok := rec.Get("error"); ok {
			raw, _ := e.(*dal.Record)
			if raw == nil {
				raw = dal.NewRecord()
			}
			headers := dal.NewRecord()
			for k := range res.Header {
				headers.Set(strings.ToLower(k), res.Header.Get(k))
			}
			raw.Set("headers", headers)
			raw.Set("statusCode", float64(res.StatusCode))
			raw.Set("requestId", res.Header.Get("Request-Id"))
			return nil, generateV1Error(raw, res.StatusCode)
		}
		return rec, nil
	}
}

func shouldRetry(res *http.Response) bool {
	switch res.Header.Get("Stripe-Should-Retry") {
	case "false":
		return false
	case "true":
		return true
	}
	return res.StatusCode == 409 || res.StatusCode >= 500
}

// SignatureVerificationError mirrors StripeSignatureVerificationError.
func signatureError(msg string) *StripeError {
	raw := dal.NewRecord()
	raw.Set("message", msg)
	return &StripeError{Type: "StripeSignatureVerificationError", Message: msg, Raw: raw}
}

var blankSecretRe = regexp.MustCompile(`^[ \t\r\n\f\v]*$`)

const signatureDocs = "\nLearn more about webhook signing and explore webhook integration examples for various frameworks at https://docs.stripe.com/webhooks/signature"

// ConstructEvent mirrors stripe.webhooks.constructEvent(payload, header,
// secret) with the default tolerance (300s).
func ConstructEvent(payload []byte, header string, secret string) (*dal.Record, error) {
	if len(payload) == 0 {
		return nil, signatureError("No webhook payload was provided.")
	}
	if header == "" {
		return nil, signatureError("No stripe-signature header value was provided.")
	}
	timestamp := int64(-1)
	var signatures []string
	for _, item := range strings.Split(header, ",") {
		kv := strings.Split(item, "=")
		val := ""
		if len(kv) > 1 {
			val = kv[1]
		}
		if kv[0] == "t" {
			timestamp = parseIntJS(val)
		}
		if kv[0] == "v1" {
			signatures = append(signatures, val)
		}
	}
	if timestamp == -1 {
		return nil, signatureError("Unable to extract timestamp and signatures from header")
	}
	if len(signatures) == 0 {
		return nil, signatureError("No signatures found with expected scheme")
	}
	if secret == "" || blankSecretRe.MatchString(secret) {
		return nil, signatureError("No webhook secret value was provided. It should start with `whsec_`")
	}
	whitespace := ""
	if regexp.MustCompile(`\s`).MatchString(secret) {
		whitespace = "\n\nNote: The provided signing secret contains whitespace. This often indicates an extra newline or space is in the value"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "." + string(payload)))
	expected := hex.EncodeToString(mac.Sum(nil))
	found := false
	for _, s := range signatures {
		if len(s) == len(expected) && subtle.ConstantTimeCompare([]byte(s), []byte(expected)) == 1 {
			found = true
		}
	}
	if !found {
		return nil, signatureError("No signatures found matching the expected signature for payload. Are you passing the raw request body you received from Stripe? \n" +
			" If a webhook request is being forwarded by a third-party tool, ensure that the exact request body, including JSON formatting and new line style, is preserved.\n" +
			signatureDocs + "\n" + whitespace)
	}
	if time.Now().Unix()-timestamp > webhookDefaultToleranceSecs {
		return nil, signatureError("Timestamp outside the tolerance zone")
	}
	v, err := dal.DecodeOrderedJSON(payload)
	if err != nil {
		return nil, err
	}
	event, _ := v.(*dal.Record)
	if event != nil {
		if o, _ := event.Get("object"); o == "v2.core.event" {
			return nil, fmt.Errorf("You passed a thin event notification to a function that expects a webhook. Use the corresponding EventNotification method instead.")
		}
	}
	return event, nil
}

// parseIntJS mirrors parseInt(s, 10) (NaN as -1 never matches a header).
func parseIntJS(s string) int64 {
	s = strings.TrimSpace(s)
	end := 0
	if end < len(s) && (s[end] == '-' || s[end] == '+') {
		end++
	}
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	n, err := strconv.ParseInt(s[:end], 10, 64)
	if err != nil {
		return math.MinInt64
	}
	return n
}

// GenerateTestHeaderString mirrors webhooks.generateTestHeaderString.
func GenerateTestHeaderString(payload []byte, secret string, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "." + string(payload)))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}

var _ = url.QueryEscape
