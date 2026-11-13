package composer

import (
	"fmt"
	"sync"

	"github.com/oklog/ulid/v2"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Step graphs of the upstream workflows.
//
// Upstream composes a workflow once, when it is defined: the steps form a
// graph (a step's "next" is a list when branches run in parallel, steps
// that do not depend on each other run side by side) and every run of the
// workflow follows it. The TransactionFlow of a run lists every step of the
// graph from the start (not_started until it runs), the steps of a when()
// whose condition is false end up skipped and a runAsStep workflow is one
// step of its parent. A Go workflow body is plain code, so the journal of a
// run takes the graph upstream composes (extracted into upstream-flows.go)
// and places the steps there as they run; a step the graph does not know
// (a workflow without an upstream counterpart) is appended after the last
// step that ran.

// flowNode is one step of a composed graph.
type flowNode struct {
	action string
	// uuid is generated once per process, like the uuid a step gets when
	// upstream composes the workflow
	uuid string
	// fields are the definition fields of the step except next and uuid
	// (action, noCompensation, async, ...), in upstream order
	fields *dal.Record
	next   []*flowNode
}

// flowDef is the composed graph of a workflow and the options it is
// created with (createWorkflow({ name, store, ... }) or {} for a name).
type flowDef struct {
	nodes   []*flowNode
	options *dal.Record
}

var (
	flowsOnce sync.Once
	flowDefs  map[string]*flowDef
	flowsMu   sync.RWMutex
)

func loadFlowDefinitions() {
	flowDefs = map[string]*flowDef{}
	raw, err := dal.DecodeOrderedJSON([]byte(upstreamFlowsJSON))
	if err != nil {
		panic(fmt.Sprintf("composer: upstream flows: %v", err))
	}
	all, _ := raw.(*dal.Record)
	for _, id := range all.Keys() {
		v, _ := all.Get(id)
		entry, _ := v.(*dal.Record)
		flow, _ := entry.Get("flow")
		opts, _ := entry.Get("options")
		options, _ := opts.(*dal.Record)
		if options == nil {
			options = dal.NewRecord()
		}
		flowDefs[id] = &flowDef{nodes: parseFlowNodes(flow), options: options}
	}
}

func parseFlowNodes(v any) []*flowNode {
	switch t := v.(type) {
	case []any:
		var out []*flowNode
		for _, x := range t {
			out = append(out, parseFlowNodes(x)...)
		}
		return out
	case *dal.Record:
		if _, ok := t.Get("action"); !ok {
			// the root of a workflow starting with parallel steps holds
			// only their list: the steps hang off _root directly
			x, _ := t.Get("next")
			return parseFlowNodes(x)
		}
		n := &flowNode{uuid: ulid.Make().String(), fields: dal.NewRecord()}
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			switch k {
			case "next":
				n.next = parseFlowNodes(x)
			default:
				switch k {
				case "action":
					n.action, _ = x.(string)
				case "uuid":
					x = n.uuid
				}
				n.fields.Set(k, x)
			}
		}
		return []*flowNode{n}
	}
	return nil
}

// flowDefinition returns the first steps of the graph upstream composes for
// the workflow (nil when it has none).
func flowDefinition(workflowID string) []*flowNode {
	if d := lookupFlowDef(workflowID); d != nil {
		return d.nodes
	}
	return nil
}

func lookupFlowDef(workflowID string) *flowDef {
	flowsOnce.Do(loadFlowDefinitions)
	flowsMu.RLock()
	defer flowsMu.RUnlock()
	return flowDefs[workflowID]
}

// RegisterFlowDefinition sets the step graph of a workflow (a project
// workflow mirroring one defined in JavaScript), as flow definition JSON
// (the WorkflowManager flow: { action, noCompensation, next, ... }) and
// the options it is created with (nil for none).
func RegisterFlowDefinition(workflowID string, flowJSON []byte, optionsJSON []byte) error {
	flowsOnce.Do(loadFlowDefinitions)
	raw, err := dal.DecodeOrderedJSON(flowJSON)
	if err != nil {
		return err
	}
	options := dal.NewRecord()
	if len(optionsJSON) > 0 {
		o, err := dal.DecodeOrderedJSON(optionsJSON)
		if err != nil {
			return err
		}
		if r, ok := o.(*dal.Record); ok {
			options = r
		}
	}
	flowsMu.Lock()
	flowDefs[workflowID] = &flowDef{nodes: parseFlowNodes(raw), options: options}
	flowsMu.Unlock()
	return nil
}

// findFlowNode returns the step of the graph with the action.
func findFlowNode(nodes []*flowNode, action string) *flowNode {
	for _, n := range nodes {
		if n.action == action {
			return n
		}
		if found := findFlowNode(n.next, action); found != nil {
			return found
		}
	}
	return nil
}

func fieldBool(r *dal.Record, k string) bool {
	if r == nil {
		return false
	}
	v, _ := r.Get(k)
	b, _ := v.(bool)
	return b
}

func fieldInt(r *dal.Record, k string) int {
	if r == nil {
		return 0
	}
	v, _ := r.Get(k)
	f, _ := toFloat(v)
	return int(f)
}
