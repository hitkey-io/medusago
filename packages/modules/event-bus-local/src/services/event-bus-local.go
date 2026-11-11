// Package services mirrors @medusajs/event-bus-local/src/services.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// Event mirrors the published event body: { name, data, metadata }.
type Event = eventbus.Event

// Subscriber handles a published event.
type Subscriber = eventbus.Subscriber

// LocalEventBusService mirrors LocalEventBusService: in-process
// subscribers, grouped (staged) events per event group id.
type LocalEventBusService struct {
	mu           sync.Mutex
	subscribers  map[string][]Subscriber
	interceptors []*Interceptor
	grouped      map[string][]composer.Message
	logger       *slog.Logger
	// wg tracks the asynchronous deliveries (tests wait on it).
	wg sync.WaitGroup
}

// NewLocalEventBusService creates the service.
func NewLocalEventBusService(logger *slog.Logger) *LocalEventBusService {
	if logger == nil {
		logger = slog.Default()
	}
	return &LocalEventBusService{subscribers: map[string][]Subscriber{}, grouped: map[string][]composer.Message{}, logger: logger}
}

func withMeta(m *dal.Record, key string) *dal.Record {
	out := dal.NewRecord()
	if m != nil {
		for _, k := range m.Keys() {
			v, _ := m.Get(k)
			out.Set(k, v)
		}
	}
	out.Set(key, time.Now())
	return out
}

// Interceptor mirrors InterceptorSubscriber: it receives every message
// before it is published.
type Interceptor func(m composer.Message, isGrouped bool, eventGroupID string)

// AddInterceptor mirrors addInterceptor.
func (s *LocalEventBusService) AddInterceptor(fn Interceptor) *Interceptor {
	p := &fn
	s.mu.Lock()
	s.interceptors = append(s.interceptors, p)
	s.mu.Unlock()
	return p
}

// RemoveInterceptor mirrors removeInterceptor.
func (s *LocalEventBusService) RemoveInterceptor(p *Interceptor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, x := range s.interceptors {
		if x == p {
			s.interceptors = append(s.interceptors[:i], s.interceptors[i+1:]...)
			return
		}
	}
}

// callInterceptors mirrors callInterceptors (errors are logged).
func (s *LocalEventBusService) callInterceptors(m composer.Message, isGrouped bool, eventGroupID string) {
	s.mu.Lock()
	interceptors := append([]*Interceptor{}, s.interceptors...)
	s.mu.Unlock()
	for _, p := range interceptors {
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error(fmt.Sprintf("Error in event bus interceptor: %v", r))
				}
			}()
			(*p)(m, isGrouped, eventGroupID)
		}()
	}
}

// Subscribe mirrors subscribe(event, subscriber).
func (s *LocalEventBusService) Subscribe(event string, sub Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[event] = append(s.subscribers[event], sub)
}

// Emit mirrors emit: grouped events are staged, others published
// asynchronously.
func (s *LocalEventBusService) Emit(ctx context.Context, messages []composer.Message) error {
	for _, m := range messages {
		m.Metadata = withMeta(m.Metadata, "created_at")
		gid := ""
		if m.Metadata != nil {
			if v, ok := m.Metadata.Get("eventGroupId"); ok {
				gid, _ = v.(string)
			}
		}
		if gid != "" {
			s.mu.Lock()
			s.grouped[gid] = append(s.grouped[gid], m)
			s.mu.Unlock()
			continue
		}
		// options.delay (ms) postpones the publication
		if d := delayOf(m.Options); d > 0 {
			m := m
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				time.Sleep(d)
				s.publish(m, false, "")
			}()
			continue
		}
		s.publish(m, false, "")
	}
	return nil
}

func delayOf(options *dal.Record) time.Duration {
	if options == nil {
		return 0
	}
	v, _ := options.Get("delay")
	switch n := v.(type) {
	case float64:
		return time.Duration(n * float64(time.Millisecond))
	case int:
		return time.Duration(n) * time.Millisecond
	}
	return 0
}

func (s *LocalEventBusService) publish(m composer.Message, isGrouped bool, eventGroupID string) {
	s.callInterceptors(m, isGrouped, eventGroupID)
	s.mu.Lock()
	subs := append(append([]Subscriber{}, s.subscribers[m.Name]...), s.subscribers["*"]...)
	s.mu.Unlock()
	if len(subs) == 0 {
		return
	}
	internal := false
	if m.Options != nil {
		v, _ := m.Options.Get("internal")
		internal, _ = v.(bool)
	}
	if !internal {
		s.logger.Info(fmt.Sprintf("Processing %s which has %d subscribers", m.Name, len(subs)))
	}
	ev := Event{Name: m.Name, Data: m.Data, Metadata: withMeta(m.Metadata, "published_at")}
	for _, sub := range subs {
		sub := sub
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := sub(context.Background(), ev); err != nil {
				s.logger.Error(fmt.Sprintf("An error occurred while processing %s: %v", m.Name, err))
			}
		}()
	}
}

// ReleaseGroupedEvents mirrors releaseGroupedEvents.
func (s *LocalEventBusService) ReleaseGroupedEvents(_ context.Context, id string) error {
	s.mu.Lock()
	events := s.grouped[id]
	delete(s.grouped, id)
	s.mu.Unlock()
	for _, m := range events {
		s.publish(m, true, id)
	}
	return nil
}

// ClearGroupedEvents mirrors clearGroupedEvents.
func (s *LocalEventBusService) ClearGroupedEvents(_ context.Context, id string, eventNames []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(eventNames) == 0 {
		delete(s.grouped, id)
		return nil
	}
	drop := map[string]bool{}
	for _, n := range eventNames {
		drop[n] = true
	}
	var keep []composer.Message
	for _, m := range s.grouped[id] {
		if !drop[m.Name] {
			keep = append(keep, m)
		}
	}
	s.grouped[id] = keep
	return nil
}

// Wait waits for the in-flight deliveries.
func (s *LocalEventBusService) Wait() { s.wg.Wait() }
