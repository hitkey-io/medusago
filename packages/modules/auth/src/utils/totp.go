package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const base32Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

// EncodeBase32 mirrors encodeBase32 (no padding).
func EncodeBase32(buf []byte) string {
	var bits, value int
	var out strings.Builder
	for _, b := range buf {
		value = (value << 8) | int(b)
		bits += 8
		for bits >= 5 {
			out.WriteByte(base32Alphabet[(value>>(bits-5))&31])
			bits -= 5
			value &= (1 << bits) - 1
		}
	}
	if bits > 0 {
		out.WriteByte(base32Alphabet[(value<<(5-bits))&31])
	}
	return out.String()
}

var wsRe = regexp.MustCompile(`\s+`)

// DecodeBase32 mirrors decodeBase32.
func DecodeBase32(secret string) ([]byte, error) {
	s := strings.ToUpper(wsRe.ReplaceAllString(strings.TrimRight(secret, "="), ""))
	var bits, value int
	var out []byte
	for _, c := range s {
		idx := strings.IndexRune(base32Alphabet, c)
		if idx < 0 {
			return nil, errors.New("Invalid TOTP secret")
		}
		value = (value << 5) | idx
		bits += 5
		if bits >= 8 {
			out = append(out, byte((value>>(bits-8))&255))
			bits -= 8
			value &= (1 << bits) - 1
		}
	}
	return out, nil
}

// GenerateTotpSecret mirrors generateTotpSecret (20 random bytes, base32).
func GenerateTotpSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return EncodeBase32(b)
}

func hotp(key []byte, counter uint64, digits int) string {
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, counter)
	m := hmac.New(sha1.New, key)
	m.Write(msg)
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	code := (uint32(sum[off])&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, code%mod)
}

// GenerateTotpCode mirrors generateTotpCode.
func GenerateTotpCode(secret string, digits, period int, at time.Time) (string, error) {
	key, err := DecodeBase32(secret)
	if err != nil {
		return "", err
	}
	return hotp(key, uint64(at.Unix()/int64(period)), digits), nil
}

var digitsRe = regexp.MustCompile(`^\d+$`)

// VerifyTotpCode mirrors verifyTotpCode: the codes of the time steps within
// window*period seconds around now are accepted.
func VerifyTotpCode(secret, code string, digits, period, window int, at time.Time) bool {
	if code == "" || !digitsRe.MatchString(code) || len(code) != digits {
		return false
	}
	key, err := DecodeBase32(secret)
	if err != nil {
		return false
	}
	epoch := at.Unix()
	tolerance := int64(window * period)
	first := (epoch - tolerance) / int64(period)
	last := (epoch + tolerance) / int64(period)
	if first < 0 {
		first = 0
	}
	for c := first; c <= last; c++ {
		if hmac.Equal([]byte(hotp(key, uint64(c), digits)), []byte(code)) {
			return true
		}
	}
	return false
}

// encodeURIComponent mirrors JavaScript's encodeURIComponent.
func encodeURIComponent(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'()"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(unreserved, c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// GenerateTotpURI mirrors generateTotpUri (@otplib/uri generateTOTP): the
// label is "<issuer>:<account>", sha1 / 6 digits / 30s are omitted.
func GenerateTotpURI(issuer, accountName, secret string, digits, period int) string {
	label := accountName
	if issuer != "" {
		label = issuer + ":" + accountName
	}
	parts := strings.Split(label, ":")
	for i, p := range parts {
		parts[i] = encodeURIComponent(p)
	}
	var params []string
	if secret != "" {
		params = append(params, "secret="+encodeURIComponent(secret))
	}
	if issuer != "" {
		params = append(params, "issuer="+encodeURIComponent(issuer))
	}
	if digits != 0 && digits != 6 {
		params = append(params, fmt.Sprintf("digits=%d", digits))
	}
	if period != 30 {
		params = append(params, fmt.Sprintf("period=%d", period))
	}
	return "otpauth://totp/" + strings.Join(parts, ":") + "?" + strings.Join(params, "&")
}
