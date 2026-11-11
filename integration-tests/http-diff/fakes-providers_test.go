package httpdiff

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash/crc32"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Fake third-party services for the provider scenarios. A scenario's
// environment enables them (both servers run with it):
//
//	MEDUSA_TEST_FAKE_PORT=12451      the fake server's port (localhost)
//	MEDUSA_TEST_FAKE_HOSTS=github.com=/github,...
//	                                 outbound requests to these hosts go to
//	                                 the fake under the given path prefix
//	                                 (Go: http.DefaultTransport; upstream:
//	                                 fake-hosts.js, required by the
//	                                 reference application's config)
//	MEDUSA_TEST_S3_UPSTREAM=http://localhost:12459
//	MEDUSA_TEST_S3_BUCKET=medusa-test
//	                                 requests for the bucket (path style)
//	                                 are proxied to an S3 server (MinIO)
//	MEDUSA_TEST_MODULES=[...]        modules added to the configuration, in
//	                                 the medusa-config format
//	                                 ({ key, resolve, options: { providers } })
//
// The fake records the requests it receives; a step requesting
// "external": "http://localhost:<port>/__requests" reads (and clears) them.
// It answers like GitHub (OAuth token exchange, /user), Google (token
// exchange returning an RS256 id_token, JWKS) depending on the "code".

type fakeServer struct {
	mu       sync.Mutex
	requests []any
	key      *rsa.PrivateKey
	bucket   string
	s3       *httputil.ReverseProxy
}

// recordedRequestHeaders are the request headers the fake records (S3
// signatures and dates differ between runs and are left out).
var recordedRequestHeaders = []string{
	"accept", "authorization", "content-type", "cache-control",
	"x-amz-acl", "x-amz-meta-original-filename", "x-amz-checksum-crc32", "x-amz-sdk-checksum-algorithm",
	"x-amz-checksum-mode", "x-amz-checksum-algorithm", "x-amz-content-sha256",
}

func (f *fakeServer) record(r *http.Request, body []byte, s3 bool) {
	rec := dal.NewRecord()
	rec.Set("method", r.Method)
	rec.Set("path", r.URL.EscapedPath())
	rec.Set("query", r.URL.RawQuery)
	headers := dal.NewRecord()
	for _, h := range recordedRequestHeaders {
		if s3 && h == "authorization" {
			continue
		}
		v := r.Header.Get(h)
		if v == "" {
			continue
		}
		// body checksums: checked here (the bodies hold generated keys)
		switch {
		case h == "x-amz-content-sha256" && v == sha256Hex(body):
			v = "<sha256 of the body>"
		case h == "x-amz-checksum-crc32" && v == crc32Base64(body):
			v = "<crc32 of the body>"
		}
		headers.Set(h, v)
	}
	rec.Set("headers", headers)
	if len(body) > 0 {
		if v, err := dal.DecodeOrderedJSON(body); err == nil {
			rec.Set("body", v)
		} else {
			rec.Set("body", string(body))
		}
	}
	f.mu.Lock()
	f.requests = append(f.requests, rec)
	f.mu.Unlock()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func crc32Base64(b []byte) string {
	var out [4]byte
	binary.BigEndian.PutUint32(out[:], crc32.ChecksumIEEE(b))
	return base64.StdEncoding.EncodeToString(out[:])
}

func providerWriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// idToken signs an RS256 JWT with the fake's key.
func (f *fakeServer) idToken(header, claims map[string]any, corrupt bool) string {
	h, _ := json.Marshal(header)
	c, _ := json.Marshal(claims)
	signing := b64url(h) + "." + b64url(c)
	sum := sha256.Sum256([]byte(signing))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, sum[:])
	if corrupt {
		sig[0] ^= 0xff
	}
	return signing + "." + b64url(sig)
}

func (f *fakeServer) googleToken(w http.ResponseWriter, q url.Values) {
	code := q.Get("code")
	now := time.Now().Unix()
	header := map[string]any{"alg": "RS256", "kid": "fake-key", "typ": "JWT"}
	claims := map[string]any{
		"iss": "https://accounts.google.com", "azp": q.Get("client_id"), "aud": q.Get("client_id"),
		"sub": "google-sub-1", "email": "gopher@example.test", "email_verified": true, "name": "Go Pher",
		"picture": "https://example.test/gopher.png", "given_name": "Go", "family_name": "Pher", "iat": now, "exp": now + 3600,
	}
	corrupt := false
	switch code {
	case "bad":
		providerWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_grant", "error_description": "Bad Request"})
		return
	case "noidtoken":
		providerWriteJSON(w, http.StatusOK, map[string]any{"access_token": "ya29.token", "expires_in": 3599})
		return
	case "unverified":
		claims["email_verified"] = false
	case "wrongaud":
		claims["aud"] = "another-client"
	case "expired":
		claims["exp"] = now - 3600
	case "nokid":
		delete(header, "kid")
	case "unknownkid":
		header["kid"] = "unknown-key"
	case "badiss":
		claims["iss"] = "https://issuer.example.test"
	case "nosub":
		delete(claims, "sub")
	case "badsig":
		corrupt = true
	case "hs256":
		header["alg"] = "HS256"
	case "other":
		claims["sub"] = "google-sub-2"
		claims["email"] = "other@example.test"
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"access_token": "ya29.token", "expires_in": 3599, "scope": "openid email profile", "token_type": "Bearer",
		"id_token": f.idToken(header, claims, corrupt),
	})
}

func (f *fakeServer) githubToken(w http.ResponseWriter, q url.Values) {
	code := q.Get("code")
	switch code {
	case "bad":
		providerWriteJSON(w, http.StatusUnauthorized, map[string]any{"error": "bad_verification_code"})
	case "notoken":
		providerWriteJSON(w, http.StatusOK, map[string]any{"error": "bad_verification_code", "error_description": "The code passed is incorrect or expired."})
	case "noexpiry":
		providerWriteJSON(w, http.StatusOK, map[string]any{"access_token": "gho_" + code, "token_type": "bearer", "scope": ""})
	case "notjson":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("access_token=gho_x&token_type=bearer"))
	default:
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"access_token": "gho_" + code, "expires_in": 28800, "refresh_token": "ghr_" + code,
			"refresh_token_expires_in": 15811200, "token_type": "bearer", "scope": "",
		})
	}
}

func (f *fakeServer) githubUser(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	switch token {
	case "gho_baduser":
		providerWriteJSON(w, http.StatusUnauthorized, map[string]any{"message": "Bad credentials", "documentation_url": "https://docs.github.com/rest"})
	case "gho_other":
		providerWriteJSON(w, http.StatusOK, map[string]any{"login": "other", "id": 2002, "url": "https://api.github.com/users/other", "avatar_url": "https://example.test/other.png", "email": nil, "name": nil, "company": "Example"})
	default:
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"login": "octo", "id": 1001, "url": "https://api.github.com/users/octo", "avatar_url": "https://example.test/octo.png",
			"email": "octo@example.test", "name": "Octo Cat", "company": nil, "two_factor_authentication": true,
		})
	}
}

func (f *fakeServer) jwks(w http.ResponseWriter) {
	pub := f.key.PublicKey
	providerWriteJSON(w, http.StatusOK, map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "fake-key", "use": "sig", "alg": "RS256",
		"n": b64url(pub.N.Bytes()), "e": b64url(big.NewInt(int64(pub.E)).Bytes()),
	}}})
}

func (f *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	path := r.URL.Path
	if path == "/__requests" {
		f.mu.Lock()
		reqs := f.requests
		f.requests = nil
		f.mu.Unlock()
		if reqs == nil {
			reqs = []any{}
		}
		out := dal.NewRecord()
		out.Set("requests", reqs)
		raw, _ := dal.MarshalValue(out)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(raw)
		return
	}
	if f.s3 != nil && (path == "/"+f.bucket || strings.HasPrefix(path, "/"+f.bucket+"/")) {
		// public object reads (file URLs) are not recorded
		if r.Header.Get("Authorization") != "" || r.URL.Query().Has("X-Amz-Signature") || r.Method != http.MethodGet {
			f.record(r, body, true)
		}
		f.s3.ServeHTTP(w, r)
		return
	}
	f.record(r, body, false)
	switch {
	case path == "/github/login/oauth/access_token":
		f.githubToken(w, r.URL.Query())
	case path == "/github-api/user":
		f.githubUser(w, r)
	case path == "/google-oauth/token":
		f.googleToken(w, r.URL.Query())
	case path == "/googleapis/oauth2/v3/certs":
		f.jwks(w)
	default:
		providerWriteJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
	}
}

// startProviderFakes starts the fake services and the outbound host rewriting a
// scenario's environment asks for.
func startProviderFakes(t *testing.T, env map[string]string) {
	t.Helper()
	port := env["MEDUSA_TEST_FAKE_PORT"]
	if port == "" {
		return
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeServer{key: key, bucket: env["MEDUSA_TEST_S3_BUCKET"]}
	if upstream := env["MEDUSA_TEST_S3_UPSTREAM"]; upstream != "" {
		u, err := url.Parse(upstream)
		if err != nil {
			t.Fatal(err)
		}
		f.s3 = httputil.NewSingleHostReverseProxy(u)
	}
	ln, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: f}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	if hosts := env["MEDUSA_TEST_FAKE_HOSTS"]; hosts != "" {
		prev := http.DefaultTransport
		http.DefaultTransport = &hostRewriteTransport{base: prev, target: "localhost:" + port, prefixes: parseFakeHosts(hosts)}
		t.Cleanup(func() { http.DefaultTransport = prev })
	}
}

func parseFakeHosts(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		if host, prefix, ok := strings.Cut(strings.TrimSpace(part), "="); ok {
			out[host] = prefix
		}
	}
	return out
}

// hostRewriteTransport sends the requests for the faked hosts to the fake.
type hostRewriteTransport struct {
	base     http.RoundTripper
	target   string
	prefixes map[string]string
}

func (h *hostRewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	prefix, ok := h.prefixes[r.URL.Host]
	if !ok {
		return h.base.RoundTrip(r)
	}
	r2 := r.Clone(r.Context())
	u := *r.URL
	u.Scheme, u.Host = "http", h.target
	u.Path = prefix + r.URL.Path
	if r.URL.RawPath != "" {
		u.RawPath = prefix + r.URL.RawPath
	}
	r2.URL = &u
	r2.Host = h.target
	return h.base.RoundTrip(r2)
}

// envModules reads MEDUSA_TEST_MODULES: modules in the medusa-config format.
func envModules() []config.ModuleConfig {
	raw := os.Getenv("MEDUSA_TEST_MODULES")
	if raw == "" {
		return nil
	}
	var mods []struct {
		Key     string `json:"key"`
		Resolve string `json:"resolve"`
		Options struct {
			Providers []config.ModuleProviderConfig `json:"providers"`
		} `json:"options"`
	}
	if err := json.Unmarshal([]byte(raw), &mods); err != nil {
		panic(errors.New("MEDUSA_TEST_MODULES: " + err.Error()))
	}
	out := make([]config.ModuleConfig, len(mods))
	for i, m := range mods {
		out[i] = config.ModuleConfig{Key: m.Key, Resolve: m.Resolve, Providers: m.Options.Providers}
	}
	return out
}
