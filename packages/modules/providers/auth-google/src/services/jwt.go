package services

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// verifyIDToken mirrors jsonwebtoken's verify(token, getSigningKey,
// { algorithms: ["RS256"], audience, issuer }) with the error messages of
// JsonWebTokenError / TokenExpiredError / NotBeforeError.
func verifyIDToken(ctx context.Context, token any, defined bool, keys *jwksClient, audience string, issuers []string) (*dal.Record, error) {
	if !truthy(token, defined) {
		return nil, errors.New("jwt must be provided")
	}
	jwtString, ok := token.(string)
	if !ok {
		return nil, errors.New("jwt must be a string")
	}
	parts := strings.Split(jwtString, ".")
	if len(parts) != 3 {
		return nil, errors.New("jwt malformed")
	}
	header, payload, err := decodeJWS(jwtString)
	if err != nil {
		return nil, errors.New("invalid token")
	}

	// getSigningKey_: the key of the header's kid from Google's JWKS
	kid, kidOK := header.Get("kid")
	if !truthy(kid, kidOK) {
		return nil, errors.New("error in secret or public key callback: ID token is missing 'kid' header")
	}
	key, err := keys.getSigningKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("error in secret or public key callback: %s", err.Error())
	}

	if strings.TrimSpace(parts[2]) == "" {
		return nil, errors.New("jwt signature is required")
	}
	alg, _ := header.Get("alg")
	if alg != "RS256" {
		return nil, errors.New("invalid algorithm")
	}
	sig, err := decodeBase64Lenient(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err != nil || rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig) != nil {
		return nil, errors.New("invalid signature")
	}

	claims, ok := payload.(*dal.Record)
	if !ok {
		// a non-object payload has none of the checked claims
		claims = dal.NewRecord()
	}
	clock := math.Floor(float64(time.Now().UnixMilli()) / 1000)
	if nbf, ok := claims.Get("nbf"); ok {
		n, isNum := nbf.(float64)
		if !isNum {
			return nil, errors.New("invalid nbf value")
		}
		if n > clock {
			return nil, errors.New("jwt not active")
		}
	}
	if exp, ok := claims.Get("exp"); ok {
		n, isNum := exp.(float64)
		if !isNum {
			return nil, errors.New("invalid exp value")
		}
		if clock >= n {
			return nil, errors.New("jwt expired")
		}
	}
	aud, audOK := claims.Get("aud")
	targets := []any{aud}
	if list, isList := aud.([]any); isList {
		targets = list
	} else if !audOK {
		targets = []any{nil}
	}
	matched := false
	for _, t := range targets {
		if s, isStr := t.(string); isStr && s == audience {
			matched = true
		}
	}
	if !matched {
		return nil, fmt.Errorf("jwt audience invalid. expected: %s", audience)
	}
	iss, _ := claims.Get("iss")
	validIssuer := false
	for _, i := range issuers {
		if s, isStr := iss.(string); isStr && s == i {
			validIssuer = true
		}
	}
	if !validIssuer {
		return nil, fmt.Errorf("jwt issuer invalid. expected: %s", strings.Join(issuers, ","))
	}
	if !ok {
		return nil, errors.New("Invalid id_token")
	}
	return claims, nil
}

var jwsRe = regexp.MustCompile(`^[a-zA-Z0-9\-_]+?\.[a-zA-Z0-9\-_]+?\.([a-zA-Z0-9\-_]+)?$`)

// decodeBase64Lenient mirrors Buffer.from(s, "base64") for base64url input.
func decodeBase64Lenient(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	s = strings.NewReplacer("+", "-", "/", "_").Replace(s)
	return base64.RawURLEncoding.DecodeString(s)
}

// decodeJWS mirrors jws.decode(token, { complete: true }) as used by
// jsonwebtoken: the header must be a JSON object, the payload is parsed
// when it is JSON.
func decodeJWS(token string) (*dal.Record, any, error) {
	if !jwsRe.MatchString(token) {
		return nil, nil, errors.New("invalid")
	}
	parts := strings.Split(token, ".")
	rawHeader, err := decodeBase64Lenient(parts[0])
	if err != nil {
		return nil, nil, err
	}
	h, err := dal.DecodeOrderedJSON(rawHeader)
	header, ok := h.(*dal.Record)
	if err != nil || !ok {
		return nil, nil, errors.New("invalid header")
	}
	rawPayload, err := decodeBase64Lenient(parts[1])
	if err != nil {
		return nil, nil, err
	}
	var payload any = string(rawPayload)
	if v, err := dal.DecodeOrderedJSON(rawPayload); err == nil {
		if r, isObj := v.(*dal.Record); isObj {
			payload = r
		}
	}
	return header, payload, nil
}

// jwksClient mirrors jwks-rsa's client with { cache: true, rateLimit: true }:
// signing keys are memoized per kid for 10 minutes, at most 10 JWKS lookups
// per minute.
type jwksClient struct {
	uri string

	mu       sync.Mutex
	cache    map[string]cachedKey
	tokens   float64
	lastFill time.Time
}

type cachedKey struct {
	key     *rsa.PublicKey
	expires time.Time
}

const (
	jwksCacheMaxAge        = 10 * time.Minute
	jwksRequestsPerMinute  = 10
	jwksCacheMaxEntries    = 5
	jwksErrNoKeys          = "The JWKS endpoint did not contain any keys"
	jwksErrNoSigningKeys   = "The JWKS endpoint did not contain any signing keys"
	jwksErrTooManyRequests = "Too many requests to the JWKS endpoint"
)

func newJwksClient(uri string) *jwksClient {
	return &jwksClient{uri: uri, cache: map[string]cachedKey{}, tokens: jwksRequestsPerMinute, lastFill: time.Now()}
}

func (c *jwksClient) getSigningKey(ctx context.Context, kid any) (*rsa.PublicKey, error) {
	kidString := jsString(kid, true)
	c.mu.Lock()
	if k, ok := c.cache[kidString]; ok && time.Now().Before(k.expires) {
		c.mu.Unlock()
		return k.key, nil
	}
	now := time.Now()
	c.tokens = math.Min(jwksRequestsPerMinute, c.tokens+now.Sub(c.lastFill).Minutes()*jwksRequestsPerMinute)
	c.lastFill = now
	if c.tokens < 1 {
		c.mu.Unlock()
		return nil, errors.New(jwksErrTooManyRequests)
	}
	c.tokens--
	c.mu.Unlock()

	key, err := c.fetchSigningKey(ctx, kid)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if len(c.cache) >= jwksCacheMaxEntries {
		c.cache = map[string]cachedKey{}
	}
	c.cache[kidString] = cachedKey{key: key, expires: time.Now().Add(jwksCacheMaxAge)}
	c.mu.Unlock()
	return key, nil
}

type jwk struct {
	Kid any     `json:"kid"`
	Kty string  `json:"kty"`
	Use *string `json:"use"`
	N   string  `json:"n"`
	E   string  `json:"e"`
}

func (c *jwksClient) fetchSigningKey(ctx context.Context, kid any) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.uri, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(strings.TrimPrefix(res.Status, fmt.Sprint(res.StatusCode)))
		if msg == "" {
			msg = fmt.Sprintf("Http Error %d", res.StatusCode)
		}
		return nil, errors.New(msg)
	}
	var body struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if len(body.Keys) == 0 {
		return nil, errors.New(jwksErrNoKeys)
	}
	type signingKey struct {
		kid string
		key *rsa.PublicKey
	}
	var keys []signingKey
	for _, k := range body.Keys {
		if (k.Use != nil && *k.Use != "sig") || k.Kty != "RSA" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.N, "="))
		e, err2 := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.E, "="))
		if err1 != nil || err2 != nil || len(n) == 0 || len(e) == 0 {
			continue
		}
		kidString, _ := k.Kid.(string)
		keys = append(keys, signingKey{kid: kidString, key: &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}})
	}
	if len(keys) == 0 {
		return nil, errors.New(jwksErrNoSigningKeys)
	}
	kidDefined := kid != nil
	if !kidDefined && len(keys) > 1 {
		return nil, errors.New("No KID specified and JWKS endpoint returned more than 1 key")
	}
	for _, k := range keys {
		if !kidDefined || (k.kid != "" && k.kid == kid) {
			return k.key, nil
		}
	}
	return nil, fmt.Errorf("Unable to find a signing key that matches '%s'", jsString(kid, true))
}
