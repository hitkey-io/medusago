package compensation

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestCompensationRuns(t *testing.T) {
	_, err := HelloWorldWorkflow.Run(context.Background(), container{}, nil)
	var me *common.MedusaError
	if !errors.As(err, &me) || me.Type != common.MedusaErrorTypes.UNEXPECTED_STATE {
		t.Fatalf("got %v", err)
	}
}

type erp struct {
	items   map[string]map[string]any
	failOn  string
	created []string
}

func (e *erp) Retrieve(_ context.Context, id string) (map[string]any, error) { return e.items[id], nil }
func (e *erp) Create(_ context.Context, d map[string]any) error {
	e.created = append(e.created, d["id"].(string))
	return nil
}
func (e *erp) Delete(_ context.Context, id string) error {
	if id == e.failOn {
		return errors.New("boom")
	}
	return nil
}

func TestPartialDeleteRestores(t *testing.T) {
	e := &erp{items: map[string]map[string]any{"a": {"id": "a"}, "b": {"id": "b"}}, failOn: "b"}
	wf := deleteItemsStep
	_ = wf
	c := container{"erp": e, common.ContainerRegistrationKeys.LOGGER: slog.Default()}
	_, err := deleteWorkflow.Run(context.Background(), c, DeleteItemsInput{IDs: []string{"a", "b"}})
	if err == nil || len(e.created) != 1 || e.created[0] != "a" {
		t.Fatalf("%v %v", err, e.created)
	}
	// after a later failure the compensation function restores all of them
	e2 := &erp{items: map[string]map[string]any{"a": {"id": "a"}, "b": {"id": "b"}}}
	c2 := container{"erp": e2}
	if _, err := deleteThenFailWorkflow.Run(context.Background(), c2, DeleteItemsInput{IDs: []string{"a", "b"}}); err == nil || len(e2.created) != 2 {
		t.Fatalf("%v %v", err, e2.created)
	}
}
