// Package client mirrors the parts of the Adyen Node API library
// (@adyen/api-library) that @medusajs/payment-adyen uses: the Checkout API
// requests with the ObjectSerializer's key ordering, the HTTP errors and the
// notification HMAC validator.
package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

const (
	checkoutTestBase = "https://checkout-test.adyen.com/v72"
	libName          = "adyen-node-api-library"
	libVersion       = "32.2.0"
)

var (
	// HTTPClient sends the API requests.
	HTTPClient = &http.Client{Timeout: 60 * time.Second}
	// EndpointOverride replaces the scheme and host of the API URLs (tests
	// point it at a local fake).
	EndpointOverride = ""
)

// Model is an ObjectSerializer model (attributeTypeMap).
type Model struct {
	Discriminator string
	Mapping       map[string]string
	Attrs         []Attr
	// NoAttributes marks the oneOf classes (no getAttributeTypeMap): data
	// that their discriminator does not map is copied as is.
	NoAttributes bool
}

// Attr is one attributeTypeMap entry.
type Attr struct {
	Name     string
	BaseName string
	Type     string
	Format   string
}

// Config mirrors the Client configuration.
type Config struct {
	APIKey                string
	Environment           string
	LiveEndpointURLPrefix string
}

// Client mirrors CheckoutAPI over a Client.
type Client struct {
	config Config
}

// New mirrors new CheckoutAPI(new Client(config)).
func New(c Config) *Client { return &Client{config: c} }

// HTTPClientException mirrors HttpClientException.
type HTTPClientException struct {
	Message      string
	StatusCode   int
	ErrorCode    string
	ResponseBody string
}

func (e *HTTPClientException) Error() string { return e.Message }

// RequestError mirrors the plain Error thrown for request errors
// ({ errors: [...] } bodies): it has no status code.
type RequestError struct{ Body string }

func (e *RequestError) Error() string { return e.Body }

// baseURL mirrors Service.createBaseUrl for the checkout endpoints.
func (c *Client) baseURL() string {
	u := checkoutTestBase
	if c.config.Environment == "LIVE" {
		u = strings.Replace(u, "https://checkout-test.adyen.com/", "https://"+c.config.LiveEndpointURLPrefix+"-checkout-live.adyenpayments.com/checkout/", 1)
	}
	if EndpointOverride != "" {
		u = regexp.MustCompile(`^https://[^/]+`).ReplaceAllString(u, strings.TrimSuffix(EndpointOverride, "/"))
	}
	return u
}

// RequestOptions mirrors IRequest.Options.
type RequestOptions struct {
	IdempotencyKey string
	Params         [][2]string
}

func (c *Client) do(ctx context.Context, method, path string, body any, opts RequestOptions) (any, error) {
	endpoint := c.baseURL() + path
	if len(opts.Params) > 0 {
		q := url.Values{}
		var parts []string
		for _, p := range opts.Params {
			q.Set(p[0], p[1])
			parts = append(parts, url.QueryEscape(p[0])+"="+url.QueryEscape(p[1]))
		}
		endpoint += "?" + strings.Join(parts, "&")
	}
	var reader io.Reader
	if body != nil {
		raw, err := dal.MarshalValue(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	} else if method != http.MethodGet && method != http.MethodDelete {
		reader = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if opts.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", opts.IdempotencyKey)
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept-Charset", "utf-8")
	req.Header.Set("User-Agent", libName+"/"+libVersion)
	req.Header.Set("adyen-library-name", libName)
	req.Header.Set("adyen-library-version", libVersion)
	res, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	statusText := http.StatusText(res.StatusCode)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		v, perr := dal.DecodeOrderedJSON(raw)
		if perr != nil {
			return nil, &HTTPClientException{
				Message:      fmt.Sprintf("HTTP Exception: %d. Error %s while parsing response: %s", res.StatusCode, perr.Error(), raw),
				StatusCode:   res.StatusCode,
				ResponseBody: string(raw),
			}
		}
		r, _ := v.(*dal.Record)
		if r != nil && r.Has("status") {
			st, _ := r.Get("status")
			msg, _ := r.Get("message")
			code, _ := r.Get("errorCode")
			codeStr, _ := code.(string)
			status := res.StatusCode
			if f, ok := st.(float64); ok {
				status = int(f)
			}
			return nil, &HTTPClientException{
				Message:      fmt.Sprintf("HTTP Exception: %v. %s: %v", jsValue(st), statusText, jsValue(msg)),
				StatusCode:   status,
				ErrorCode:    codeStr,
				ResponseBody: string(raw),
			}
		}
		if r != nil && r.Has("errors") {
			return nil, &RequestError{Body: string(raw)}
		}
		return nil, &HTTPClientException{Message: fmt.Sprintf("HTTP Exception: %d. %s", res.StatusCode, statusText), StatusCode: res.StatusCode, ResponseBody: string(raw)}
	}
	if len(raw) == 0 {
		return "", nil
	}
	v, err := dal.DecodeOrderedJSON(raw)
	if err != nil {
		return string(raw), nil
	}
	return v, nil
}

func jsValue(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case float64:
		return fmt.Sprint(t)
	}
	return fmt.Sprint(v)
}

func (c *Client) call(ctx context.Context, method, path string, request any, requestType, responseType string, opts RequestOptions) (*dal.Record, error) {
	var body any
	if requestType != "" {
		body = Serialize(request, requestType, "")
	}
	res, err := c.do(ctx, method, path, body, opts)
	if err != nil {
		return nil, err
	}
	if responseType == "" {
		return nil, nil
	}
	out, _ := Deserialize(res, responseType, "").(*dal.Record)
	if out == nil {
		out = dal.NewRecord()
	}
	return out, nil
}

func esc(s string) string { return url.PathEscape(s) }

// PaymentMethods mirrors PaymentsApi.paymentMethods.
func (c *Client) PaymentMethods(ctx context.Context, req *dal.Record) (*dal.Record, error) {
	return c.call(ctx, http.MethodPost, "/paymentMethods", req, "PaymentMethodsRequest", "PaymentMethodsResponse", RequestOptions{})
}

// Payments mirrors PaymentsApi.payments.
func (c *Client) Payments(ctx context.Context, req *dal.Record, opts RequestOptions) (*dal.Record, error) {
	return c.call(ctx, http.MethodPost, "/payments", req, "PaymentRequest", "PaymentResponse", opts)
}

// PaymentsDetails mirrors PaymentsApi.paymentsDetails.
func (c *Client) PaymentsDetails(ctx context.Context, req *dal.Record, opts RequestOptions) (*dal.Record, error) {
	return c.call(ctx, http.MethodPost, "/payments/details", req, "PaymentDetailsRequest", "PaymentDetailsResponse", opts)
}

// CaptureAuthorisedPayment mirrors ModificationsApi.captureAuthorisedPayment.
func (c *Client) CaptureAuthorisedPayment(ctx context.Context, psp string, req *dal.Record, opts RequestOptions) (*dal.Record, error) {
	return c.call(ctx, http.MethodPost, "/payments/"+esc(psp)+"/captures", req, "PaymentCaptureRequest", "PaymentCaptureResponse", opts)
}

// RefundCapturedPayment mirrors ModificationsApi.refundCapturedPayment.
func (c *Client) RefundCapturedPayment(ctx context.Context, psp string, req *dal.Record, opts RequestOptions) (*dal.Record, error) {
	return c.call(ctx, http.MethodPost, "/payments/"+esc(psp)+"/refunds", req, "PaymentRefundRequest", "PaymentRefundResponse", opts)
}

// RefundOrCancelPayment mirrors ModificationsApi.refundOrCancelPayment.
func (c *Client) RefundOrCancelPayment(ctx context.Context, psp string, req *dal.Record, opts RequestOptions) (*dal.Record, error) {
	return c.call(ctx, http.MethodPost, "/payments/"+esc(psp)+"/reversals", req, "PaymentReversalRequest", "PaymentReversalResponse", opts)
}

// CancelAuthorisedPaymentByPspReference mirrors
// ModificationsApi.cancelAuthorisedPaymentByPspReference.
func (c *Client) CancelAuthorisedPaymentByPspReference(ctx context.Context, psp string, req *dal.Record, opts RequestOptions) (*dal.Record, error) {
	return c.call(ctx, http.MethodPost, "/payments/"+esc(psp)+"/cancels", req, "PaymentCancelRequest", "PaymentCancelResponse", opts)
}

// StoredPaymentMethods mirrors RecurringApi.storedPaymentMethods.
func (c *Client) StoredPaymentMethods(ctx context.Context, req *dal.Record, opts RequestOptions) (*dal.Record, error) {
	return c.call(ctx, http.MethodPost, "/storedPaymentMethods", req, "StoredPaymentMethodRequest", "StoredPaymentMethodResource", opts)
}

// GetTokensForStoredPaymentDetails mirrors
// RecurringApi.getTokensForStoredPaymentDetails.
func (c *Client) GetTokensForStoredPaymentDetails(ctx context.Context, shopperReference, merchantAccount string) (*dal.Record, error) {
	var params [][2]string
	if shopperReference != "" {
		params = append(params, [2]string{"shopperReference", shopperReference})
	}
	if merchantAccount != "" {
		params = append(params, [2]string{"merchantAccount", merchantAccount})
	}
	return c.call(ctx, http.MethodGet, "/storedPaymentMethods", nil, "", "ListStoredPaymentMethodsResponse", RequestOptions{Params: params})
}

// DeleteTokenForStoredPaymentDetails mirrors
// RecurringApi.deleteTokenForStoredPaymentDetails.
func (c *Client) DeleteTokenForStoredPaymentDetails(ctx context.Context, id, shopperReference, merchantAccount string) error {
	var params [][2]string
	if shopperReference != "" {
		params = append(params, [2]string{"shopperReference", shopperReference})
	}
	if merchantAccount != "" {
		params = append(params, [2]string{"merchantAccount", merchantAccount})
	}
	_, err := c.do(ctx, http.MethodDelete, "/storedPaymentMethods/"+esc(id), nil, RequestOptions{Params: params})
	return err
}

// HmacSignatureKey mirrors ApiConstants.HMAC_SIGNATURE.
const HmacSignatureKey = "hmacSignature"

func str(v any) string {
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
		return fmt.Sprint(t)
	}
	return fmt.Sprint(v)
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// DataToSign mirrors HmacValidator.getDataToSign for a notification item.
func DataToSign(item *dal.Record) string {
	amount, _ := get(item, "amount").(*dal.Record)
	parts := []string{
		str(get(item, "pspReference")),
		str(get(item, "originalReference")),
		str(get(item, "merchantAccountCode")),
		str(get(item, "merchantReference")),
		str(get(amount, "value")),
		str(get(amount, "currency")),
		str(get(item, "eventCode")),
		str(get(item, "success")),
	}
	return strings.Join(parts, ":")
}

// CalculateHmac mirrors HmacValidator.calculateHmac.
func CalculateHmac(data string, key string) (string, error) {
	raw, err := hex.DecodeString(key)
	if err != nil {
		// Buffer.from(key, "hex") stops at the first invalid pair
		raw = []byte{}
	}
	mac := hmac.New(sha256.New, raw)
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// ValidateHMAC mirrors HmacValidator.validateHMAC.
func ValidateHMAC(item *dal.Record, key string) (bool, error) {
	additional, _ := get(item, "additionalData").(*dal.Record)
	sign, _ := get(additional, HmacSignatureKey).(string)
	if sign == "" {
		return false, fmt.Errorf("Missing %s", HmacSignatureKey)
	}
	expected, _ := CalculateHmac(DataToSign(item), key)
	e, _ := base64.StdEncoding.DecodeString(expected)
	r, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false, nil
	}
	return len(e) == len(r) && hmac.Equal(e, r), nil
}

var _ = common.ParseJSDate
