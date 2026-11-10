// Package utils mirrors @medusajs/auth/src/utils (mfa, totp, verification
// tokens).
package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	scrypt "github.com/hitkey-io/medusago/packages/modules/providers/auth-emailpass/src/services"
)

const encryptionVersion = "v1"

var recoveryCodeHashConfig = scrypt.ScryptParams{LogN: 15, R: 8, P: 1}

func deriveKey(encryptionKey string) []byte {
	sum := sha256.Sum256([]byte(encryptionKey))
	return sum[:]
}

var b64url = base64.RawURLEncoding

// EncryptSecret mirrors encryptSecret: "v1:<iv>:<tag>:<ciphertext>" with
// AES-256-GCM (12-byte iv), base64url parts.
func EncryptSecret(secret, encryptionKey string) (string, error) {
	block, err := aes.NewCipher(deriveKey(encryptionKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, iv, []byte(secret), nil)
	ct, tag := sealed[:len(sealed)-gcm.Overhead()], sealed[len(sealed)-gcm.Overhead():]
	return strings.Join([]string{encryptionVersion, b64url.EncodeToString(iv), b64url.EncodeToString(tag), b64url.EncodeToString(ct)}, ":"), nil
}

// DecryptSecret mirrors decryptSecret.
func DecryptSecret(payload, encryptionKey string) (string, error) {
	parts := strings.Split(payload, ":")
	if len(parts) < 4 || parts[0] != encryptionVersion || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return "", errors.New("Invalid encrypted MFA secret")
	}
	iv, err1 := b64url.DecodeString(strings.TrimRight(parts[1], "="))
	tag, err2 := b64url.DecodeString(strings.TrimRight(parts[2], "="))
	ct, err3 := b64url.DecodeString(strings.TrimRight(parts[3], "="))
	if err1 != nil || err2 != nil || err3 != nil {
		return "", errors.New("Invalid encrypted MFA secret")
	}
	block, err := aes.NewCipher(deriveKey(encryptionKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return "", err
	}
	out, err := gcm.Open(nil, iv, append(ct, tag...), nil)
	if err != nil {
		return "", errors.New("Unsupported state or unable to authenticate data")
	}
	return string(out), nil
}

// GenerateRecoveryCode mirrors generateRecoveryCode: 16 random bytes as hex
// in groups of 4.
func GenerateRecoveryCode() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	h := hex.EncodeToString(b)
	var groups []string
	for i := 0; i < len(h); i += 4 {
		groups = append(groups, h[i:i+4])
	}
	return strings.Join(groups, "-")
}

func normalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

// HashRecoveryCode mirrors hashRecoveryCode (scrypt-kdf, base64).
func HashRecoveryCode(code string) (string, error) {
	h, err := scrypt.ScryptKdf(normalizeRecoveryCode(code), recoveryCodeHashConfig)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(h), nil
}

// VerifyRecoveryCodeHash mirrors verifyRecoveryCodeHash.
func VerifyRecoveryCodeHash(hash, code string) bool {
	raw, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	ok, err := scrypt.ScryptVerify(raw, normalizeRecoveryCode(code))
	return err == nil && ok
}

// GenerateVerificationToken mirrors generateVerificationToken.
func GenerateVerificationToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b64url.EncodeToString(b)
}

// HashVerificationToken mirrors hashVerificationToken.
func HashVerificationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
