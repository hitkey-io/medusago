// Package ioredis resolves connection options the way ioredis 5 does for
// `new Redis(url, options)` and builds the matching go-redis client, so that
// a Go server connects to the same Redis server, database and key space as
// an upstream Node server given the same redisUrl and redisOptions.
package ioredis

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Options are the resolved ioredis connection options that go-redis can
// honour. Function-valued ioredis options (retryStrategy, ...) cannot be
// expressed in a config file and are ignored.
type Options struct {
	Host           string
	Port           int
	Path           string
	Username       string
	Password       string
	DB             int
	TLS            bool
	KeyPrefix      string
	ConnectionName string
	// ConnectTimeout in milliseconds (ioredis default 10000).
	ConnectTimeout int
}

var (
	intRe      = regexp.MustCompile(`^-?\d+$`)
	protocolRe = regexp.MustCompile(`(?i)^rediss?://`)
	leadIntRe  = regexp.MustCompile(`^\s*[-+]?\d+`)
)

// ParseURL mirrors parseURL of ioredis/built/utils.
func ParseURL(raw string) map[string]any {
	if intRe.MatchString(raw) {
		return map[string]any{"port": raw}
	}
	hasProtocol := protocolRe.MatchString(raw)
	protocolRelative := strings.HasPrefix(raw, "//")
	result := map[string]any{}
	if strings.HasPrefix(raw, "/") && !protocolRelative {
		path, query, hasQuery := strings.Cut(raw, "?")
		result["path"] = path
		if hasQuery {
			defaults(result, queryOptions(query))
		}
		return result
	}
	full := raw
	switch {
	case hasProtocol:
	case protocolRelative:
		full = "redis:" + raw
	default:
		full = "redis://" + raw
	}
	u, err := url.Parse(full)
	if err != nil {
		return result
	}
	if u.User != nil {
		pw, _ := u.User.Password()
		if u.User.Username() != "" || pw != "" {
			result["username"] = u.User.Username()
			result["password"] = pw
		}
	}
	if u.Path != "" && u.Path != "/" {
		if hasProtocol || protocolRelative {
			if len(u.Path) > 1 {
				result["db"] = u.Path[1:]
			}
		} else {
			result["path"] = u.Path
		}
	}
	if h := u.Hostname(); h != "" {
		result["host"] = h
	}
	if p := u.Port(); p != "" {
		result["port"] = p
	}
	defaults(result, queryOptions(u.RawQuery))
	return result
}

func queryOptions(q string) map[string]any {
	out := map[string]any{}
	vals, _ := url.ParseQuery(q)
	for k, v := range vals {
		val := v[len(v)-1]
		if k == "family" {
			if n, err := strconv.Atoi(val); err == nil {
				out[k] = n
				continue
			}
		}
		out[k] = val
	}
	return out
}

// defaults mirrors lodash.defaults: keys missing (or undefined) in dst are
// taken from src.
func defaults(dst, src map[string]any) {
	for k, v := range src {
		if cur, ok := dst[k]; !ok || cur == nil {
			dst[k] = v
		}
	}
}

// Resolve mirrors Redis.parseOptions(url, options): URL values win over the
// options object, then the ioredis defaults apply.
func Resolve(redisURL string, redisOptions map[string]any) Options {
	opts := map[string]any{}
	if redisURL != "" {
		defaults(opts, ParseURL(redisURL))
	}
	defaults(opts, redisOptions)
	if strings.HasPrefix(redisURL, "rediss://") {
		defaults(opts, map[string]any{"tls": true})
	}
	defaults(opts, map[string]any{
		"port": 6379, "host": "localhost", "db": 0, "keyPrefix": "", "connectTimeout": 10000,
	})
	o := Options{
		Host:           str(opts["host"]),
		Path:           str(opts["path"]),
		Username:       str(opts["username"]),
		Password:       str(opts["password"]),
		KeyPrefix:      str(opts["keyPrefix"]),
		ConnectionName: str(opts["connectionName"]),
		Port:           num(opts["port"]),
		DB:             num(opts["db"]),
		ConnectTimeout: num(opts["connectTimeout"]),
	}
	switch v := opts["tls"].(type) {
	case nil:
	case bool:
		o.TLS = v
	case string:
		o.TLS = v != ""
	default:
		o.TLS = true
	}
	return o
}

func str(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

// num mirrors parseInt for strings and keeps numbers.
func num(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(leadIntRe.FindString(x)), "+"))
		return n
	}
	return 0
}

// GoRedisOptions converts the resolved options to go-redis options.
func (o Options) GoRedisOptions() *redis.Options {
	ro := &redis.Options{
		Username:   o.Username,
		Password:   o.Password,
		DB:         o.DB,
		ClientName: o.ConnectionName,
	}
	if o.Path != "" {
		ro.Network = "unix"
		ro.Addr = o.Path
	} else {
		ro.Addr = net.JoinHostPort(o.Host, strconv.Itoa(o.Port))
	}
	if o.ConnectTimeout > 0 {
		ro.DialTimeout = time.Duration(o.ConnectTimeout) * time.Millisecond
	}
	if o.TLS {
		ro.TLSConfig = &tls.Config{ServerName: o.Host}
	}
	return ro
}

// Client is a go-redis client together with the ioredis keyPrefix, which
// ioredis prepends to the keys of every command.
type Client struct {
	*redis.Client
	KeyPrefix string
}

// New mirrors `new Redis(url, options)`.
func New(redisURL string, redisOptions map[string]any) *Client {
	o := Resolve(redisURL, redisOptions)
	return &Client{Client: redis.NewClient(o.GoRedisOptions()), KeyPrefix: o.KeyPrefix}
}
