package http

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
)

// upstreamSession is a value written by upstream (express-session 1.19 +
// connect-redis 5.2) after POST /auth/session with the default options.
const upstreamSession = `{"cookie":{"originalMaxAge":36000000,"expires":"2026-11-11T01:25:00.231Z","secure":false,"httpOnly":true,"path":"/","sameSite":false},"auth_context":{"actor_id":"user_01M70Y01SPRY6AWYTJKZK9YEES","actor_type":"user","auth_identity_id":"authid_01M70Y01VANH9W8XESKQM685FJ","auth_provider":"emailpass","mfa_enabled":false,"mfa_challenge_completed_at":null,"app_metadata":{"user_id":"user_01M70Y01SPRY6AWYTJKZK9YEES"},"user_metadata":{},"iat":1794324300,"exp":1794410700}}`

func defaultSessionOptions() *SessionOptions {
	return &SessionOptions{Name: "connect.sid", Secret: "supersecret", Resave: true,
		Cookie: SessionCookieOptions{Path: "/", HttpOnly: true, MaxAge: 36000000}}
}

func TestSessionSerializeLikeExpressSession(t *testing.T) {
	o := defaultSessionOptions()
	s := &Session{ID: "sid", opts: o, Data: dal.NewRecord(), cookie: newSessionCookie(o.Cookie)}
	exp := time.Date(2026, 11, 11, 1, 25, 0, 231e6, time.UTC)
	s.cookie.expires = &exp
	auth := dal.NewRecord()
	auth.Set("actor_id", "user_01M70Y01SPRY6AWYTJKZK9YEES")
	auth.Set("actor_type", "user")
	auth.Set("auth_identity_id", "authid_01M70Y01VANH9W8XESKQM685FJ")
	auth.Set("auth_provider", "emailpass")
	auth.Set("mfa_enabled", false)
	auth.Set("mfa_challenge_completed_at", nil)
	meta := dal.NewRecord()
	meta.Set("user_id", "user_01M70Y01SPRY6AWYTJKZK9YEES")
	auth.Set("app_metadata", meta)
	auth.Set("user_metadata", dal.NewRecord())
	auth.Set("iat", float64(1794324300))
	auth.Set("exp", float64(1794410700))
	s.Set("auth_context", auth)
	if got := string(s.serialize()); got != upstreamSession {
		t.Fatalf("serialize:\n got %s\nwant %s", got, upstreamSession)
	}
}

func TestSessionInflateKeepsStoredCookie(t *testing.T) {
	cases := []string{
		upstreamSession,
		// production cookie, cookieOptions with domain / priority / partitioned
		`{"cookie":{"originalMaxAge":36000000,"partitioned":true,"priority":"high","expires":"2026-11-11T01:25:00.231Z","secure":true,"httpOnly":true,"domain":"example.com","path":"/","sameSite":"lax"},"a":1}`,
		// browser-session cookie (maxAge null), sameSite true
		`{"cookie":{"originalMaxAge":null,"expires":null,"secure":false,"httpOnly":false,"path":"/admin","sameSite":true}}`,
	}
	o := defaultSessionOptions()
	for _, c := range cases {
		s, err := inflateSession("sid", []byte(c), o)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(s.serialize()); got != c {
			t.Fatalf("round trip:\n got %s\nwant %s", got, c)
		}
	}
	if _, err := inflateSession("sid", []byte(`{"a":1}`), o); err == nil {
		t.Fatal("a session without cookie must fail like createSession")
	}
}

func TestSessionCookieTouchAndSerializeOptions(t *testing.T) {
	o := defaultSessionOptions()
	o.Cookie.SameSite = "lax"
	o.Cookie.Secure = true
	c := newSessionCookie(o.Cookie)
	raw, _ := c.data().MarshalJSON()
	want := `{"originalMaxAge":36000000,"expires":"` + dal.FormatJSDate(*c.expires) + `","secure":true,"httpOnly":true,"path":"/","sameSite":"lax"}`
	if string(raw) != want {
		t.Fatalf("cookie data:\n got %s\nwant %s", raw, want)
	}
	before := time.Now().UnixMilli()
	c.touch()
	if d := c.expires.UnixMilli() - before; d < 36000000 || d > 36000000+1000 {
		t.Fatalf("touch: expires in %dms", d)
	}
	so := c.serializeOptions()
	if got := SerializeCookie("connect.sid", "v", so, nil, nil); got != "connect.sid=v; Path=/; HttpOnly; Secure; SameSite=Lax" {
		t.Fatalf("serialize options: %s", got)
	}
	s := newSessionCookie(SessionCookieOptions{Path: "/", HttpOnly: true, SameSite: "true", Partitioned: true, Priority: "low"})
	if s.expires != nil || s.originalMaxAge != nil {
		t.Fatal("no maxAge: expires and originalMaxAge must be null")
	}
	if got := SerializeCookie("n", "v", s.serializeOptions(), nil, nil); got != "n=v; Path=/; HttpOnly; Partitioned; Priority=Low; SameSite=Strict" {
		t.Fatalf("serialize options: %s", got)
	}
}

func TestRedisStoreTTL(t *testing.T) {
	s := NewRedisStore(RedisStoreOptions{Client: ioredis.New("redis://localhost:1", nil)})
	if s.prefix != "sess:" || s.ttl != 86400 {
		t.Fatalf("defaults: %q %d", s.prefix, s.ttl)
	}
	empty := ""
	if p := NewRedisStore(RedisStoreOptions{Client: s.client, Prefix: &empty}).prefix; p != "" {
		t.Fatalf("explicit empty prefix: %q", p)
	}
	if got := s.getTTL(time.Time{}); got != 86400 {
		t.Fatalf("no expiry: %d", got)
	}
	// Math.ceil(ms / 1000)
	if got := s.getTTL(time.Now().Add(36000*time.Second - 300*time.Millisecond)); got != 36000 {
		t.Fatalf("ttl: %d", got)
	}
	if got := s.getTTL(time.Now().Add(-1500 * time.Millisecond)); got > 0 {
		t.Fatalf("expired: %d", got)
	}
}

// TestRedisStoreCommands runs against a real Redis when
// MEDUSA_TEST_REDIS_URL is set (e.g. redis://localhost:63790/15).
func TestRedisStoreCommands(t *testing.T) {
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL not set")
	}
	ctx := context.Background()
	client := ioredis.New(url, map[string]any{"keyPrefix": "kp:"})
	defer client.Close()
	prefix := "medusa:sess:"
	s := NewRedisStore(RedisStoreOptions{Client: client, Prefix: &prefix})
	key := "kp:medusa:sess:sid"
	client.Del(ctx, key)

	if _, ok, err := s.Get("sid"); ok || err != nil {
		t.Fatalf("get missing: %v %v", ok, err)
	}
	exp := time.Now().Add(36000 * time.Second)
	if err := s.Set("sid", []byte(upstreamSession), exp); err != nil {
		t.Fatal(err)
	}
	raw, ok, err := s.Get("sid")
	if err != nil || !ok || string(raw) != upstreamSession {
		t.Fatalf("get: %s %v %v", raw, ok, err)
	}
	if ttl := client.TTL(ctx, key).Val(); ttl != 36000*time.Second {
		t.Fatalf("ttl after set: %v", ttl)
	}
	if err := s.Touch("sid", nil, time.Now().Add(100*time.Second)); err != nil {
		t.Fatal(err)
	}
	if ttl := client.TTL(ctx, key).Val(); ttl != 100*time.Second {
		t.Fatalf("ttl after touch: %v", ttl)
	}
	// an expired session is deleted instead of stored
	if err := s.Set("sid", []byte(upstreamSession), time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if n := client.Exists(ctx, key).Val(); n != 0 {
		t.Fatal("expired session must be deleted")
	}
	_ = s.Set("sid", []byte(upstreamSession), exp)
	if err := s.Destroy("sid"); err != nil {
		t.Fatal(err)
	}
	if n := client.Exists(ctx, key).Val(); n != 0 {
		t.Fatal("destroy must delete the key")
	}
}
