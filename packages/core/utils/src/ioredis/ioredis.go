// Package ioredis resolves connection options the way ioredis 5 does for
// `new Redis(url, options)` and builds the matching go-redis client, so that
// a Go server connects to the same Redis server, database and key space as
// an upstream Node server given the same redisUrl and redisOptions.
package ioredis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Options are the resolved ioredis connection options that go-redis can
// honour. Function-valued ioredis options (retryStrategy, reconnectOnError,
// ...) cannot be expressed in a config file: the default retry strategy
// (min(times * 50, 2000) ms) is applied as the go-redis retry backoff.
// Options without a go-redis counterpart (enableOfflineQueue,
// enableReadyCheck, autoResubscribe, autoResendUnfulfilledCommands,
// enableAutoPipelining, stringNumbers, readOnly, natMap) are ignored:
// go-redis dials on demand and retries each command up to
// maxRetriesPerRequest times.
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
	// Family is the IP family (4, 6, or 0 for both).
	Family int
	// KeepAlive is the TCP keep-alive initial delay in milliseconds (0:
	// keep-alive disabled, the ioredis default).
	KeepAlive int
	// NoDelay disables Nagle's algorithm (ioredis default true).
	NoDelay bool
	// CommandTimeout and SocketTimeout in milliseconds (0: none).
	CommandTimeout int
	SocketTimeout  int
	// MaxRetriesPerRequest (ioredis default 20); -1 for null (no limit).
	MaxRetriesPerRequest int
	// LazyConnect false (the default) connects when the client is created.
	LazyConnect bool
	// DisableClientInfo skips CLIENT SETINFO on connect.
	DisableClientInfo bool
	// TLSOptions are the options of a tls object (tls: {...}).
	TLSOptions TLSOptions
	// Sentinels ("host:port") with the master Name, the Role ("master" or
	// "slave") and the sentinel credentials: a sentinel-managed connection.
	Sentinels        string
	Name             string
	Role             string
	SentinelUsername string
	SentinelPassword string
}

// TLSOptions are the tls.connect options of ioredis's tls object that
// can be written in a config file (PEM strings).
type TLSOptions struct {
	ServerName         string
	RejectUnauthorized bool
	CA, Cert, Key      string
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
		// null is a value (only undefined takes the default); JSON has no
		// undefined, so only a missing key does
		if _, ok := dst[k]; !ok {
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
		"family": 0, "keepAlive": 0, "noDelay": true, "maxRetriesPerRequest": 20, "lazyConnect": false,
		"role": "master", "disableClientInfo": false,
	})
	o := Options{
		Host:              str(opts["host"]),
		Path:              str(opts["path"]),
		Username:          str(opts["username"]),
		Password:          str(opts["password"]),
		KeyPrefix:         str(opts["keyPrefix"]),
		ConnectionName:    str(opts["connectionName"]),
		Port:              num(opts["port"]),
		DB:                num(opts["db"]),
		ConnectTimeout:    num(opts["connectTimeout"]),
		Family:            num(opts["family"]),
		KeepAlive:         num(opts["keepAlive"]),
		NoDelay:           truthy(opts["noDelay"]),
		CommandTimeout:    num(opts["commandTimeout"]),
		SocketTimeout:     num(opts["socketTimeout"]),
		LazyConnect:       truthy(opts["lazyConnect"]),
		DisableClientInfo: truthy(opts["disableClientInfo"]),
		Name:              str(opts["name"]),
		Role:              str(opts["role"]),
		SentinelUsername:  str(opts["sentinelUsername"]),
		SentinelPassword:  str(opts["sentinelPassword"]),
	}
	if v, ok := opts["maxRetriesPerRequest"]; ok && v == nil {
		o.MaxRetriesPerRequest = -1
	} else {
		o.MaxRetriesPerRequest = num(v)
	}
	switch v := opts["tls"].(type) {
	case nil:
	case bool:
		o.TLS = v
	case string:
		o.TLS = v != ""
	case map[string]any:
		o.TLS = true
		o.TLSOptions = TLSOptions{
			ServerName:         str(v["servername"]),
			RejectUnauthorized: v["rejectUnauthorized"] == nil || truthy(v["rejectUnauthorized"]),
			CA:                 pem(v["ca"]),
			Cert:               pem(v["cert"]),
			Key:                pem(v["key"]),
		}
	default:
		o.TLS = true
	}
	if o.TLS && o.TLSOptions == (TLSOptions{}) {
		o.TLSOptions.RejectUnauthorized = true
	}
	if list, ok := opts["sentinels"].([]any); ok {
		var addrs []string
		for _, s := range list {
			if m, ok := s.(map[string]any); ok {
				host := str(m["host"])
				if host == "" {
					host = "localhost"
				}
				port := num(m["port"])
				if port == 0 {
					port = 26379
				}
				addrs = append(addrs, net.JoinHostPort(host, strconv.Itoa(port)))
			}
		}
		o.Sentinels = strings.Join(addrs, ",")
	}
	return o
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case int:
		return x != 0
	}
	return true
}

// pem joins a PEM string or a list of them.
func pem(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		parts := make([]string, 0, len(x))
		for _, p := range x {
			parts = append(parts, str(p))
		}
		return strings.Join(parts, "\n")
	}
	return ""
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
		Username:        o.Username,
		Password:        o.Password,
		DB:              o.DB,
		ClientName:      o.ConnectionName,
		DisableIdentity: o.DisableClientInfo,
		// ioredis retryStrategy: min(times * 50, 2000) ms
		MinRetryBackoff: 50 * time.Millisecond,
		MaxRetryBackoff: 2000 * time.Millisecond,
	}
	switch {
	case o.MaxRetriesPerRequest < 0:
		ro.MaxRetries = math.MaxInt32
	case o.MaxRetriesPerRequest == 0:
		ro.MaxRetries = -1 // go-redis: no retry
	default:
		ro.MaxRetries = o.MaxRetriesPerRequest
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
	if timeout := o.CommandTimeout; timeout > 0 || o.SocketTimeout > 0 {
		if timeout == 0 || (o.SocketTimeout > 0 && o.SocketTimeout < timeout) {
			timeout = o.SocketTimeout
		}
		ro.ReadTimeout = time.Duration(timeout) * time.Millisecond
		ro.WriteTimeout = ro.ReadTimeout
	}
	if o.TLS {
		ro.TLSConfig = o.tlsConfig()
	}
	ro.Dialer = o.dialer(ro.DialTimeout, ro.TLSConfig)
	return ro
}

func (o Options) tlsConfig() *tls.Config {
	c := &tls.Config{ServerName: o.Host, InsecureSkipVerify: !o.TLSOptions.RejectUnauthorized}
	if o.TLSOptions.ServerName != "" {
		c.ServerName = o.TLSOptions.ServerName
	}
	if o.TLSOptions.CA != "" {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM([]byte(o.TLSOptions.CA))
		c.RootCAs = pool
	}
	if o.TLSOptions.Cert != "" && o.TLSOptions.Key != "" {
		if cert, err := tls.X509KeyPair([]byte(o.TLSOptions.Cert), []byte(o.TLSOptions.Key)); err == nil {
			c.Certificates = []tls.Certificate{cert}
		}
	}
	return c
}

// dialer mirrors the socket options of ioredis: the IP family, TCP
// keep-alive (disabled unless keepAlive > 0) and no-delay.
func (o Options) dialer(timeout time.Duration, tlsConfig *tls.Config) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if network == "tcp" {
			switch o.Family {
			case 4:
				network = "tcp4"
			case 6:
				network = "tcp6"
			}
		}
		d := &net.Dialer{Timeout: timeout, KeepAlive: -1}
		if o.KeepAlive > 0 {
			d.KeepAlive = time.Duration(o.KeepAlive) * time.Millisecond
		}
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		if tcp, ok := conn.(*net.TCPConn); ok {
			_ = tcp.SetNoDelay(o.NoDelay)
		}
		if tlsConfig != nil {
			tc := tls.Client(conn, tlsConfig)
			if err := tc.HandshakeContext(ctx); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return tc, nil
		}
		return conn, nil
	}
}

// Client is a go-redis client together with the ioredis keyPrefix, which
// ioredis prepends to the keys of every command.
type Client struct {
	*redis.Client
	KeyPrefix string
}

// New mirrors `new Redis(url, options)`: with sentinels, a connection to
// the master (or a replica for role "slave") named name; unless
// lazyConnect, the client connects right away (in the background, like
// ioredis, failures are retried by the next commands).
func New(redisURL string, redisOptions map[string]any) *Client {
	o := Resolve(redisURL, redisOptions)
	var client *redis.Client
	if o.Sentinels != "" {
		ro := o.GoRedisOptions()
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       o.Name,
			SentinelAddrs:    strings.Split(o.Sentinels, ","),
			SentinelUsername: o.SentinelUsername,
			SentinelPassword: o.SentinelPassword,
			ReplicaOnly:      o.Role == "slave",
			Username:         ro.Username,
			Password:         ro.Password,
			DB:               ro.DB,
			ClientName:       ro.ClientName,
			DisableIdentity:  ro.DisableIdentity,
			MaxRetries:       ro.MaxRetries,
			MinRetryBackoff:  ro.MinRetryBackoff,
			MaxRetryBackoff:  ro.MaxRetryBackoff,
			DialTimeout:      ro.DialTimeout,
			ReadTimeout:      ro.ReadTimeout,
			WriteTimeout:     ro.WriteTimeout,
			TLSConfig:        ro.TLSConfig,
		})
	} else {
		client = redis.NewClient(o.GoRedisOptions())
	}
	if !o.LazyConnect {
		go func() { _ = client.Ping(context.Background()).Err() }()
	}
	return &Client{Client: client, KeyPrefix: o.KeyPrefix}
}
