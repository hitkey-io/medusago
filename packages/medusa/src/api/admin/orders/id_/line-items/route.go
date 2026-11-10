// Package lineitems mirrors api/admin/orders/[id]/line-items.
package lineitems

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func num(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	}
	return 0
}

// GET mirrors GET /admin/orders/:id/line-items.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("order_id", req.Params["id"])
	fields := append([]string{}, req.QueryConfig.Fields...)
	for _, f := range []string{"item_id", "version"} {
		found := false
		for _, x := range fields {
			if x == f {
				found = true
			}
		}
		if !found {
			fields = append(fields, f)
		}
	}
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "order_items", Filters: filters, Fields: fields}, joiner.Options{})
	if err != nil {
		return err
	}
	var order []string
	items := map[string]*dal.Record{}
	history := map[string]*dal.Record{}
	for _, row := range r.Data {
		item, _ := row.(*dal.Record)
		if item == nil {
			continue
		}
		itemID, _ := item.Get("item_id")
		key := fmt.Sprint(itemID)
		version, _ := item.Get("version")
		if _, ok := items[key]; !ok {
			v := dal.NewRecord()
			v.Set("from", version)
			v.Set("to", version)
			h := dal.NewRecord()
			h.Set("version", v)
			history[key] = h
			items[key] = withHistory(item, h)
			order = append(order, key)
			continue
		}
		v, _ := history[key].Get("version")
		vr := v.(*dal.Record)
		to, _ := vr.Get("to")
		from, _ := vr.Get("from")
		if num(version) > num(to) {
			vr.Set("to", version)
		}
		if num(version) < num(from) {
			vr.Set("from", version)
		}
		current, _ := items[key].Get("version")
		if num(version) > num(current) {
			items[key] = withHistory(item, history[key])
		}
	}
	list := []any{}
	for _, k := range order {
		list = append(list, items[k])
	}
	out := dal.NewRecord()
	out.Set("order_items", list)
	return res.JSON(out)
}

func withHistory(item, h *dal.Record) *dal.Record {
	out := dal.NewRecord()
	for _, k := range item.Keys() {
		v, _ := item.Get(k)
		out.Set(k, v)
	}
	out.Set("history", h)
	return out
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
