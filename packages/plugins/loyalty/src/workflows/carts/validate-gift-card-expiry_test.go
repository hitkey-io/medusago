package carts

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// gcEmptyContainer resolves nothing (createContainer()).
type gcEmptyContainer struct{}

func (gcEmptyContainer) Resolve(string) any { return nil }

var gcTestWorkflowSeq atomic.Int64

func gcHoursFromNow(hours float64) string {
	return time.Now().Add(time.Duration(hours * float64(time.Hour))).UTC().Format("2006-01-02T15:04:05.000Z")
}

// gcRunStep runs a single step in an isolated workflow and returns the error
// message it threw, or "" if the step passed.
func gcRunStep[I any](step *composer.Step[I, any], input I) string {
	w := composer.CreateWorkflow(fmt.Sprintf("gift-card-expiry-test-%d", gcTestWorkflowSeq.Add(1)),
		func(e *composer.Execution, _ any) (any, error) {
			return step.Run(e, input)
		})
	if _, err := w.Run(context.Background(), gcEmptyContainer{}, nil); err != nil {
		return err.Error()
	}
	return ""
}

func gcTestCart() *dal.Record {
	return gcRecord("id", "cart_123", "currency_code", "usd", "gift_cards", []any{})
}

func TestValidateCartGiftCardStepRejectsExpired(t *testing.T) {
	got := gcRunStep(ValidateCartGiftCardStep, ValidateCartGiftCardStepInput{Cart: gcTestCart(), GiftCards: []*dal.Record{
		gcRecord("id", "gc_123", "code", "GC-EXPIRED", "currency_code", "usd", "expires_at", gcHoursFromNow(-1)),
	}})
	if got != "Gift card (GC-EXPIRED) has expired" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateCartGiftCardStepAcceptsFutureExpiry(t *testing.T) {
	got := gcRunStep(ValidateCartGiftCardStep, ValidateCartGiftCardStepInput{Cart: gcTestCart(), GiftCards: []*dal.Record{
		gcRecord("id", "gc_123", "code", "GC-VALID", "currency_code", "usd", "expires_at", gcHoursFromNow(1)),
	}})
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateCartGiftCardStepAcceptsNoExpiry(t *testing.T) {
	got := gcRunStep(ValidateCartGiftCardStep, ValidateCartGiftCardStepInput{Cart: gcTestCart(), GiftCards: []*dal.Record{
		gcRecord("id", "gc_123", "code", "GC-NO-EXPIRY", "currency_code", "usd", "expires_at", nil),
	}})
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateGiftCardsNotExpiredStepRejectsExpired(t *testing.T) {
	got := gcRunStep(ValidateGiftCardsNotExpiredStep, ValidateGiftCardsNotExpiredStepInput{GiftCards: []*dal.Record{
		gcRecord("code", "GC-VALID", "expires_at", gcHoursFromNow(1)),
		gcRecord("code", "GC-EXPIRED", "expires_at", gcHoursFromNow(-1)),
	}})
	if got != "Gift card (GC-EXPIRED) has expired" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateGiftCardsNotExpiredStepAcceptsValid(t *testing.T) {
	got := gcRunStep(ValidateGiftCardsNotExpiredStep, ValidateGiftCardsNotExpiredStepInput{GiftCards: []*dal.Record{
		gcRecord("code", "GC-VALID", "expires_at", gcHoursFromNow(1)),
		gcRecord("code", "GC-NO-EXPIRY", "expires_at", nil),
	}})
	if got != "" {
		t.Fatalf("got %q", got)
	}
}
