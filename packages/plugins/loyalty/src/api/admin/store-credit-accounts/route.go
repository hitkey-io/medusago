package storecreditaccounts

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storecredit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

// Service resolves the store credit module service from a request scope.
func Service(req *medusahttp.MedusaRequest) *storecredit.StoreCreditModuleService {
	s, _ := req.Scope.Resolve(storecredit.ServiceName).(*storecredit.StoreCreditModuleService)
	return s
}

// WithAccountStats mirrors { ...account, ...accountStats } for every account
// (retrieveAccountStats per account).
func WithAccountStats(req *medusahttp.MedusaRequest, accounts []any) ([]any, error) {
	out := make([]any, 0, len(accounts))
	for _, a := range accounts {
		account, _ := a.(*dal.Record)
		id, _ := account.Get("id")
		merged, err := MergeAccountStats(req, account, fmt.Sprint(id))
		if err != nil {
			return nil, err
		}
		out = append(out, merged)
	}
	return out, nil
}

// MergeAccountStats mirrors { ...account, ...retrieveAccountStats({
// account_id }) }.
func MergeAccountStats(req *medusahttp.MedusaRequest, account *dal.Record, accountID string) (*dal.Record, error) {
	stats, err := Service(req).RetrieveAccountStats(req.Ctx(), accountID)
	if err != nil {
		return nil, err
	}
	merged := dal.NewRecord()
	if account != nil {
		for _, k := range account.Keys() {
			v, _ := account.Get(k)
			merged.Set(k, v)
		}
	}
	for _, k := range stats.Keys() {
		v, _ := stats.Get(k)
		merged.Set(k, v)
	}
	return merged, nil
}

// RetrieveAccount mirrors query.graph({ entity, fields, filters }, {
// throwIfKeyNotFound: true }).data[0].
func RetrieveAccount(req *medusahttp.MedusaRequest, entity string, filters *dal.Record) (any, error) {
	res, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: entity, IDOrFilter: filters, Fields: req.QueryConfig.Fields, Options: joiner.Options{ThrowIfKeyNotFound: true},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	return res.Data[0], nil
}

func accountResponse(res *medusahttp.MedusaResponse, account any) error {
	out := dal.NewRecord()
	out.Set("store_credit_account", account)
	return res.JSON(out)
}

// GET mirrors GET /admin/store-credit-accounts.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "store_credit_account", IDOrFilter: req.FilterableFields, Fields: req.QueryConfig.Fields, Pagination: req.QueryConfig.Pagination,
	})
	if err != nil {
		return err
	}
	accounts, err := WithAccountStats(req, r.Data)
	if err != nil {
		return err
	}
	r.Data = accounts
	return res.JSON(httputils.ListResponse("store_credit_accounts", r))
}

// POST mirrors POST /admin/store-credit-accounts.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	input := dal.NewRecord()
	if body != nil {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			input.Set(k, v)
		}
	}
	accounts, err := scworkflows.CreateStoreCreditAccountsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{input})
	if err != nil {
		return err
	}
	var id any
	if len(accounts) > 0 {
		id, _ = accounts[0].Get("id")
	}
	f := dal.NewRecord()
	f.Set("id", id)
	account, err := RetrieveAccount(req, "store_credit_accounts", f)
	if err != nil {
		return err
	}
	return accountResponse(res, account)
}

// AccountResponse writes { store_credit_account }.
func AccountResponse(res *medusahttp.MedusaResponse, account any) error {
	return accountResponse(res, account)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
