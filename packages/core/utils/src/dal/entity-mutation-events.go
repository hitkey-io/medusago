package dal

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// This file mirrors how upstream module services observe entity mutations:
// a MikroORM event subscriber (createMedusaMikroOrmEventSubscriber) forwards
// afterCreate / afterUpdate / afterDelete to interceptEntityMutationEvents,
// which saves `<module>.<entity>.<action>` messages into the message
// aggregator of the @EmitEvents method.
//
// MikroORM dispatches these events when the unit of work is flushed: at the
// end of the outermost transaction, or before a query on an entity with
// pending changes (FlushMode.AUTO). The Go repository writes immediately, so
// the mutations stay pending in a Recorder bound to the transaction and are
// ordered like a flush (creates, updates, deletes, each in commit order)
// when the transaction commits or an auto flush happens.

// ReadyEvent is an aggregated event (RawMessageFormat): an entity mutation
// once flushed, or an event a module builds explicitly.
type ReadyEvent struct {
	EventName string
	Source    string
	Object    string
	Action    string
	Data      any
}

// Mutation actions (CommonEvents).
const (
	ActionCreated  = "created"
	ActionUpdated  = "updated"
	ActionDeleted  = "deleted"
	ActionRestored = "restored"
)

const (
	changeCreate = iota
	changeUpdate
	changeDelete
)

type mutation struct {
	change int
	action string
	entity *dml.Entity
	source string
	id     any
}

// Recorder collects the events of one transaction.
type Recorder struct {
	mu      sync.Mutex
	pending []mutation
	ready   []ReadyEvent
}

// EventSink receives the events of a committed transaction.
type EventSink interface {
	EmitModuleEvents(ctx context.Context, events []ReadyEvent)
}

type eventsModeKey struct{}

// EventsMode says what a module call does with its events.
type EventsMode int

const (
	// EmitEventsMode mirrors a call through an @EmitEvents method: the
	// events are aggregated and emitted when the call ends.
	EmitEventsMode EventsMode = iota
	// DropEventsMode drops the events (a call without any message
	// aggregator: e.g. an internal service called with a new context).
	DropEventsMode
	// SaveEventsMode mirrors a method without @EmitEvents: the events only
	// reach the message aggregator the shared context may already hold (a
	// workflow's medusaContext), emitted by a later @EmitEvents call.
	SaveEventsMode
	// DeferEventsMode mirrors @EmitEvents inside @InjectTransactionManager:
	// the aggregator is created (in the shared context) and emitted before
	// the transaction flushes, so the mutations wait in it for the next
	// @EmitEvents call.
	DeferEventsMode
)

func withMode(ctx context.Context, m EventsMode) context.Context {
	return context.WithValue(ctx, eventsModeKey{}, m)
}

// WithoutEvents marks a call whose module mutations emit no events.
func WithoutEvents(ctx context.Context) context.Context { return withMode(ctx, DropEventsMode) }

// WithoutEmit marks a call of a module method without @EmitEvents.
func WithoutEmit(ctx context.Context) context.Context { return withMode(ctx, SaveEventsMode) }

// WithDeferredEmit marks a call of a module method whose @EmitEvents runs
// inside its transaction.
func WithDeferredEmit(ctx context.Context) context.Context { return withMode(ctx, DeferEventsMode) }

// WithEmit marks a call of an @EmitEvents method (the default).
func WithEmit(ctx context.Context) context.Context { return withMode(ctx, EmitEventsMode) }

// EventsModeOf returns the events mode of the context.
func EventsModeOf(ctx context.Context) EventsMode {
	v, _ := ctx.Value(eventsModeKey{}).(EventsMode)
	return v
}

// EventsDisabled reports whether WithoutEvents marked the context.
func EventsDisabled(ctx context.Context) bool {
	return EventsModeOf(ctx) == DropEventsMode
}

// EventTx is a transaction recording the entity mutation events.
type EventTx struct {
	pgx.Tx
	rec       *Recorder
	nested    bool
	committed bool
	pending   int
	ready     int
}

// Begin starts a savepoint recording into the same recorder.
func (t *EventTx) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := t.Tx.Begin(ctx)
	if err != nil {
		return nil, err
	}
	t.rec.mu.Lock()
	p, r := len(t.rec.pending), len(t.rec.ready)
	t.rec.mu.Unlock()
	return &EventTx{Tx: tx, rec: t.rec, nested: true, pending: p, ready: r}, nil
}

// Commit commits the transaction (or releases the savepoint).
func (t *EventTx) Commit(ctx context.Context) error {
	err := t.Tx.Commit(ctx)
	if err == nil {
		t.committed = true
	}
	return err
}

// Rollback rolls back; the events of a rolled back savepoint are dropped.
func (t *EventTx) Rollback(ctx context.Context) error {
	if t.nested && !t.committed {
		t.rec.mu.Lock()
		if len(t.rec.pending) > t.pending {
			t.rec.pending = t.rec.pending[:t.pending]
		}
		if len(t.rec.ready) > t.ready {
			t.rec.ready = t.rec.ready[:t.ready]
		}
		t.rec.mu.Unlock()
	}
	return t.Tx.Rollback(ctx)
}

func recorderOf(db Querier) *Recorder {
	if t, ok := db.(*EventTx); ok {
		return t.rec
	}
	return nil
}

type txStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Transaction mirrors @InjectTransactionManager: fn runs in a transaction
// (a savepoint when db already is one). The outermost transaction records
// the entity mutations and, once committed, hands the events to the
// schema's sink (unless the context disables events).
func Transaction(ctx context.Context, db Querier, schema *Schema, fn func(tx Querier) error) error {
	if t, ok := db.(*EventTx); ok {
		if EventsDisabled(ctx) {
			// a nested call without events (no message aggregator in its
			// context): its mutations are recorded apart and dropped
			return pgx.BeginFunc(ctx, t.Tx, func(tx pgx.Tx) error {
				return fn(&EventTx{Tx: tx, rec: &Recorder{}, nested: true})
			})
		}
		return pgx.BeginFunc(ctx, t, func(tx pgx.Tx) error { return fn(tx) })
	}
	starter, ok := db.(txStarter)
	if !ok {
		return fn(db)
	}
	rec := &Recorder{}
	err := pgx.BeginFunc(ctx, starter, func(tx pgx.Tx) error {
		return fn(&EventTx{Tx: tx, rec: rec})
	})
	if err != nil {
		return err
	}
	events := rec.Flush()
	if schema == nil || schema.Sink == nil || EventsDisabled(ctx) {
		return nil
	}
	// an @EmitEvents call ends: it also emits what earlier calls left in
	// a shared aggregator
	schema.Sink.EmitModuleEvents(ctx, events)
	return nil
}

// RecordEvent saves an explicitly built event (moduleEventBuilderFactory)
// into the aggregator of the transaction db belongs to. It reports false
// when db is not a recording transaction.
func RecordEvent(db Querier, ev ReadyEvent) bool {
	rec := recorderOf(db)
	if rec == nil {
		return false
	}
	rec.mu.Lock()
	rec.ready = append(rec.ready, ev)
	rec.mu.Unlock()
	return true
}

// FlushEvents flushes the pending mutations of the transaction db belongs
// to (em.flush()).
func FlushEvents(db Querier) {
	if rec := recorderOf(db); rec != nil {
		rec.mu.Lock()
		rec.flushLocked()
		rec.mu.Unlock()
	}
}

// Flush flushes the pending mutations and returns every event.
func (r *Recorder) Flush() []ReadyEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flushLocked()
	out := r.ready
	r.ready = nil
	return out
}

func (r *Repository) recordMutation(change int, action string, e *dml.Entity, data Row) {
	rec := recorderOf(r.DB)
	if rec == nil || r.Schema == nil || r.Schema.Source == "" {
		return
	}
	pk := "id"
	if len(e.Meta.PrimaryKey) > 0 {
		pk = fieldColumn(e, e.Meta.PrimaryKey[0])
	}
	id, ok := data[pk]
	if !ok || id == nil {
		id = data["id"]
	}
	rec.mu.Lock()
	rec.pending = append(rec.pending, mutation{change: change, action: action, entity: e, source: r.Schema.Source, id: id})
	rec.mu.Unlock()
}

// recordUpdate records an update, a soft delete or a restore (the
// isSoftDeleted / isRestored checks of interceptEntityMutationEvents).
func (r *Repository) recordUpdate(e *dml.Entity, data Row, deletedAtChanged bool, before, after any) {
	action := ActionUpdated
	if deletedAtChanged {
		if after != nil && before == nil {
			action = ActionDeleted
		}
		if before != nil && after == nil {
			action = ActionRestored
		}
	}
	r.recordMutation(changeUpdate, action, e, data)
}

// autoFlush mirrors FlushMode.AUTO: a query on an entity with pending
// changes flushes the unit of work first.
func (r *Repository) autoFlush(e *dml.Entity) {
	rec := recorderOf(r.DB)
	if rec == nil {
		return
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, m := range rec.pending {
		if m.entity.Meta.ClassName == e.Meta.ClassName {
			rec.flushLocked()
			return
		}
	}
}

// flushLocked orders the pending mutations like UnitOfWork.persistToDatabase
// (creates and updates in commit order, deletes in reverse commit order)
// and turns them into events.
func (r *Recorder) flushLocked() {
	if len(r.pending) == 0 {
		return
	}
	order := commitOrder(r.pending)
	emit := func(change int, reverse bool) {
		for i := range order {
			c := order[i]
			if reverse {
				c = order[len(order)-1-i]
			}
			for _, m := range r.pending {
				if m.change == change && m.entity.Meta.ClassName == c {
					r.ready = append(r.ready, mutationEvent(m))
				}
			}
		}
	}
	emit(changeCreate, false)
	emit(changeUpdate, false)
	emit(changeDelete, true)
	r.pending = nil
}

func mutationEvent(m mutation) ReadyEvent {
	object := strings.ToLower(camelToSnake(m.entity.Meta.ClassName))
	data := NewRecord()
	data.Set("id", m.id)
	return ReadyEvent{
		EventName: BuildModuleResourceEventName(m.source, object, m.action),
		Source:    m.source,
		Object:    object,
		Action:    m.action,
		Data:      data,
	}
}

var (
	camelRe      = regexp.MustCompile(`([a-z])([A-Z])`)
	kebabSpaceRe = regexp.MustCompile(`[\s_]+`)
)

func camelToSnake(s string) string {
	return strings.ToLower(camelRe.ReplaceAllString(s, "${1}_${2}"))
}

// BuildModuleResourceEventName mirrors buildModuleResourceEventName:
// `${prefix}.${lowerCaseFirst(kebabCase(objectName))}.${action}`.
func BuildModuleResourceEventName(prefix, objectName, action string) string {
	kebab := camelRe.ReplaceAllString(objectName, "$1-$2")
	kebab = strings.ToLower(kebabSpaceRe.ReplaceAllString(kebab, "-"))
	if kebab != "" {
		r := []rune(kebab)
		r[0] = unicode.ToLower(r[0])
		kebab = string(r)
	}
	if prefix != "" {
		return fmt.Sprintf("%s.%s.%s", prefix, kebab, action)
	}
	return fmt.Sprintf("%s.%s", kebab, action)
}

// commitOrder mirrors UnitOfWork.getCommitOrder: the classes of the change
// sets (in change set order) sorted by CommitOrderCalculator, the reverse
// post order of a depth first search where a class referenced by a to-one
// owner (or an owning m:n without pivot entity) comes before the owner.
func commitOrder(pending []mutation) []string {
	type edge struct {
		to     string
		weight int
	}
	type node struct {
		state int
		deps  []edge
	}
	var names []string
	nodes := map[string]*node{}
	entities := map[string]*dml.Entity{}
	for _, m := range pending {
		c := m.entity.Meta.ClassName
		if _, ok := nodes[c]; ok {
			continue
		}
		nodes[c] = &node{}
		names = append(names, c)
		entities[c] = m.entity
	}
	for _, name := range names {
		for _, p := range entities[name].Meta.Properties {
			toOneOwner := (p.Kind == "1:1" && p.Owner) || p.Kind == "m:1"
			toManyOwner := p.Kind == "m:n" && p.Owner && p.PivotEntity == ""
			if !toOneOwner && !toManyOwner {
				continue
			}
			target, ok := nodes[p.Entity]
			if !ok {
				continue
			}
			weight := 1
			if p.Nullable || (p.Persist != nil && !*p.Persist) {
				weight = 0
			}
			// addDependency(propertyType, entityName): the last one wins
			replaced := false
			for i, d := range target.deps {
				if d.to == name {
					target.deps[i].weight = weight
					replaced = true
				}
			}
			if !replaced {
				target.deps = append(target.deps, edge{to: name, weight: weight})
			}
		}
	}
	var sorted []string
	var visit func(name string)
	visitOpen := func(name, target string, weight int) {
		t := nodes[target]
		back := -1
		for _, d := range t.deps {
			if d.to == name {
				back = d.weight
			}
		}
		if back < 0 || back >= weight {
			return
		}
		for _, d := range t.deps {
			if nodes[d.to].state == 0 {
				visit(d.to)
			}
		}
		t.state = 2
		sorted = append(sorted, target)
	}
	visit = func(name string) {
		n := nodes[name]
		n.state = 1
		for _, d := range n.deps {
			switch nodes[d.to].state {
			case 2:
			case 1:
				visitOpen(name, d.to, d.weight)
			case 0:
				visit(d.to)
			}
		}
		if n.state != 2 {
			n.state = 2
			sorted = append(sorted, name)
		}
	}
	for _, name := range names {
		if nodes[name].state == 0 {
			visit(name)
		}
	}
	out := make([]string, len(sorted))
	for i, s := range sorted {
		out[len(sorted)-1-i] = s
	}
	return out
}

// recordDispatched mirrors eventManager.dispatchEvent from the internal
// service (delete, upsertWithReplace): the event is aggregated at once,
// not at flush time.
func (r *Repository) recordDispatched(action string, e *dml.Entity, id any) {
	rec := recorderOf(r.DB)
	if rec == nil || r.Schema == nil || r.Schema.Source == "" {
		return
	}
	rec.mu.Lock()
	rec.ready = append(rec.ready, mutationEvent(mutation{action: action, entity: e, source: r.Schema.Source, id: id}))
	rec.mu.Unlock()
}

// performedActions mirrors the PerformedActions of upsertWithReplace: the
// ids created, updated and deleted per entity, in order.
type performedActions struct {
	created, updated, deleted classIDs
}

type classIDs struct {
	order []*dml.Entity
	ids   map[*dml.Entity][]any
}

func (c *classIDs) add(e *dml.Entity, ids ...any) {
	if len(ids) == 0 {
		return
	}
	if c.ids == nil {
		c.ids = map[*dml.Entity][]any{}
	}
	if _, ok := c.ids[e]; !ok {
		c.order = append(c.order, e)
	}
	c.ids[e] = append(c.ids[e], ids...)
}

// dispatchPerformedActions mirrors the internal service's upsertWithReplace:
// afterCreate, then afterUpdate, then afterDelete for every performed action.
func (r *Repository) dispatchPerformedActions(pa *performedActions) {
	for _, set := range []struct {
		c      *classIDs
		action string
	}{{&pa.created, ActionCreated}, {&pa.updated, ActionUpdated}, {&pa.deleted, ActionDeleted}} {
		for _, e := range set.c.order {
			for _, id := range set.c.ids[e] {
				r.recordDispatched(set.action, e, id)
			}
		}
	}
}
