package storecredit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// Transaction types (TransactionType).
const (
	TransactionTypeCredit = "credit"
	TransactionTypeDebit  = "debit"
)

// StoreCreditModuleService mirrors service.ts (StoreCreditService):
// MedusaService({ StoreCreditAccount, AccountTransaction }) with the
// account balance methods.
type StoreCreditModuleService struct {
	*utilsmodulessdk.MedusaService
}

func invalidData(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

// bnNumber mirrors MathBN.convert(x).toNumber().
func bnNumber(v any) float64 {
	f, _ := strconv.ParseFloat(totals.Convert(v).Text('f'), 64)
	return f
}

// ListBySuffix mirrors the listStoreCreditAccounts override: every
// account carries its balance, credits and debits (appended after the
// serialized fields). Query lists through it; listAndCount is not
// overridden upstream.
func (s *StoreCreditModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	accounts, err := s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
	if err != nil || suffix != "StoreCreditAccounts" {
		return accounts, err
	}
	for _, account := range accounts {
		id, _ := account.Get("id")
		stats, err := retrieveAccountStats(ctx, s.MedusaService, fmt.Sprint(id))
		if err != nil {
			return nil, err
		}
		for _, k := range []string{"balance", "credits", "debits"} {
			v, _ := stats.Get(k)
			account.Set(k, v)
		}
	}
	return accounts, nil
}

// Call dispatches the overridden and declared methods by name before the
// generated ones.
func (s *StoreCreditModuleService) Call(ctx context.Context, method string, args []any) (any, error) {
	arg := func(i int) any {
		if i < len(args) {
			return args[i]
		}
		return nil
	}
	switch method {
	case "listStoreCreditAccounts":
		filters, _ := arg(0).(*dal.Record)
		config, err := utilsmodulessdk.ParseFindConfig(arg(1))
		if err != nil {
			return nil, err
		}
		return s.ListBySuffix(ctx, "StoreCreditAccounts", filters, config)
	case "updateStoreCreditAccounts":
		return s.UpdateStoreCreditAccounts(ctx, arg(0))
	case "deleteTransactions":
		return nil, s.DeleteTransactions(ctx, arg(0))
	case "retrieveAccountStats":
		r, _ := arg(0).(*dal.Record)
		id, _ := r.Get("account_id")
		return s.RetrieveAccountStats(ctx, fmt.Sprint(id))
	case "creditAccounts":
		return s.CreditAccounts(ctx, records(arg(0)))
	case "debitAccounts":
		return s.DebitAccounts(ctx, records(arg(0)))
	}
	return s.MedusaService.Call(ctx, method, args)
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case *dal.Record:
		return []*dal.Record{t}
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// CreateStoreCreditAccounts is the generated createStoreCreditAccounts.
func (s *StoreCreditModuleService) CreateStoreCreditAccounts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	list := make([]any, len(data))
	for i, d := range data {
		list[i] = d
	}
	out, err := s.MedusaService.Call(ctx, "createStoreCreditAccounts", []any{list})
	if err != nil {
		return nil, err
	}
	return records(out), nil
}

// DeleteStoreCreditAccounts is the generated deleteStoreCreditAccounts.
func (s *StoreCreditModuleService) DeleteStoreCreditAccounts(ctx context.Context, ids any) error {
	_, err := s.MedusaService.Call(ctx, "deleteStoreCreditAccounts", []any{ids})
	return err
}

// UpdateStoreCreditAccounts mirrors the updateStoreCreditAccounts
// override. Its whitelist check is inverted upstream: a key in ["id",
// "metadata"] throws, so every update (which carries an id) is rejected
// with a plain Error.
func (s *StoreCreditModuleService) UpdateStoreCreditAccounts(ctx context.Context, data any) (any, error) {
	list := records(data)
	for _, account := range list {
		for _, key := range account.Keys() {
			if key == "id" || key == "metadata" {
				return nil, fmt.Errorf("Field %s is not allowed to be updated", key)
			}
		}
	}
	items := make([]any, len(list))
	for i, d := range list {
		items[i] = d
	}
	return s.MedusaService.Call(ctx, "updateStoreCreditAccounts", []any{items})
}

// DeleteTransactions mirrors deleteTransactions: a forwarder to the
// generated deleteAccountTransactions.
func (s *StoreCreditModuleService) DeleteTransactions(ctx context.Context, ids any) error {
	_, err := s.MedusaService.Call(ctx, "deleteAccountTransactions", []any{ids})
	return err
}

// RetrieveAccountStats mirrors retrieveAccountStats({ account_id }): the
// record { id, balance, debits, credits } (JavaScript numbers).
func (s *StoreCreditModuleService) RetrieveAccountStats(ctx context.Context, accountID string) (*dal.Record, error) {
	return retrieveAccountStats(ctx, s.MedusaService, accountID)
}

func retrieveAccountStats(ctx context.Context, m *utilsmodulessdk.MedusaService, accountID string) (*dal.Record, error) {
	account, err := m.Internal("StoreCreditAccount").Retrieve(ctx, accountID, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	id, _ := account.Get("id")
	rows, err := dal.QueryRows(ctx, m.DB(), `select "account"."id" as "id", `+
		`SUM(CASE WHEN at.type = 'credit' THEN at.amount ELSE 0 END) as "credits", `+
		`SUM(CASE WHEN at.type = 'debit' THEN at.amount ELSE 0 END) as "debits", `+
		`SUM(CASE WHEN at.type = 'credit' THEN at.amount::numeric ELSE -at.amount::numeric END) as "balance" `+
		`from "store_credit_account" as "account" `+
		`left join "store_credit_account_transaction" as "at" on "at"."account_id" = "account"."id" and "at"."deleted_at" is null `+
		`where "account"."id" = $1 group by "account"."id"`, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		// accountBalances[0] is undefined upstream
		return nil, errors.New("Cannot read properties of undefined (reading 'balance')")
	}
	row := rows[0]
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("balance", bnNumber(row["balance"]))
	out.Set("debits", bnNumber(row["debits"]))
	out.Set("credits", bnNumber(row["credits"]))
	return out, nil
}

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0
	}
	return true
}

// transactionData builds the account transaction to create: undefined
// reference fields are left out.
func transactionData(accountID any, amount any, kind string, data *dal.Record) *dal.Record {
	t := dal.NewRecord()
	t.Set("account_id", accountID)
	t.Set("amount", amount)
	t.Set("type", kind)
	for _, k := range []string{"reference", "reference_id", "note"} {
		if v, ok := data.Get(k); ok {
			t.Set(k, v)
		}
	}
	return t
}

// CreditAccounts mirrors creditAccounts (creditAccounts_ in one
// transaction): one credit transaction per input.
func (s *StoreCreditModuleService) CreditAccounts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var transactions []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		for _, d := range data {
			accountID := get(d, "account_id")
			if !truthy(accountID) {
				return invalidData("Account ID is required")
			}
			amount := get(d, "amount")
			if amount == nil {
				return invalidData("Amount is required")
			}
			account, err := m.Internal("StoreCreditAccount").Retrieve(ctx, accountID, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			created, err := m.Internal("AccountTransaction").Create(ctx, []*dal.Record{transactionData(get(account, "id"), amount, TransactionTypeCredit, d)})
			if err != nil {
				return err
			}
			transactions = append(transactions, created...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if transactions == nil {
		transactions = []*dal.Record{}
	}
	return transactions, nil
}

// lockAccountsForUpdate mirrors lockAccountsForUpdate_: the account rows
// are locked (sorted ids) for the surrounding transaction.
func lockAccountsForUpdate(ctx context.Context, m *utilsmodulessdk.MedusaService, accountIDs []string) error {
	seen := map[string]bool{}
	var ids []string
	for _, id := range accountIDs {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return nil
	}
	args := make([]any, len(ids))
	sql := `select "id" from "store_credit_account" where "id" in (`
	for i, id := range ids {
		if i > 0 {
			sql += ", "
		}
		sql += "$" + strconv.Itoa(i+1)
		args[i] = id
	}
	sql += ") for update"
	rows, err := m.DB().Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}

// DebitAccounts mirrors debitAccounts (debitAccounts_ in one
// transaction): the accounts are locked, then every debit checks the
// balance before it is inserted.
func (s *StoreCreditModuleService) DebitAccounts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var transactions []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		var ids []string
		for _, d := range data {
			if id := get(d, "account_id"); truthy(id) {
				ids = append(ids, fmt.Sprint(id))
			}
		}
		if err := lockAccountsForUpdate(ctx, m, ids); err != nil {
			return err
		}
		for _, d := range data {
			accountID := get(d, "account_id")
			amount := get(d, "amount")
			amountBN := totals.Convert(amount)
			if !truthy(accountID) {
				return invalidData("Account ID is required")
			}
			if amount == nil {
				return invalidData("Amount is required")
			}
			if totals.Lte(amountBN, 0) {
				return invalidData("Amount must be greater than 0")
			}
			stats, err := retrieveAccountStats(ctx, m, fmt.Sprint(accountID))
			if err != nil {
				return err
			}
			if totals.Lt(get(stats, "balance"), amountBN) {
				return invalidData("Insufficient balance")
			}
			created, err := m.Internal("AccountTransaction").Create(ctx, []*dal.Record{transactionData(get(stats, "id"), amountBN, TransactionTypeDebit, d)})
			if err != nil {
				return err
			}
			transactions = append(transactions, created...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if transactions == nil {
		transactions = []*dal.Record{}
	}
	return transactions, nil
}
