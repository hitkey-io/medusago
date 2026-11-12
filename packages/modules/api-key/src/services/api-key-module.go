// Package services mirrors @medusajs/api-key/src/services.
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/api-key/src/models"
	"golang.org/x/crypto/scrypt"
)

// ApiKeyModuleService mirrors ApiKeyModuleService.
type ApiKeyModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewApiKeyModuleService builds the service.
func NewApiKeyModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*ApiKeyModuleService, error) {
	return &ApiKeyModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

type token struct {
	raw, hashed, salt, redacted string
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// redactKey mirrors redactKey (String.substring clamps its bounds).
func redactKey(key string) string {
	head := key
	if len(head) > 6 {
		head = head[:6]
	}
	tail := ""
	if len(key) > 3 {
		tail = key[len(key)-3:]
	} else {
		tail = key
	}
	return head + "***" + tail
}

// Authenticate mirrors authenticate: the unrevoked secret keys (revoked_at
// unset or in the future) with the token's redacted form are hashed with
// their salt; the first one matching the token is returned (nil when none,
// upstream false).
func (s *ApiKeyModuleService) Authenticate(ctx context.Context, tok string) (*dal.Record, error) {
	now := dal.NewRecord()
	now.Set("$gt", time.Now())
	notRevoked := dal.NewRecord()
	notRevoked.Set("revoked_at", nil)
	revokedLater := dal.NewRecord()
	revokedLater.Set("revoked_at", now)
	f := dal.NewRecord()
	f.Set("type", "secret")
	f.Set("redacted", redactKey(tok))
	f.Set("$or", []any{notRevoked, revokedLater})
	keys, err := s.Internal("ApiKey").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		salt, _ := get(k, "salt").(string)
		hashed, err := CalculateHash(tok, salt)
		if err != nil {
			return nil, err
		}
		if hashed == get(k, "token") {
			return k, nil
		}
	}
	return nil, nil
}

func generatePublishableKey() token {
	t := "pk_" + randomHex(32)
	return token{raw: t, hashed: t, salt: "", redacted: redactKey(t)}
}

// CalculateHash mirrors calculateHash (scrypt, 64 bytes, hex; Node defaults
// N=16384, r=8, p=1).
func CalculateHash(tok, salt string) (string, error) {
	b, err := scrypt.Key([]byte(tok), []byte(salt), 16384, 8, 1, 64)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateSecretKey() (token, error) {
	t := "sk_" + randomHex(32)
	salt := randomHex(16)
	h, err := CalculateHash(t, salt)
	if err != nil {
		return token{}, err
	}
	return token{raw: t, hashed: h, salt: salt, redacted: redactKey(t)}, nil
}

// omitToken mirrors omitToken: secret tokens are blanked and the salt
// removed.
func omitToken(r *dal.Record) *dal.Record {
	if r == nil {
		return r
	}
	if _, has := r.Get("token"); has && get(r, "type") == "secret" {
		r.Set("token", "")
	}
	r.Delete("salt")
	return r
}

// ListBySuffix applies omitToken to the listed api keys.
func (s *ApiKeyModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	rows, err := s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
	if err != nil || suffix != "ApiKeys" {
		return rows, err
	}
	for _, r := range rows {
		omitToken(r)
	}
	return rows, nil
}

// ListAndCountBySuffix applies omitToken to the listed api keys.
func (s *ApiKeyModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	rows, n, err := s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
	if err != nil || suffix != "ApiKeys" {
		return rows, n, err
	}
	for _, r := range rows {
		omitToken(r)
	}
	return rows, n, nil
}

// CreateApiKeys mirrors createApiKeys: the raw token is returned once.
func (s *ApiKeyModuleService) CreateApiKeys(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	input := make([]*dal.Record, len(data))
	tokens := make([]token, len(data))
	for i, key := range data {
		var t token
		if get(key, "type") == "publishable" {
			t = generatePublishableKey()
		} else {
			var err error
			t, err = generateSecretKey()
			if err != nil {
				return nil, err
			}
		}
		tokens[i] = t
		n := dal.NewRecord()
		for _, k := range key.Keys() {
			v, _ := key.Get(k)
			n.Set(k, v)
		}
		n.Set("token", t.hashed)
		n.Set("salt", t.salt)
		n.Set("redacted", t.redacted)
		input[i] = n
	}
	created, err := s.Internal("ApiKey").Create(ctx, input)
	if err != nil {
		return nil, err
	}
	for _, c := range created {
		hashed := fmt.Sprint(get(c, "token"))
		for _, t := range tokens {
			if t.hashed == hashed {
				c.Set("token", t.raw)
			}
		}
		c.Delete("salt")
	}
	return created, nil
}

func (s *ApiKeyModuleService) normalize(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	withID := func(id any) *dal.Record {
		r := dal.NewRecord()
		r.Set("id", id)
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			r.Set(k, v)
		}
		return r
	}
	if id, ok := idOrSelector.(string); ok {
		return []*dal.Record{withID(id)}, nil
	}
	sel, _ := idOrSelector.(*dal.Record)
	keys, err := s.Internal("ApiKey").List(ctx, sel, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	out := make([]*dal.Record, len(keys))
	for i, k := range keys {
		out[i] = withID(get(k, "id"))
	}
	return out, nil
}

func rows(res any) []*dal.Record {
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}
	}
	l, _ := res.([]*dal.Record)
	return l
}

// UpdateApiKeys mirrors updateApiKeys (only the title is updatable).
func (s *ApiKeyModuleService) UpdateApiKeys(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	in, err := s.normalize(ctx, idOrSelector, data)
	if err != nil {
		return nil, err
	}
	return s.updateApiKeys(ctx, in)
}

func (s *ApiKeyModuleService) updateApiKeys(ctx context.Context, in []*dal.Record) ([]*dal.Record, error) {
	if len(in) == 0 {
		return []*dal.Record{}, nil
	}
	req := make([]any, len(in))
	for i, k := range in {
		r := dal.NewRecord()
		r.Set("id", get(k, "id"))
		r.Set("title", get(k, "title"))
		req[i] = r
	}
	res, err := s.Internal("ApiKey").Update(ctx, req)
	if err != nil {
		return nil, err
	}
	out := rows(res)
	for _, r := range out {
		omitToken(r)
	}
	return out, nil
}

// UpsertApiKeys mirrors upsertApiKeys (updates only, as used by the
// compensations).
func (s *ApiKeyModuleService) UpsertApiKeys(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var forUpdate, forCreate []*dal.Record
	for _, d := range data {
		if id := get(d, "id"); id != nil && id != "" {
			forUpdate = append(forUpdate, d)
		} else {
			forCreate = append(forCreate, d)
		}
	}
	var out []*dal.Record
	if len(forCreate) > 0 {
		c, err := s.CreateApiKeys(ctx, forCreate)
		if err != nil {
			return nil, err
		}
		out = append(out, c...)
	}
	if len(forUpdate) > 0 {
		u, err := s.updateApiKeys(ctx, forUpdate)
		if err != nil {
			return nil, err
		}
		out = append(out, u...)
	}
	return out, nil
}

// Revoke mirrors revoke.
func (s *ApiKeyModuleService) Revoke(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	in, err := s.normalize(ctx, idOrSelector, data)
	if err != nil {
		return nil, err
	}
	if len(in) == 0 {
		return []*dal.Record{}, nil
	}
	var out []*dal.Record
	err = s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ApiKeyModuleService{MedusaService: tx}
		for _, k := range in {
			if id := get(k, "id"); id == nil || id == "" {
				return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "You must provide an api key id field when revoking a key.")
			}
			if by := get(k, "revoked_by"); by == nil || by == "" {
				return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "You must provide a revoked_by field when revoking a key.")
			}
		}
		ids := make([]any, len(in))
		for i, k := range in {
			ids[i] = get(k, "id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		f.Set("type", "secret")
		lt := dal.NewRecord()
		lt.Set("$lt", time.Now().UTC())
		f.Set("revoked_at", lt)
		revoked, err := t.Internal("ApiKey").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		if len(revoked) > 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("There are %d secret keys that are already revoked.", len(revoked)))
		}
		req := make([]any, len(in))
		for i, k := range in {
			at := time.Now().UTC()
			if secs, ok := get(k, "revoke_in").(float64); ok && secs > 0 {
				at = at.Add(time.Duration(secs) * time.Second)
			}
			r := dal.NewRecord()
			r.Set("id", get(k, "id"))
			r.Set("revoked_at", at)
			r.Set("revoked_by", get(k, "revoked_by"))
			req[i] = r
		}
		res, err := t.Internal("ApiKey").Update(ctx, req)
		if err != nil {
			return err
		}
		out = rows(res)
		return nil
	})
	for _, r := range out {
		omitToken(r)
	}
	return out, err
}

// DeleteApiKeys mirrors deleteApiKeys: only revoked keys can be deleted.
func (s *ApiKeyModuleService) DeleteApiKeys(ctx context.Context, ids []any) error {
	f := dal.NewRecord()
	f.Set("id", ids)
	eq := dal.NewRecord()
	eq.Set("$eq", nil)
	gt := dal.NewRecord()
	gt.Set("$gt", time.Now().UTC())
	a := dal.NewRecord()
	a.Set("revoked_at", eq)
	b := dal.NewRecord()
	b.Set("revoked_at", gt)
	f.Set("$or", []any{a, b})
	unrevoked, err := s.Internal("ApiKey").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
	if err != nil {
		return err
	}
	if len(unrevoked) > 0 {
		list := make([]string, len(unrevoked))
		for i, k := range unrevoked {
			list[i] = fmt.Sprint(get(k, "id"))
		}
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Cannot delete api keys that are not revoked - "+strings.Join(list, ", "))
	}
	_, err = s.Internal("ApiKey").Delete(ctx, ids)
	return err
}
