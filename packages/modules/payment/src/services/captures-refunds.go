package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/defaults"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

func invalidData(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

// CurrencyEpsilon mirrors getEpsilonFromDecimalPrecision(defaultCurrencies[code]?.decimal_digits).
func CurrencyEpsilon(code string) *totals.BN {
	upper := strings.ToUpper(code)
	for _, c := range defaults.DefaultCurrencies {
		if c.Code == upper {
			return totals.GetEpsilonFromDecimalPrecision(c.DecimalDigits)
		}
	}
	return totals.GetEpsilonFromDecimalPrecision(-1)
}

func lockPayment(ctx context.Context, tx *utilsmodulessdk.MedusaService, id any) error {
	db := tx.DB()
	if _, err := db.Exec(ctx, "SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}
	rows, err := db.Query(ctx, `select "id" from "payment" where "id" = $1 for update`, id)
	if err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}

func sumAmounts(list []*dal.Record, key string) *totals.BN {
	out := totals.Convert(0)
	for _, r := range list {
		out = totals.Add(out, get(r, key))
	}
	return out
}

// CapturePayment mirrors capturePayment.
func (s *PaymentModuleService) CapturePayment(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	return s.capturePayment(ctx, data, false)
}

// capturePayment mirrors capturePayment; nested runs inside the
// transaction of authorizePaymentSession_ (sharedContext).
func (s *PaymentModuleService) capturePayment(ctx context.Context, data *dal.Record, nested bool) (*dal.Record, error) {
	if !nested {
		ctx = dal.WithSharedIdentityMap(ctx)
	}
	paymentID := get(data, "payment_id")
	payment, err := s.Internal("Payment").Retrieve(ctx, paymentID, utilsmodulessdk.FindConfig{
		Select:    []string{"id", "data", "provider_id", "payment_collection_id", "amount", "raw_amount", "currency_code", "captured_at", "canceled_at"},
		Relations: []string{"captures.raw_amount"},
	})
	if err != nil {
		return nil, err
	}
	isCaptured, _ := get(data, "is_captured").(bool)
	if !isCaptured {
		isCaptured = get(payment, "captured_at") != nil
	}
	isFullyCaptured := false
	var capture *dal.Record
	err = s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		if get(payment, "canceled_at") != nil {
			return invalidData(fmt.Sprintf("The payment: %v has been canceled.", get(payment, "id")))
		}
		if get(payment, "captured_at") != nil {
			isFullyCaptured = true
			return nil
		}
		amount := get(data, "amount")
		if amount == nil {
			amount = get(payment, "amount")
		} else if totals.Lte(amount, 0) {
			return invalidData("Capture amount must be greater than 0.")
		}
		if err := lockPayment(ctx, tx, paymentID); err != nil {
			return err
		}
		locked, err := tx.Internal("Payment").Retrieve(ctx, paymentID, utilsmodulessdk.FindConfig{Select: []string{"id"}, Relations: []string{"captures.raw_amount"}})
		if err != nil {
			return err
		}
		captured := sumAmounts(recordList(get(locked, "captures")), "raw_amount")
		authorized := get(payment, "raw_amount")
		code, _ := get(payment, "currency_code").(string)
		remaining := totals.Sub(authorized, captured)
		if totals.Gt(roundToCurrencyPrecision(amount, code), roundToCurrencyPrecision(remaining, code)) {
			return invalidData("You cannot capture more than the authorized amount substracted by what is already captured.")
		}
		isFullyCaptured = totals.Gte(roundToCurrencyPrecision(totals.Add(captured, amount), code), roundToCurrencyPrecision(authorized, code))
		c := dal.NewRecord()
		c.Set("payment", paymentID)
		c.Set("amount", amount)
		c.Set("created_by", get(data, "captured_by"))
		created, err := tx.Internal("Capture").Create(ctx, []*dal.Record{c})
		if err != nil {
			return err
		}
		capture = created[0]
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !isCaptured {
		pctx := dal.NewRecord()
		if capture != nil {
			pctx.Set("idempotency_key", get(capture, "id"))
		}
		res, err := s.provider.CapturePayment(ctx, fmt.Sprint(get(payment, "provider_id")), kv("data", get(payment, "data"), "context", pctx))
		if err != nil {
			if capture != nil {
				_, _ = s.Internal("Capture").Delete(ctx, []any{get(capture, "id")})
			}
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", get(payment, "id"))
		setDefined(u, res, "data")
		if isFullyCaptured {
			u.Set("captured_at", dal.Now())
		}
		if _, err := s.Internal("Payment").Update(ctx, u); err != nil {
			return nil, err
		}
	} else if isFullyCaptured && get(payment, "captured_at") == nil {
		u := dal.NewRecord()
		u.Set("id", get(payment, "id"))
		u.Set("captured_at", dal.Now())
		if _, err := s.Internal("Payment").Update(ctx, u); err != nil {
			return nil, err
		}
	}
	if err := s.maybeUpdatePaymentCollection(ctx, get(payment, "payment_collection_id")); err != nil {
		return nil, err
	}
	return s.Internal("Payment").Retrieve(context.Background(), get(payment, "id"), utilsmodulessdk.FindConfig{Relations: []string{"captures"}})
}

// RefundPayment mirrors refundPayment.
func (s *PaymentModuleService) RefundPayment(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	ctx = dal.WithSharedIdentityMap(ctx)
	paymentID := get(data, "payment_id")
	payment, err := s.Internal("Payment").Retrieve(ctx, paymentID, utilsmodulessdk.FindConfig{
		Select:    []string{"id", "currency_code", "data", "provider_id", "payment_collection_id", "amount", "raw_amount"},
		Relations: []string{"captures.raw_amount", "refunds.raw_amount"},
	})
	if err != nil {
		return nil, err
	}
	var refund *dal.Record
	alreadyRefunded := false
	err = s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		amount := get(data, "amount")
		if amount == nil {
			amount = get(payment, "amount")
		} else if totals.Lte(amount, 0) {
			return invalidData("Refund amount must be greater than 0.")
		}
		if err := lockPayment(ctx, tx, paymentID); err != nil {
			return err
		}
		locked, err := tx.Internal("Payment").Retrieve(ctx, paymentID, utilsmodulessdk.FindConfig{Select: []string{"id"}, Relations: []string{"captures.raw_amount", "refunds"}, WithDeleted: true})
		if err != nil {
			return err
		}
		refunds := recordList(get(locked, "refunds"))
		var resume *dal.Record
		if key := get(data, "idempotency_key"); key != nil {
			for _, r := range refunds {
				if get(r, "idempotency_key") == key {
					resume = r
					break
				}
			}
			if resume != nil {
				if !totals.Eq(get(resume, "raw_amount"), amount) {
					return invalidData(fmt.Sprintf("A refund with idempotency key %v already exists on payment %v for a different amount.", key, paymentID))
				}
				switch get(resume, "status") {
				case "pending":
					return common.NewMedusaError(common.MedusaErrorTypes.CONFLICT, fmt.Sprintf("A refund with idempotency key %v is still in progress on payment %v.", key, paymentID))
				case "succeeded":
					refund = resume
					alreadyRefunded = true
					return nil
				}
			}
		}
		var active []*dal.Record
		for _, r := range refunds {
			if get(r, "deleted_at") == nil && (resume == nil || get(r, "id") != get(resume, "id")) {
				active = append(active, r)
			}
		}
		captured := sumAmounts(recordList(get(locked, "captures")), "raw_amount")
		refunded := sumAmounts(active, "raw_amount")
		total := totals.Add(refunded, amount)
		code, _ := get(payment, "currency_code").(string)
		if totals.Lt(totals.Sub(captured, total), totals.Mult(CurrencyEpsilon(code), -1)) {
			return invalidData("You cannot refund more than what is captured on the payment.")
		}
		if resume != nil {
			if get(resume, "deleted_at") != nil {
				if _, err := tx.Internal("Refund").Restore(ctx, []any{get(resume, "id")}); err != nil {
					return err
				}
			}
			u := dal.NewRecord()
			u.Set("id", get(resume, "id"))
			u.Set("status", "pending")
			res, err := tx.Internal("Refund").Update(ctx, u)
			if err != nil {
				return err
			}
			refund, _ = res.(*dal.Record)
			return nil
		}
		r := dal.NewRecord()
		r.Set("payment", paymentID)
		r.Set("amount", amount)
		for _, k := range []string{"created_by", "note", "refund_reason_id", "metadata", "idempotency_key"} {
			if v, ok := data.Get(k); ok {
				r.Set(k, v)
			}
		}
		r.Set("status", "pending")
		created, err := tx.Internal("Refund").Create(ctx, []*dal.Record{r})
		if err != nil {
			return err
		}
		refund = created[0]
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !alreadyRefunded {
		key := get(refund, "idempotency_key")
		if key == nil {
			key = get(refund, "id")
		}
		res, err := s.provider.RefundPayment(ctx, fmt.Sprint(get(payment, "provider_id")), kv(
			"data", get(payment, "data"),
			"amount", get(refund, "raw_amount"),
			"context", kv("idempotency_key", key),
		))
		if err != nil {
			u := dal.NewRecord()
			u.Set("id", get(refund, "id"))
			u.Set("status", "failed")
			_, _ = s.Internal("Refund").Update(ctx, u)
			_, _ = s.Internal("Refund").SoftDelete(ctx, []any{get(refund, "id")})
			return nil, err
		}
		pu := dal.NewRecord()
		pu.Set("id", get(payment, "id"))
		setDefined(pu, res, "data")
		if _, err := s.Internal("Payment").Update(ctx, pu); err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", get(refund, "id"))
		u.Set("status", "succeeded")
		if _, err := s.Internal("Refund").Update(ctx, u); err != nil {
			return nil, err
		}
		if err := s.maybeUpdatePaymentCollection(ctx, get(payment, "payment_collection_id")); err != nil {
			return nil, err
		}
	}
	return s.Internal("Payment").Retrieve(context.Background(), get(payment, "id"), utilsmodulessdk.FindConfig{Relations: []string{"refunds"}})
}

// CancelPayment mirrors cancelPayment.
func (s *PaymentModuleService) CancelPayment(ctx context.Context, id any) (*dal.Record, error) {
	payment, err := s.Internal("Payment").Retrieve(ctx, id, utilsmodulessdk.FindConfig{Select: []string{"id", "data", "provider_id"}})
	if err != nil {
		return nil, err
	}
	res, err := s.provider.CancelPayment(ctx, fmt.Sprint(get(payment, "provider_id")), kv("data", get(payment, "data"), "context", kv("idempotency_key", get(payment, "id"))))
	if err != nil {
		return nil, err
	}
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("canceled_at", dal.Now())
	setDefined(u, res, "data")
	if _, err := s.Internal("Payment").Update(ctx, u); err != nil {
		return nil, err
	}
	return s.Internal("Payment").Retrieve(ctx, id, utilsmodulessdk.FindConfig{})
}
