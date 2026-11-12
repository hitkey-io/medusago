package services

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// Port of promotion/src/utils/compute-actions/buy-get.ts.

// EligibleItem mirrors EligibleItem.
type EligibleItem struct {
	ItemID   string
	Quantity *totals.BN
}

// bnMap is a JS Map<string, BigNumberInput> (insertion ordered).
type bnMap struct {
	keys   []string
	values map[string]*totals.BN
}

func newBNMap() *bnMap { return &bnMap{values: map[string]*totals.BN{}} }

// getOr mirrors map.get(k) || MathBN.convert(0).
func (m *bnMap) getOr(k string) *totals.BN {
	if v, ok := m.values[k]; ok && v != nil {
		return v
	}
	return totals.Convert(0)
}

func (m *bnMap) has(k string) bool {
	_, ok := m.values[k]
	return ok
}

func (m *bnMap) set(k string, v *totals.BN) {
	if _, ok := m.values[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.values[k] = v
}

// sum mirrors MathBN.sum(...map.values()).
func (m *bnMap) sum() *totals.BN {
	vs := []any{0}
	for _, k := range m.keys {
		vs = append(vs, m.values[k])
	}
	return totals.Add(vs...)
}

// jsNumberBN mirrors BigNumber.toNumber() (the value becomes a JS number).
func jsNumberBN(v any) *totals.BN {
	return totals.Convert(number(v))
}

// buyGetUnitPrice mirrors getUnitPrice: subtotal / quantity, 0 without a
// positive quantity.
func buyGetUnitPrice(item *dal.Record) *totals.BN {
	if totals.Gt(cget(item, "quantity"), 0) {
		return totals.Div(cget(item, "subtotal"), cget(item, "quantity"))
	}
	return totals.Convert(0)
}

// buyGetSortByUnitPrice mirrors items.sort(compareByUnitPrice(direction)):
// items ranked by unit price, equal prices keep their order (stable sort).
func buyGetSortByUnitPrice(items []*dal.Record, ascending bool) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := buyGetUnitPrice(items[i]), buyGetUnitPrice(items[j])
		if ascending {
			return totals.Lt(a, b)
		}
		return totals.Gt(a, b)
	})
}

// sortTargetItemsByStrategy mirrors sortTargetItemsByStrategy: the order in
// which target units are discounted ("most_expensive_first" by default,
// "cheapest_first" ascending). The input is not modified.
func sortTargetItemsByStrategy(items []*dal.Record, strategy string) []*dal.Record {
	out := append([]*dal.Record{}, items...)
	buyGetSortByUnitPrice(out, strategy == "cheapest_first")
	return out
}

// isValidBuyGetPromotionContext mirrors isValidPromotionContext.
func isValidBuyGetPromotionContext(promotion *dal.Record, items []*dal.Record, itemsDefined bool) (bool, error) {
	if !itemsDefined {
		return false, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, `"items" should be present as an array in the context to compute actions`)
	}
	if len(items) == 0 {
		return false, nil
	}
	am := crec(promotion, "application_method")
	minimumBuyQuantity := totals.Convert(firstDefined(cget(am, "buy_rules_min_quantity"), 0))
	if totals.Lte(minimumBuyQuantity, 0) || len(crecords(cget(am, "buy_rules"))) == 0 {
		return false, nil
	}
	return true, nil
}

type buyGetConfig struct {
	minimumBuyQuantity   *totals.BN
	targetApplyQuantity  *totals.BN
	maximumApplyQuantity *totals.BN
	promotionValue       any
}

// normalizePromotionApplicationConfiguration mirrors
// normalizePromotionApplicationConfiguration.
func normalizePromotionApplicationConfiguration(promotion *dal.Record) buyGetConfig {
	am := crec(promotion, "application_method")
	return buyGetConfig{
		minimumBuyQuantity:   totals.Convert(firstDefined(cget(am, "buy_rules_min_quantity"), 0)),
		targetApplyQuantity:  totals.Convert(firstDefined(cget(am, "apply_to_quantity"), 0)),
		maximumApplyQuantity: totals.Convert(firstDefined(cget(am, "max_quantity"), 1)),
		promotionValue:       firstDefined(cget(am, "value"), 0),
	}
}

// eligibleItemsByCode is a JS Map<string, EligibleItem[]> keyed by
// promotion code (insertion ordered).
type eligibleItemsByCode struct {
	codes []string
	items map[string][]EligibleItem
}

func newEligibleItemsByCode() *eligibleItemsByCode {
	return &eligibleItemsByCode{items: map[string][]EligibleItem{}}
}

func (m *eligibleItemsByCode) set(code string, items []EligibleItem) {
	if _, ok := m.items[code]; !ok {
		m.codes = append(m.codes, code)
	}
	m.items[code] = items
}

// calculateRemainingQuantities mirrors calculateRemainingQuantities.
func calculateRemainingQuantities(eligibleItems []*dal.Record, itemsMap *eligibleItemsByCode, currentPromotionCode string) *bnMap {
	remaining := newBNMap()
	for _, item := range eligibleItems {
		id := cstr(item, "id")
		consumed := totals.Convert(0)
		for _, code := range itemsMap.codes {
			if code == currentPromotionCode {
				continue
			}
			for _, e := range itemsMap.items[code] {
				if e.ItemID == id {
					consumed = totals.Add(consumed, e.Quantity)
				}
			}
		}
		remaining.set(id, totals.Max(totals.Sub(cget(item, "quantity"), consumed), 0))
	}
	return remaining
}

type buyGetApplication struct {
	buyItems    []EligibleItem
	targetItems []EligibleItem
	isValid     bool
}

// preparePromotionApplicationState mirrors preparePromotionApplicationState.
func preparePromotionApplicationState(eligibleBuyItems, eligibleTargetItems []*dal.Record, remainingBuy, remainingTarget *bnMap, config buyGetConfig, appliedPromotionQuantity *totals.BN) buyGetApplication {
	if totals.Lt(remainingBuy.sum(), config.minimumBuyQuantity) {
		return buyGetApplication{}
	}
	var buyItems []EligibleItem
	accumulated := totals.Convert(0)

	// Items that cannot serve as targets are reserved for the buy side first.
	targetIDs := map[string]bool{}
	for _, t := range eligibleTargetItems {
		targetIDs[cstr(t, "id")] = true
	}
	var reservationOrder []*dal.Record
	for _, b := range eligibleBuyItems {
		if !targetIDs[cstr(b, "id")] {
			reservationOrder = append(reservationOrder, b)
		}
	}
	for _, b := range eligibleBuyItems {
		if targetIDs[cstr(b, "id")] {
			reservationOrder = append(reservationOrder, b)
		}
	}
	for _, b := range reservationOrder {
		if totals.Gte(accumulated, config.minimumBuyQuantity) {
			break
		}
		available := remainingBuy.getOr(cstr(b, "id"))
		if totals.Lte(available, 0) {
			continue
		}
		reservable := totals.Min(available, totals.Sub(config.minimumBuyQuantity, accumulated))
		if totals.Lte(reservable, 0) {
			continue
		}
		buyItems = append(buyItems, EligibleItem{ItemID: cstr(b, "id"), Quantity: jsNumberBN(reservable)})
		accumulated = totals.Add(accumulated, reservable)
	}
	if totals.Lt(accumulated, config.minimumBuyQuantity) {
		return buyGetApplication{}
	}
	usedInBuy := newBNMap()
	for _, b := range buyItems {
		usedInBuy.set(b.ItemID, totals.Add(usedInBuy.getOr(b.ItemID), b.Quantity))
	}
	var targetItems []EligibleItem
	availableTarget := totals.Convert(0)
	for _, t := range eligibleTargetItems {
		id := cstr(t, "id")
		applicable := totals.Sub(remainingTarget.getOr(id), usedInBuy.getOr(id))
		if totals.Lte(applicable, 0) {
			continue
		}
		remainingNeeded := totals.Sub(config.targetApplyQuantity, availableTarget)
		remainingMax := totals.Sub(config.maximumApplyQuantity, appliedPromotionQuantity)
		fulfillable := totals.Min(remainingNeeded, applicable, remainingMax)
		if totals.Lte(fulfillable, 0) {
			continue
		}
		targetItems = append(targetItems, EligibleItem{ItemID: id, Quantity: jsNumberBN(fulfillable)})
		availableTarget = totals.Add(availableTarget, fulfillable)
		if totals.Gte(availableTarget, config.targetApplyQuantity) {
			break
		}
	}
	return buyGetApplication{
		buyItems:    buyItems,
		targetItems: targetItems,
		isValid:     totals.Gte(availableTarget, config.targetApplyQuantity),
	}
}

// applyPromotionToTargetItems mirrors applyPromotionToTargetItems.
func applyPromotionToTargetItems(targetItems []EligibleItem, itemIDPromotionAmount *bnMap, applied map[string]any, promotion *dal.Record, itemsMap map[string]*dal.Record, config buyGetConfig, usage *dal.Record) ([]*dal.Record, *totals.BN) {
	var actions []*dal.Record
	appliedQuantity := totals.Convert(0)
	remainingQty := totals.Convert(config.targetApplyQuantity)
	am := crec(promotion, "application_method")
	inclusive := ctruthy(cget(promotion, "is_tax_inclusive"))
	for _, target := range targetItems {
		if totals.Lte(remainingQty, 0) {
			break
		}
		item := itemsMap[target.ItemID]
		id := cstr(item, "id")
		var appliedPromoValue any = 0
		if v, ok := applied[id]; ok && v != nil {
			appliedPromoValue = v
		}
		multiplier := totals.Min(target.Quantity, remainingQty)
		base := cget(item, "subtotal")
		if inclusive {
			base = cget(item, "original_total")
		}
		pricePerUnit := totals.Div(base, cget(item, "quantity"))
		applicableAmount := totals.Mult(pricePerUnit, multiplier)
		var amount *totals.BN
		if cstr(am, "type") == "fixed" {
			// Fixed: value per unit, capped at the item price.
			amount = totals.Min(totals.Mult(config.promotionValue, multiplier), applicableAmount)
		} else {
			amount = totals.Div(totals.Mult(applicableAmount, config.promotionValue), 100)
		}
		if totals.Lte(amount, 0) {
			continue
		}
		remainingQty = totals.Sub(remainingQty, multiplier)
		if a := computeActionForBudgetExceeded(promotion, amount, usage); a != nil {
			actions = append(actions, a)
			continue
		}
		var appliedExcl any = amount
		if inclusive {
			appliedExcl = totals.Mult(amount, totals.Div(cget(item, "subtotal"), cget(item, "original_total")))
		}
		// methodIdPromoValueMap stores a JS number (toNumber()).
		applied[id] = number(totals.Add(appliedPromoValue, appliedExcl))
		itemIDPromotionAmount.set(id, totals.Add(itemIDPromotionAmount.getOr(id), amount))
		appliedQuantity = totals.Add(appliedQuantity, multiplier)
	}
	return actions, appliedQuantity
}

// updateEligibleItemQuantities mirrors updateEligibleItemQuantities.
func updateEligibleItemQuantities(remainingBuy, remainingTarget *bnMap, application buyGetApplication) {
	consumed := newBNMap()
	for _, b := range application.buyItems {
		consumed.set(b.ItemID, totals.Add(consumed.getOr(b.ItemID), b.Quantity))
	}
	for _, t := range application.targetItems {
		consumed.set(t.ItemID, totals.Add(consumed.getOr(t.ItemID), t.Quantity))
	}
	for _, id := range consumed.keys {
		q := consumed.values[id]
		if remainingBuy.has(id) {
			remainingBuy.set(id, totals.Sub(remainingBuy.getOr(id), q))
		}
		if remainingTarget.has(id) {
			remainingTarget.set(id, totals.Sub(remainingTarget.getOr(id), q))
		}
	}
}

// eligibleItemsMap is a JS Map<string, EligibleItem> (insertion ordered).
type eligibleItemsMap struct {
	keys   []string
	values map[string]*EligibleItem
}

func newEligibleItemsMap() *eligibleItemsMap {
	return &eligibleItemsMap{values: map[string]*EligibleItem{}}
}

// update mirrors updateEligibleItems.
func (m *eligibleItemsMap) update(items []EligibleItem) {
	for _, item := range items {
		if existing, ok := m.values[item.ItemID]; ok {
			existing.Quantity = jsNumberBN(totals.Add(existing.Quantity, item.Quantity))
			continue
		}
		c := item
		m.keys = append(m.keys, item.ItemID)
		m.values[item.ItemID] = &c
	}
}

func (m *eligibleItemsMap) list() []EligibleItem {
	out := make([]EligibleItem, 0, len(m.keys))
	for _, k := range m.keys {
		out = append(out, *m.values[k])
	}
	return out
}

// filterItemsByPromotionRules mirrors filterItemsByPromotionRules.
func filterItemsByPromotionRules(items []*dal.Record, rules []*dal.Record) []*dal.Record {
	out := []*dal.Record{}
	for _, item := range items {
		if areRulesValidForContext(rules, item, "items") {
			out = append(out, item)
		}
	}
	buyGetSortByUnitPrice(out, false)
	return out
}

// getComputedActionsForBuyGet mirrors getComputedActionsForBuyGet.
func getComputedActionsForBuyGet(promotion *dal.Record, items []*dal.Record, itemsDefined bool, applied map[string]any, eligibleBuyItemMap, eligibleTargetItemMap *eligibleItemsByCode, usage *dal.Record) ([]*dal.Record, error) {
	valid, err := isValidBuyGetPromotionContext(promotion, items, itemsDefined)
	if err != nil || !valid {
		return nil, err
	}
	config := normalizePromotionApplicationConfiguration(promotion)
	itemsMap := map[string]*dal.Record{}
	for _, item := range items {
		itemsMap[cstr(item, "id")] = item
	}
	am := crec(promotion, "application_method")
	code := cstr(promotion, "code")
	eligibleBuy := filterItemsByPromotionRules(items, crecords(cget(am, "buy_rules")))
	eligibleTarget := filterItemsByPromotionRules(items, crecords(cget(am, "target_rules")))
	remainingBuy := calculateRemainingQuantities(eligibleBuy, eligibleBuyItemMap, code)
	remainingTarget := calculateRemainingQuantities(eligibleTarget, eligibleTargetItemMap, code)
	targetItemsInDiscountOrder := sortTargetItemsByStrategy(eligibleTarget, cstr(am, "target_selection_strategy"))

	totalBuy := newEligibleItemsMap()
	totalTarget := newEligibleItemsMap()
	itemIDPromotionAmount := newBNMap()
	var actions []*dal.Record

	const maxPromotionIterations = 1000
	iteration := 0
	appliedQuantity := totals.Convert(0)
	for {
		iteration++
		if iteration > maxPromotionIterations {
			slog.Warn(fmt.Sprintf("Buy-get promotion %s exceeded maximum iterations (%d). Breaking loop to prevent infinite execution.", code, maxPromotionIterations))
			break
		}
		state := preparePromotionApplicationState(eligibleBuy, targetItemsInDiscountOrder, remainingBuy, remainingTarget, config, appliedQuantity)
		if !state.isValid {
			break
		}
		applicationActions, quantity := applyPromotionToTargetItems(state.targetItems, itemIDPromotionAmount, applied, promotion, itemsMap, config, usage)
		actions = append(actions, applicationActions...)
		appliedQuantity = totals.Add(appliedQuantity, quantity)
		updateEligibleItemQuantities(remainingBuy, remainingTarget, state)
		totalBuy.update(state.buyItems)
		totalTarget.update(state.targetItems)
	}

	// createComputedActionsFromPromotionApplication: is_tax_inclusive is only
	// set when the promotion is tax inclusive.
	inclusive := ctruthy(cget(promotion, "is_tax_inclusive"))
	for _, id := range itemIDPromotionAmount.keys {
		total := itemIDPromotionAmount.values[id]
		if !totals.Gt(total, 0) {
			continue
		}
		a := action("action", "addItemAdjustment", "item_id", id, "amount", total, "code", cget(promotion, "code"))
		if inclusive {
			a.Set("is_tax_inclusive", true)
		}
		actions = append(actions, a)
	}
	eligibleBuyItemMap.set(code, totalBuy.list())
	eligibleTargetItemMap.set(code, totalTarget.list())
	return actions, nil
}

// sortByBuyGetType mirrors sortByBuyGetType (a comparator: buy-get first,
// then application_method.value descending; buy-get ties by
// buy_rules_min_quantity and apply_to_quantity descending).
func sortByBuyGetType(a, b *dal.Record) int {
	aBuyGet, bBuyGet := cstr(a, "type") == "buyget", cstr(b, "type") == "buyget"
	switch {
	case aBuyGet && !bBuyGet:
		return -1
	case !aBuyGet && bBuyGet:
		return 1
	case cstr(a, "type") != cstr(b, "type"):
		return 0
	}
	keys := []string{"value"}
	if aBuyGet {
		keys = append(keys, "buy_rules_min_quantity", "apply_to_quantity")
	}
	for _, k := range keys {
		av := number(cget(crec(a, "application_method"), k))
		bv := number(cget(crec(b, "application_method"), k))
		if av < bv {
			return 1
		}
		if av > bv {
			return -1
		}
	}
	return 0
}
