// Package auth mirrors @medusajs/utils/src/auth.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"hash"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// JwtOptions mirrors the jsonwebtoken sign/verify options used by Medusa.
type JwtOptions struct {
	// Algorithm for signing (default HS256).
	Algorithm string
}

// Now is the clock used for iat/exp (tests override it).
var Now = time.Now

// GenerateJwtToken mirrors generateJwtToken (jsonwebtoken.sign with an HMAC
// secret): iat and exp are appended to the payload.
func GenerateJwtToken(payload *dal.Record, secret, expiresIn string, opts JwtOptions) (string, error) {
	if secret == "" || expiresIn == "" {
		return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, "JWT secret and expiresIn must be provided when generating a token")
	}
	alg := opts.Algorithm
	if alg == "" {
		alg = "HS256"
	}
	iat := Now().Unix()
	exp, err := Timespan(expiresIn, iat)
	if err != nil {
		return "", err
	}
	p := dal.NewRecord()
	for _, k := range payload.Keys() {
		v, _ := payload.Get(k)
		p.Set(k, v)
	}
	if !p.Has("iat") {
		p.Set("iat", float64(iat))
	}
	p.Set("exp", float64(exp))

	header := dal.NewRecord()
	header.Set("alg", alg)
	header.Set("typ", "JWT")
	hb, _ := header.MarshalJSON()
	pb, err := p.MarshalJSON()
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
	h, err := hmacFor(alg, secret)
	if err != nil {
		return "", err
	}
	h.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil)), nil
}

func hmacFor(alg, secret string) (hash.Hash, error) {
	switch alg {
	case "HS256":
		return hmac.New(sha256.New, []byte(secret)), nil
	case "HS384":
		return hmac.New(sha512.New384, []byte(secret)), nil
	case "HS512":
		return hmac.New(sha512.New, []byte(secret)), nil
	}
	return nil, errors.New(`"algorithm" must be a valid string enum value`)
}

var msRe = regexp.MustCompile(`(?i)^(-?(?:\d+)?\.?\d+) *(milliseconds?|msecs?|ms|seconds?|secs?|s|minutes?|mins?|m|hours?|hrs?|h|days?|d|weeks?|w|years?|yrs?|y)?$`)

// Ms mirrors the ms package parse (milliseconds).
func Ms(s string) (float64, bool) {
	if len(s) > 100 {
		return 0, false
	}
	m := msRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	n, _ := strconv.ParseFloat(m[1], 64)
	const (
		sec  = 1000.0
		min  = sec * 60
		hour = min * 60
		day  = hour * 24
		week = day * 7
		year = day * 365.25
	)
	switch strings.ToLower(m[2]) {
	case "years", "year", "yrs", "yr", "y":
		return n * year, true
	case "weeks", "week", "w":
		return n * week, true
	case "days", "day", "d":
		return n * day, true
	case "hours", "hour", "hrs", "hr", "h":
		return n * hour, true
	case "minutes", "minute", "mins", "min", "m":
		return n * min, true
	case "seconds", "second", "secs", "sec", "s":
		return n * sec, true
	default:
		return n, true
	}
}

// Timespan mirrors jsonwebtoken's timespan: a ms string or a number of
// seconds, relative to timestamp.
func Timespan(expiresIn string, timestamp int64) (int64, error) {
	if n, err := strconv.ParseFloat(expiresIn, 64); err == nil && !strings.ContainsAny(expiresIn, " ") {
		return timestamp + int64(n), nil
	}
	ms, ok := Ms(expiresIn)
	if !ok {
		return 0, errors.New(`"expiresIn" should be a number of seconds or string representing a timespan`)
	}
	return int64(math.Floor(float64(timestamp) + ms/1000)), nil
}

// VerifyJwtToken mirrors jsonwebtoken.verify with an HMAC secret: the
// signature, then exp and nbf. It returns the payload in its key order.
func VerifyJwtToken(token, secret string) (*dal.Record, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("jwt malformed")
	}
	hb, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[0], "="))
	if err != nil {
		return nil, errors.New("invalid token")
	}
	hv, err := dal.DecodeOrderedJSON(hb)
	header, ok := hv.(*dal.Record)
	if err != nil || !ok {
		return nil, errors.New("invalid token")
	}
	algV, _ := header.Get("alg")
	alg, _ := algV.(string)
	if parts[2] == "" {
		return nil, errors.New("jwt signature is required")
	}
	if alg != "HS256" && alg != "HS384" && alg != "HS512" {
		return nil, errors.New("invalid algorithm")
	}
	h, _ := hmacFor(alg, secret)
	h.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[2], "="))
	if err != nil || !hmac.Equal(sig, h.Sum(nil)) {
		return nil, errors.New("invalid signature")
	}
	pb, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, errors.New("invalid token")
	}
	pv, err := dal.DecodeOrderedJSON(pb)
	payload, ok := pv.(*dal.Record)
	if err != nil || !ok {
		return nil, errors.New("invalid token")
	}
	now := float64(Now().Unix())
	if v, ok := payload.Get("nbf"); ok {
		n, isNum := v.(float64)
		if !isNum {
			return nil, errors.New("invalid nbf value")
		}
		if n > now {
			return nil, errors.New("jwt not active")
		}
	}
	if v, ok := payload.Get("exp"); ok {
		n, isNum := v.(float64)
		if !isNum {
			return nil, errors.New("invalid exp value")
		}
		if now >= n {
			return nil, errors.New("jwt expired")
		}
	}
	return payload, nil
}
