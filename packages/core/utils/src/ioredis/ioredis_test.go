package ioredis

import "testing"

func TestResolveLikeIoredis(t *testing.T) {
	cases := []struct {
		url  string
		opts map[string]any
		want Options
	}{
		{"redis://localhost:63790", nil, Options{Host: "localhost", Port: 63790, ConnectTimeout: 10000}},
		{"redis://:pw@cache:6380/3", map[string]any{"db": 5, "password": "x", "keyPrefix": "k:"},
			Options{Host: "cache", Port: 6380, Password: "pw", DB: 3, KeyPrefix: "k:", ConnectTimeout: 10000}},
		{"redis://u%40x:p%3Aw@h", nil, Options{Host: "h", Port: 6379, Username: "u@x", Password: "p:w", ConnectTimeout: 10000}},
		{"rediss://h?db=2", map[string]any{"connectTimeout": 500}, Options{Host: "h", Port: 6379, DB: 2, TLS: true, ConnectTimeout: 500}},
		{"localhost:7000", map[string]any{"db": "4"}, Options{Host: "localhost", Port: 7000, DB: 4, ConnectTimeout: 10000}},
		{"/tmp/redis.sock?db=1", nil, Options{Host: "localhost", Port: 6379, Path: "/tmp/redis.sock", DB: 1, ConnectTimeout: 10000}},
		{"6390", nil, Options{Host: "localhost", Port: 6390, ConnectTimeout: 10000}},
	}
	for _, c := range cases {
		if got := Resolve(c.url, c.opts); got != c.want {
			t.Errorf("%s: got %+v want %+v", c.url, got, c.want)
		}
	}
}
