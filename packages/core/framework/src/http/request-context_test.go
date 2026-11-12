package http

import (
	"context"
	nethttp "net/http/httptest"
	"testing"
)

type ctxKey struct{}

func TestCtxIgnoresClientDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey{}, "v"))
	r := NewMedusaRequest(nethttp.NewRequest("GET", "/", nil).WithContext(ctx))
	cancel() // the client disconnected
	if r.Request.Context().Err() == nil {
		t.Fatal("the request context should report the disconnection")
	}
	if err := r.Ctx().Err(); err != nil {
		t.Fatalf("Ctx() = %v, want a context that is not canceled", err)
	}
	if r.Ctx().Value(ctxKey{}) != "v" {
		t.Fatal("Ctx() lost the request values")
	}
}
