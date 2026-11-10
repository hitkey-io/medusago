package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// retrieveVersionedOrder mirrors super.retrieveOrder(id, { select: ["id",
// "version"] }) plus the "no previous versions" guard of revertLastVersion
// and undoLastChange.
func (s *OrderModuleService) retrieveVersionedOrder(ctx context.Context, orderID string) (string, int, error) {
	order, err := s.Internal("Order").Retrieve(ctx, orderID, utilsmodulessdk.FindConfig{Select: []string{"id", "version"}})
	if err != nil {
		return "", 0, err
	}
	version := versionOf(get(order, "version"))
	if version < 2 {
		return "", 0, invalidData(fmt.Sprintf("Order with id %s has no previous versions", orderID))
	}
	return fmt.Sprint(get(order, "id")), version, nil
}

// RevertLastVersion mirrors revertLastVersion.
func (s *OrderModuleService) RevertLastVersion(ctx context.Context, orderID string) error {
	id, version, err := s.retrieveVersionedOrder(ctx, orderID)
	if err != nil {
		return err
	}
	// one entity manager for the whole @InjectTransactionManager method
	ctx = dal.WithSharedIdentityMap(ctx)
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		return s.tx(m).revertLastChange(ctx, id, version)
	})
}

// UndoLastChange mirrors undoLastChange.
func (s *OrderModuleService) UndoLastChange(ctx context.Context, orderID string, lastOrderChange *dal.Record) error {
	id, version, err := s.retrieveVersionedOrder(ctx, orderID)
	if err != nil {
		return err
	}
	ctx = dal.WithSharedIdentityMap(ctx)
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		return s.tx(m).undoLastChange(ctx, id, version, lastOrderChange)
	})
}

// listVersioned lists the rows of a versioned model of the order.
func (s *OrderModuleService) listVersioned(ctx context.Context, className, orderID string, version int, fields ...string) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("order_id", orderID)
	f.Set("version", version)
	return s.Internal(className).List(ctx, f, utilsmodulessdk.FindConfig{Select: append([]string{"id", "version"}, fields...)})
}

func pluck(list []*dal.Record, key string) []any {
	out := make([]any, 0, len(list))
	for _, r := range list {
		out = append(out, get(r, key))
	}
	return out
}

// softDeleteVersioned soft deletes the rows of className created in version.
func (s *OrderModuleService) softDeleteVersioned(ctx context.Context, className, orderID string, version int, fields ...string) ([]*dal.Record, error) {
	rows, err := s.listVersioned(ctx, className, orderID, version, fields...)
	if err != nil {
		return nil, err
	}
	if _, err := s.Internal(className).SoftDelete(ctx, pluck(rows, "id")); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *OrderModuleService) setOrderVersion(ctx context.Context, orderID string, version int) error {
	sel := dal.NewRecord()
	sel.Set("id", orderID)
	d := dal.NewRecord()
	d.Set("version", version)
	u := dal.NewRecord()
	u.Set("selector", sel)
	u.Set("data", d)
	_, err := s.Internal("Order").Update(ctx, u)
	return err
}

// undoLastChange mirrors undoLastChange_: the changes of the current version
// go back to their previous status (pending by default) and their actions
// become unapplied, the versioned rows of the current version are soft
// deleted and the order returns to the previous version.
func (s *OrderModuleService) undoLastChange(ctx context.Context, orderID string, currentVersion int, lastOrderChange *dal.Record) error {
	changes, err := s.listVersioned(ctx, "OrderChange", orderID, currentVersion)
	if err != nil {
		return err
	}
	status := get(lastOrderChange, "status")
	if status == nil {
		status = "pending"
	}
	changeUpdates := make([]any, 0, len(changes))
	for _, c := range changes {
		u := dal.NewRecord()
		u.Set("id", get(c, "id"))
		u.Set("status", status)
		u.Set("confirmed_at", nil)
		changeUpdates = append(changeUpdates, u)
	}
	if _, err := s.Internal("OrderChange").Update(ctx, changeUpdates); err != nil {
		return err
	}

	actions, err := s.listVersioned(ctx, "OrderChangeAction", orderID, currentVersion)
	if err != nil {
		return err
	}
	actionUpdates := make([]any, 0, len(actions))
	for _, a := range actions {
		u := dal.NewRecord()
		u.Set("id", get(a, "id"))
		u.Set("applied", false)
		actionUpdates = append(actionUpdates, u)
	}
	if _, err := s.Internal("OrderChangeAction").Update(ctx, actionUpdates); err != nil {
		return err
	}

	for _, className := range []string{"OrderSummary", "OrderItem", "OrderShipping", "OrderCreditLine"} {
		if _, err := s.softDeleteVersioned(ctx, className, orderID, currentVersion); err != nil {
			return err
		}
	}
	return s.setOrderVersion(ctx, orderID, currentVersion-1)
}

// revertLastChange mirrors revertLastChange_: everything created in the
// current version (changes, actions, summaries, items, shipping methods,
// their adjustments and credit lines) is soft deleted, the returns of that
// version are deleted and the order returns to the previous version.
func (s *OrderModuleService) revertLastChange(ctx context.Context, orderID string, currentVersion int) error {
	for _, className := range []string{"OrderChange", "OrderChangeAction", "OrderSummary"} {
		if _, err := s.softDeleteVersioned(ctx, className, orderID, currentVersion); err != nil {
			return err
		}
	}
	orderItems, err := s.softDeleteVersioned(ctx, "OrderItem", orderID, currentVersion, "item_id")
	if err != nil {
		return err
	}
	orderShippings, err := s.softDeleteVersioned(ctx, "OrderShipping", orderID, currentVersion, "shipping_method_id")
	if err != nil {
		return err
	}

	adjustments := func(className, key string, ids []any) error {
		if len(ids) == 0 {
			return nil
		}
		f := dal.NewRecord()
		f.Set(key, ids)
		f.Set("version", currentVersion)
		rows, err := s.Internal(className).List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil || len(rows) == 0 {
			return err
		}
		_, err = s.Internal(className).SoftDelete(ctx, pluck(rows, "id"))
		return err
	}
	if err := adjustments("OrderLineItemAdjustment", "item_id", pluck(orderItems, "item_id")); err != nil {
		return err
	}
	if err := adjustments("OrderShippingMethodAdjustment", "shipping_method_id", pluck(orderShippings, "shipping_method_id")); err != nil {
		return err
	}

	if _, err := s.softDeleteVersioned(ctx, "OrderCreditLine", orderID, currentVersion); err != nil {
		return err
	}
	if err := s.setOrderVersion(ctx, orderID, currentVersion-1); err != nil {
		return err
	}
	sel := dal.NewRecord()
	sel.Set("order_id", orderID)
	sel.Set("order_version", currentVersion)
	_, err = s.Internal("Return").Delete(ctx, sel)
	return err
}
