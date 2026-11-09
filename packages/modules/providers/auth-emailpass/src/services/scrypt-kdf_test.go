package services

import (
	"encoding/base64"
	"testing"
)

// A hash produced by upstream (scrypt-kdf, logN 15) for "supersecret".
const upstreamHash = "c2NyeXB0AA8AAAAIAAAAAS6dWnZVLfW8eQPKNDu82KSIGHpqMApDjGTd6mT+vD3ciFQDNEkWaO93QN0HOh7U5wsljasrziC/mxzEFALYnqEp9fyAHd/4gR2KrCOjIoQW"

func TestScryptVerifyUpstreamHash(t *testing.T) {
	key, _ := base64.StdEncoding.DecodeString(upstreamHash)
	ok, err := ScryptVerify(key, "supersecret")
	if err != nil || !ok {
		t.Fatalf("verify failed: %v %v", ok, err)
	}
	if ok, _ := ScryptVerify(key, "wrong"); ok {
		t.Fatal("wrong password accepted")
	}
	k, err := ScryptKdf("pw", ScryptParams{LogN: 10, R: 8, P: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := ScryptVerify(k, "pw"); !ok {
		t.Fatal("roundtrip failed")
	}
}
