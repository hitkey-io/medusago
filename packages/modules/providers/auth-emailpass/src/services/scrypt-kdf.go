package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"golang.org/x/crypto/scrypt"
)

// This file reproduces the scrypt-kdf package format (v0): a 96 byte key
// "scrypt" | version | logN | r | p | salt(32) | sha256 checksum(16) |
// hmac-sha256(64 bytes header, key = second half of the derived key).

// ScryptParams mirrors the scrypt-kdf parameters.
type ScryptParams struct {
	LogN uint8
	R    uint32
	P    uint32
}

// ScryptKdf mirrors Scrypt.kdf.
func ScryptKdf(passphrase string, params ScryptParams) ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return scryptKdfWithSalt(passphrase, params, salt)
}

func scryptKdfWithSalt(passphrase string, params ScryptParams, salt []byte) ([]byte, error) {
	dk, err := scrypt.Key([]byte(passphrase), salt, 1<<params.LogN, int(params.R), int(params.P), 64)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 96)
	copy(buf, "scrypt")
	buf[6] = 0
	buf[7] = params.LogN
	binary.BigEndian.PutUint32(buf[8:], params.R)
	binary.BigEndian.PutUint32(buf[12:], params.P)
	copy(buf[16:48], salt)
	sum := sha256.Sum256(buf[:48])
	copy(buf[48:64], sum[:16])
	mac := hmac.New(sha256.New, dk[32:])
	mac.Write(buf[:64])
	copy(buf[64:], mac.Sum(nil))
	return buf, nil
}

// ScryptVerify mirrors Scrypt.verify.
func ScryptVerify(key []byte, passphrase string) (bool, error) {
	if len(key) != 96 || !bytes.Equal(key[:6], []byte("scrypt")) || key[6] != 0 {
		return false, errors.New("Invalid key")
	}
	sum := sha256.Sum256(key[:48])
	if !bytes.Equal(sum[:16], key[48:64]) {
		return false, errors.New("Invalid key; checksum mismatch")
	}
	params := ScryptParams{LogN: key[7], R: binary.BigEndian.Uint32(key[8:]), P: binary.BigEndian.Uint32(key[12:])}
	dk, err := scrypt.Key([]byte(passphrase), key[16:48], 1<<params.LogN, int(params.R), int(params.P), 64)
	if err != nil {
		return false, err
	}
	mac := hmac.New(sha256.New, dk[32:])
	mac.Write(key[:64])
	return hmac.Equal(mac.Sum(nil), key[64:]), nil
}
