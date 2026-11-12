// Package subscribe mirrors api/admin/workflows-executions/[workflow_id]/subscribe.
package subscribe

import (
	"encoding/json"
	"math/rand"
	nethttp "net/http"
	"strconv"
	"sync"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// subscriberID mirrors "__sub__" + Math.random().toString(36).substring(2, 9).
func subscriberID() string {
	s := strconv.FormatUint(rand.Uint64(), 36)
	if len(s) > 7 {
		s = s[:7]
	}
	return "__sub__" + s
}

// Stream mirrors the subscribe routes: an event stream of the workflow
// engine notifications until the client disconnects.
func Stream(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, workflowID, transactionID string) error {
	w := res.Handoff()
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	w.WriteHeader(nethttp.StatusOK)
	flusher, _ := w.(nethttp.Flusher)
	if flusher != nil {
		flusher.Flush()
	}
	var mu sync.Mutex
	closed := false
	id := subscriberID()
	composer.Subscribe(workflowID, transactionID, id, func(ev composer.WorkflowEvent) {
		data := dal.NewRecord()
		data.Set("event_type", ev.EventType)
		data.Set("workflow_id", ev.WorkflowID)
		data.Set("transaction_id", ev.TransactionID)
		if ev.Step != nil {
			data.Set("step", ev.Step)
		}
		if ev.HasResponse {
			data.Set("response", ev.Response)
		}
		if ev.HasResult {
			data.Set("result", ev.Result)
		}
		if ev.HasErrors {
			data.Set("errors", ev.Errors)
		}
		b, err := json.Marshal(data)
		if err != nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		_, _ = w.Write([]byte("event: " + ev.EventType + "\ndata: " + string(b) + "\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	})
	// the stream ends when the client disconnects
	<-req.Request.Context().Done()
	mu.Lock()
	closed = true
	mu.Unlock()
	composer.Unsubscribe(workflowID, transactionID, id)
	return nil
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return Stream(req, res, req.Params["workflow_id"], "")
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
