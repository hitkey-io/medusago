package utils

import (
	"fmt"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// normalizeQuery mirrors normalizeQuery.
func normalizeQuery(q *dal.Record) (*dal.Record, error) {
	acc := dal.NewRecord()
	if q == nil {
		return acc, nil
	}
	for _, key := range q.Keys() {
		val, _ := q.Get(key)
		normalized := val
		if arr, ok := val.([]any); ok && len(arr) == 1 {
			if s, ok := arr[0].(string); ok {
				parts := strings.Split(s, ",")
				out := make([]any, len(parts))
				for i, p := range parts {
					out[i] = p
				}
				normalized = out
			}
		}
		if strings.Contains(key, ".") {
			segs := strings.Split(key, ".")
			if len(segs) > 2 {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, fmt.Sprintf("Key accessor more than 2 levels deep: %s", key))
			}
			parent, child := segs[0], segs[1]
			cur, _ := acc.Get(parent)
			obj := dal.NewRecord()
			if r, ok := cur.(*dal.Record); ok {
				for _, k := range r.Keys() {
					v, _ := r.Get(k)
					obj.Set(k, v)
				}
			}
			obj.Set(child, normalized)
			acc.Set(parent, obj)
			continue
		}
		acc.Set(key, normalized)
	}
	return acc, nil
}

func getFilterableFields(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	for _, k := range r.Keys() {
		switch k {
		case "limit", "offset", "fields", "order":
			continue
		}
		v, _ := r.Get(k)
		if zod.IsUndefined(v) {
			continue
		}
		out.Set(k, removeUndefinedDeeply(v))
	}
	return out
}

// removeUndefinedDeeply mirrors removeUndefinedDeeply of
// removeUndefinedProperties: "null" strings become null.
func removeUndefinedDeeply(v any) any {
	switch t := v.(type) {
	case string:
		if t == "null" {
			return nil
		}
		return t
	case []any:
		out := make([]any, 0, len(t))
		for _, x := range t {
			if zod.IsUndefined(x) {
				continue
			}
			out = append(out, removeUndefinedDeeply(x))
		}
		return out
	case *dal.Record:
		if t == nil {
			return t
		}
		out := dal.NewRecord()
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			if zod.IsUndefined(x) {
				continue
			}
			out.Set(k, removeUndefinedDeeply(x))
		}
		return out
	}
	return v
}

// GetRelationsDepth mirrors getRelationsDepth.
func GetRelationsDepth(field string) int {
	isStar := strings.HasPrefix(field, "*") || strings.HasSuffix(field, ".*")
	f := field
	if strings.HasPrefix(f, "*") {
		f = f[1:]
	} else {
		f = strings.TrimSuffix(f, ".*")
	}
	n := len(strings.Split(f, "."))
	if isStar {
		return n
	}
	return n - 1
}

// ValidateRelationsLimit mirrors validateRelationsLimit.
func ValidateRelationsLimit(fields []string, limit int) error {
	var invalid []string
	for _, f := range fields {
		if GetRelationsDepth(f) > limit {
			invalid = append(invalid, f)
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("The following fields expand more than the maximum of %d allowed relations: %s", limit, strings.Join(invalid, ", ")))
}

// ValidateAndTransformQuery mirrors validateAndTransformQuery.
func ValidateAndTransformQuery(schema *zod.Schema, qc QueryConfig) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, _ *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		restricted := req.RestrictedFields
		allowed := append([]string{}, qc.Allowed...)
		custom := req.Allowed
		if len(custom) == 0 {
			if c, ok := req.Locals["consumedAllowedFields"].([]string); ok {
				custom = c
			}
		}
		if len(custom) > 0 {
			allowed = append(allowed, custom...)
			req.Locals["consumedAllowedFields"] = custom
		}
		req.Allowed = []string{}
		disallowed := req.Disallowed
		if disallowed == nil {
			disallowed = qc.Disallowed
		}
		req.Disallowed = nil

		query, err := normalizeQuery(req.Query)
		if err != nil {
			return err
		}
		out, err := zod.ZodValidator(schema, query)
		if err != nil {
			return err
		}
		validated, _ := out.(*dal.Record)
		if validated == nil {
			validated = dal.NewRecord()
		}
		cfg := qc
		cfg.Allowed = allowed
		cfg.Disallowed = disallowed
		var prepared *PreparedQuery
		if qc.IsList {
			prepared, err = PrepareListQuery(validated, cfg, restricted)
		} else {
			prepared, err = PrepareRetrieveQuery(validated, cfg, restricted)
		}
		if err != nil {
			return err
		}
		if req.StoreRelationsLimit != 0 {
			limit := req.StoreRelationsLimit
			if qc.StoreRelationsLimit != 0 {
				limit = qc.StoreRelationsLimit
			}
			if err := ValidateRelationsLimit(prepared.RemoteQueryConfig.Fields, limit); err != nil {
				return err
			}
		}
		vq := dal.NewRecord()
		for _, k := range validated.Keys() {
			if k == "with_deleted" {
				continue
			}
			v, _ := validated.Get(k)
			vq.Set(k, v)
		}
		req.ValidatedQuery = vq
		req.FilterableFields = getFilterableFields(vq)
		req.QueryConfig = prepared.RemoteQueryConfig
		req.RemoteQueryConfig = req.QueryConfig
		req.ListConfig = prepared.ListConfig
		req.RetrieveConfig = prepared.RetrieveConfig
		next(nil)
		return nil
	}
}

// ValidateAndTransformBody mirrors validateAndTransformBody.
func ValidateAndTransformBody(schema func(req *medusahttp.MedusaRequest) *zod.Schema) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, _ *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		out, err := zod.ZodValidator(schema(req), req.Body)
		if err != nil {
			return err
		}
		req.ValidatedBody = out
		next(nil)
		return nil
	}
}

// Schema wraps a static schema for ValidateAndTransformBody.
func Schema(s *zod.Schema) func(*medusahttp.MedusaRequest) *zod.Schema {
	return func(*medusahttp.MedusaRequest) *zod.Schema { return s }
}
