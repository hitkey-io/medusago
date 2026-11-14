package locks

import (
	"context"
	"testing"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

type fakeLocking struct{ log []string }

func (f *fakeLocking) Execute(ctx context.Context, keys []string, job lockingtypes.Job, _ lockingtypes.ExecuteArgs) (any, error) {
	f.log = append(f.log, "acquire "+keys[0])
	out, err := job(ctx)
	f.log = append(f.log, "release "+keys[0])
	return out, err
}
func (f *fakeLocking) Acquire(_ context.Context, keys []string, _ lockingtypes.AcquireArgs) error {
	f.log = append(f.log, "acquire "+keys[0])
	return nil
}
func (f *fakeLocking) Release(_ context.Context, keys []string, _ lockingtypes.ReleaseArgs) (bool, error) {
	f.log = append(f.log, "release "+keys[0])
	return true, nil
}
func (f *fakeLocking) ReleaseAll(context.Context, lockingtypes.ReleaseArgs) error { return nil }

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestLocks(t *testing.T) {
	l := &fakeLocking{}
	c := container{utilsmodulessdk.LOCKING: l}
	out, err := ChargeCustomerWorkflow.Run(context.Background(), c, WorkflowInput{OrderID: "order_1"})
	if err != nil || out != "charged" || len(l.log) != 2 || l.log[0] != "acquire order_1" || l.log[1] != "release order_1" {
		t.Fatalf("%v %v %v", out, err, l.log)
	}
}
