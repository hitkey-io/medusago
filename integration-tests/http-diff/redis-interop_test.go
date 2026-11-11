package httpdiff

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// TestRedisWorkflowsInteropWithNode checks the workflow engine interop of a
// Node.js and a Go instance sharing a database and a Redis
// (MEDUSA_HTTP_INTEROP_NODE_URL: an upstream server with the Redis modules
// and the http-diff-wait workflows; MEDUSA_HTTP_INTEROP_DATABASE_URL: its
// database; MEDUSA_TEST_REDIS_URL: its Redis):
//
//   - a transaction started by Node.js and waiting for its async step is
//     completed through the Go instance: Go resumes the Node.js checkpoint
//     (replay matched by step action) and runs the remaining steps;
//   - a step_timeout job of a Node.js transaction is processed by Go when
//     Node.js is stopped (MEDUSA_HTTP_INTEROP_NODE_STOP_CMD);
//   - a transaction started by Go and completed through Node.js is
//     reported (Node.js continues the stored flow, which only lists the
//     steps Go ran: the steps after the waiting one do not run).
func TestRedisWorkflowsInteropWithNode(t *testing.T) {
	node := os.Getenv("MEDUSA_HTTP_INTEROP_NODE_URL")
	dbURL := os.Getenv("MEDUSA_HTTP_INTEROP_DATABASE_URL")
	if node == "" || dbURL == "" || os.Getenv("MEDUSA_TEST_REDIS_URL") == "" {
		t.Skip("MEDUSA_HTTP_INTEROP_NODE_URL, MEDUSA_HTTP_INTEROP_DATABASE_URL and MEDUSA_TEST_REDIS_URL are required")
	}
	t.Setenv("HTTP_DIFF_REDIS", "true")
	t.Setenv("HTTP_DIFF_WAIT_WORKFLOW", "true")
	cfg := config.DefineConfig(config.InputConfig{ProjectConfig: config.ProjectConfig{DatabaseURL: dbURL}, Modules: scenarioModules()})
	app, err := loaders.InitializeWithOptions(context.Background(), cfg, loaders.Options{Routes: scenarioRoutes()})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Handler)
	defer func() {
		srv.Close()
		app.Shutdown()
	}()
	goURL := srv.URL

	call := func(base, method, path, token string, body any) (int, map[string]any) {
		t.Helper()
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, base+path, r)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	_, login := call(node, "POST", "/auth/user/emailpass", "", map[string]any{"email": "admin@medusa-test.com", "password": "supersecret"})
	token, _ := login["token"].(string)
	if token == "" {
		t.Fatalf("login: %v", login)
	}
	state := func(base, tx string) string {
		_, out := call(base, "GET", "/admin/workflows-executions?transaction_id="+tx+"&fields=state", token, nil)
		list, _ := out["workflow_executions"].([]any)
		if len(list) == 0 {
			return ""
		}
		e, _ := list[0].(map[string]any)
		return fmt.Sprint(e["state"])
	}
	waitState := func(base, tx string, want ...string) string {
		deadline := time.Now().Add(20 * time.Second)
		for {
			s := state(base, tx)
			for _, w := range want {
				if s == w {
					return s
				}
			}
			if time.Now().After(deadline) {
				return s
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	tags := func(q string) []string {
		_, out := call(goURL, "GET", "/admin/product-tags?q="+q+"&order=value&fields=value,metadata", token, nil)
		var values []string
		list, _ := out["product_tags"].([]any)
		for _, x := range list {
			m, _ := x.(map[string]any)
			md, _ := json.Marshal(m["metadata"])
			values = append(values, fmt.Sprintf("%v %s", m["value"], md))
		}
		return values
	}
	suffix := strings.ToLower(ulid.Make().String()[20:])

	// Node.js starts, Go completes
	tx := "tx-n2g-" + suffix
	if st, out := call(node, "POST", "/admin/workflows-executions/http-diff-wait/run", token, map[string]any{"transaction_id": tx, "input": map[string]any{"value": "n2g-" + suffix}}); st != 200 {
		t.Fatalf("run on node: %d %v", st, out)
	}
	if s := waitState(node, tx, "invoking"); s != "invoking" {
		t.Fatalf("node run state %q", s)
	}
	if st, out := call(goURL, "POST", "/admin/http-diff-wait/"+tx, token, map[string]any{"action": "success", "response": map[string]any{"by": "go"}}); st != 200 {
		t.Fatalf("complete on go: %d %v", st, out)
	}
	if s := waitState(goURL, tx, "done"); s != "done" {
		t.Fatalf("n2g state %q", s)
	}
	got := strings.Join(tags("n2g-"+suffix), "\n")
	want := fmt.Sprintf("n2g-%[1]s-before null\nn2g-%[1]s-before-after {\"response\":{\"by\":\"go\"}}", suffix)
	if got != want {
		t.Fatalf("n2g tags\n%s\nwant\n%s", got, want)
	}

	// Go starts, Node.js completes: reported only
	tx = "tx-g2n-" + suffix
	if st, out := call(goURL, "POST", "/admin/workflows-executions/http-diff-wait/run", token, map[string]any{"transaction_id": tx, "input": map[string]any{"value": "g2n-" + suffix}}); st != 200 {
		t.Fatalf("run on go: %d %v", st, out)
	}
	st, out := call(node, "POST", "/admin/http-diff-wait/"+tx, token, map[string]any{"action": "success", "response": map[string]any{"by": "node"}})
	final := waitState(goURL, tx, "done", "reverted", "failed")
	t.Logf("Go-started transaction completed through Node.js: %d %v, state %q, tags %v", st, out, final, tags("g2n-"+suffix))

	// Node.js starts with a 2 s step timeout: the step_timeout job is
	// processed by an instance (by Go when MEDUSA_HTTP_INTEROP_NODE_STOP_CMD
	// stops Node.js) and the first step is compensated
	tx = "tx-n2g-timeout-" + suffix
	if st, out := call(node, "POST", "/admin/workflows-executions/http-diff-wait-timeout/run", token, map[string]any{"transaction_id": tx, "input": map[string]any{"value": "n2gt-" + suffix}}); st != 200 {
		t.Fatalf("run on node: %d %v", st, out)
	}
	if stop := os.Getenv("MEDUSA_HTTP_INTEROP_NODE_STOP_CMD"); stop != "" {
		if out, err := exec.Command("sh", "-c", stop).CombinedOutput(); err != nil {
			t.Fatalf("stop node: %v %s", err, out)
		}
	}
	if s := waitState(goURL, tx, "reverted"); s != "reverted" {
		t.Fatalf("n2g timeout state %q", s)
	}
	if got := tags("n2gt-" + suffix); len(got) != 0 {
		t.Fatalf("n2g timeout tags %v", got)
	}

}
