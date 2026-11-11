package ioredis

import (
	"testing"
	"time"
)

// withDefaults sets the ioredis defaults not given by a case.
func withDefaults(o Options) Options {
	if o.ConnectTimeout == 0 {
		o.ConnectTimeout = 10000
	}
	o.NoDelay = true
	if o.MaxRetriesPerRequest == 0 {
		o.MaxRetriesPerRequest = 20
	}
	if o.Role == "" {
		o.Role = "master"
	}
	if o.TLS && o.TLSOptions == (TLSOptions{}) {
		o.TLSOptions.RejectUnauthorized = true
	}
	return o
}

func TestResolveLikeIoredis(t *testing.T) {
	cases := []struct {
		url  string
		opts map[string]any
		want Options
	}{
		{"redis://localhost:63790", nil, Options{Host: "localhost", Port: 63790}},
		{"redis://:pw@cache:6380/3", map[string]any{"db": 5, "password": "x", "keyPrefix": "k:"},
			Options{Host: "cache", Port: 6380, Password: "pw", DB: 3, KeyPrefix: "k:"}},
		{"redis://u%40x:p%3Aw@h", nil, Options{Host: "h", Port: 6379, Username: "u@x", Password: "p:w"}},
		{"rediss://h?db=2", map[string]any{"connectTimeout": 500}, Options{Host: "h", Port: 6379, DB: 2, TLS: true, ConnectTimeout: 500}},
		{"localhost:7000", map[string]any{"db": "4"}, Options{Host: "localhost", Port: 7000, DB: 4}},
		{"/tmp/redis.sock?db=1", nil, Options{Host: "localhost", Port: 6379, Path: "/tmp/redis.sock", DB: 1}},
		{"6390", nil, Options{Host: "localhost", Port: 6390}},
		// family from the query string, socket options
		{"redis://h?family=6", map[string]any{"keepAlive": 3000, "commandTimeout": 1500, "connectionName": "medusa"},
			Options{Host: "h", Port: 6379, Family: 6, KeepAlive: 3000, CommandTimeout: 1500, ConnectionName: "medusa"}},
		// null maxRetriesPerRequest: no limit; lazyConnect
		{"redis://h", map[string]any{"maxRetriesPerRequest": nil, "lazyConnect": true},
			Options{Host: "h", Port: 6379, MaxRetriesPerRequest: -1, LazyConnect: true}},
		{"redis://h", map[string]any{"maxRetriesPerRequest": 3}, Options{Host: "h", Port: 6379, MaxRetriesPerRequest: 3}},
		// tls object
		{"redis://h", map[string]any{"tls": map[string]any{"rejectUnauthorized": false, "servername": "cache.internal"}},
			Options{Host: "h", Port: 6379, TLS: true, TLSOptions: TLSOptions{ServerName: "cache.internal"}}},
		{"redis://h", map[string]any{"tls": map[string]any{}},
			Options{Host: "h", Port: 6379, TLS: true, TLSOptions: TLSOptions{RejectUnauthorized: true}}},
		// sentinels
		{"", map[string]any{"sentinels": []any{map[string]any{"host": "s1", "port": 26380}, map[string]any{"host": "s2"}}, "name": "mymaster", "role": "slave", "sentinelPassword": "sp"},
			Options{Host: "localhost", Port: 6379, Sentinels: "s1:26380,s2:26379", Name: "mymaster", Role: "slave", SentinelPassword: "sp"}},
	}
	for _, c := range cases {
		if got, want := Resolve(c.url, c.opts), withDefaults(c.want); got != want {
			t.Errorf("%s %v:\n got %+v\nwant %+v", c.url, c.opts, got, want)
		}
	}
}

func TestGoRedisOptions(t *testing.T) {
	ro := Resolve("redis://h", map[string]any{"commandTimeout": 1500, "socketTimeout": 900, "maxRetriesPerRequest": 0}).GoRedisOptions()
	if ro.ReadTimeout != 900*time.Millisecond || ro.WriteTimeout != 900*time.Millisecond {
		t.Errorf("timeouts %v %v", ro.ReadTimeout, ro.WriteTimeout)
	}
	if ro.MaxRetries != -1 || ro.MinRetryBackoff != 50*time.Millisecond || ro.MaxRetryBackoff != 2*time.Second {
		t.Errorf("retries %d %v %v", ro.MaxRetries, ro.MinRetryBackoff, ro.MaxRetryBackoff)
	}
	ro = Resolve("redis://h", map[string]any{"tls": map[string]any{"rejectUnauthorized": false, "servername": "x"}}).GoRedisOptions()
	if ro.TLSConfig == nil || !ro.TLSConfig.InsecureSkipVerify || ro.TLSConfig.ServerName != "x" {
		t.Errorf("tls %+v", ro.TLSConfig)
	}
	if ro.Dialer == nil {
		t.Error("no dialer")
	}
}
