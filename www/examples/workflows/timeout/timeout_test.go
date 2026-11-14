package timeout

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestTimeout(t *testing.T) {
	done := make(chan composer.WorkflowEvent, 1)
	composer.Subscribe("wait-workflow", "tx-timeout", "s", func(ev composer.WorkflowEvent) {
		if ev.EventType == "onFinish" {
			done <- ev
		}
	})
	start := time.Now()
	_, _, err := MyWorkflow.RunWithInfo(context.Background(), container{}, nil, composer.RunOptions{TransactionID: "tx-timeout", Notify: true})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-done:
		b, _ := json.Marshal(ev.Errors)
		if ev.State != "reverted" || !strings.Contains(string(b), "Timeout error") || time.Since(start) < 2*time.Second {
			t.Fatalf("%v %s", ev.State, b)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("timeout")
	}
}
