package store

import (
	"context"
	"testing"

	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type memStore struct {
	rows map[string]composer.ExecutionRow
}

func (m *memStore) SaveExecution(_ context.Context, r composer.ExecutionRow) error {
	m.rows[r.TransactionID] = r
	return nil
}
func (m *memStore) DeleteExecution(_ context.Context, _, tx, _ string) error {
	delete(m.rows, tx)
	return nil
}

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestStoredExecution(t *testing.T) {
	st := &memStore{rows: map[string]composer.ExecutionRow{}}
	c := container{utilsmodulessdk.WORKFLOW_ENGINE: st}
	if err := RunWithTransactionID(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	row, ok := st.rows["hello-123"]
	if !ok || row.State != "done" || row.RetentionTime == nil || *row.RetentionTime != 99999 {
		t.Fatalf("%+v", row)
	}
}
