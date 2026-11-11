package utils

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/utils/src/graphql"
)

// Clearer is the part of the caching module the strategy invalidates with.
type Clearer interface {
	Clear(ctx context.Context, opts caching.ClearOptions) error
}

// DefaultCacheStrategy mirrors DefaultCacheStrategy: keys are the hash of
// the stable stringified input; tags are the entities found in the data;
// the entity events of the event bus invalidate the entries tagged with
// their entities.
type DefaultCacheStrategy struct {
	module Clearer
	hasher func(string) string
	logger *slog.Logger

	mu     sync.RWMutex
	parser *CacheInvalidationParser
}

// NewDefaultCacheStrategy mirrors the constructor.
func NewDefaultCacheStrategy(module Clearer, hasher func(string) string, logger *slog.Logger) *DefaultCacheStrategy {
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultCacheStrategy{module: module, hasher: hasher, logger: logger}
}

// ObjectHash mirrors objectHash.
func (s *DefaultCacheStrategy) ObjectHash(input any) (string, error) {
	str, err := caching.StableStringify(input)
	if err != nil {
		return "", err
	}
	return s.hasher(str), nil
}

// OnApplicationStart mirrors onApplicationStart: the parser over the merged
// schema and the joiner configs, and the invalidation on every event of the
// event bus (a "*" subscriber and an interceptor).
func (s *DefaultCacheStrategy) OnApplicationStart(schema *graphql.Schema, joinerConfigs []*joiner.ModuleJoinerConfig, bus any) {
	s.mu.Lock()
	s.parser = NewCacheInvalidationParser(schema, joinerConfigs)
	s.mu.Unlock()
	if sub, ok := bus.(eventbus.Subscribable); ok {
		sub.Subscribe("*", func(ctx context.Context, ev eventbus.Event) error {
			s.HandleEvent(ctx, ev.Name, ev.Data)
			return nil
		})
	}
	if ic, ok := bus.(eventbus.Interceptable); ok {
		ic.AddInterceptor(func(m eventbus.Message, _ bool, _ string) {
			s.HandleEvent(context.Background(), m.Name, m.Data)
		})
	}
}

// HandleEvent mirrors the event handler: the entity of each id of the
// event data ("<module>.<entity>.<operation>") is invalidated with the
// operation.
func (s *DefaultCacheStrategy) HandleEvent(ctx context.Context, name string, data any) {
	parts := strings.Split(name, ".")
	operation := parts[len(parts)-1]
	entityType := parts[len(parts)-1]
	if len(parts) > 1 {
		entityType = parts[len(parts)-2]
	}
	var items []any
	if list, ok := data.([]any); ok {
		items = list
	} else {
		items = []any{data}
	}
	tags := []string{}
	for _, item := range items {
		var id any
		switch t := item.(type) {
		case *dal.Record:
			if t != nil {
				id, _ = t.Get("id")
			}
		case map[string]any:
			id = t["id"]
		}
		ids, ok := id.([]any)
		if !ok {
			if strs, isStrs := id.([]string); isStrs {
				for _, x := range strs {
					ids = append(ids, x)
				}
			} else {
				ids = []any{id}
			}
		}
		for _, one := range ids {
			ref := caching.EntityReference{Type: common.UpperCaseFirst(common.ToCamelCase(entityType)), ID: one}
			t, err := s.ComputeTags(item, &caching.ComputeTagsOptions{Entities: []caching.EntityReference{ref}, Operation: operation})
			if err != nil {
				continue
			}
			tags = append(tags, t...)
		}
	}
	t := true
	if err := s.module.Clear(context.WithoutCancel(ctx), caching.ClearOptions{Tags: tags, Options: true, AutoInvalidate: &t}); err != nil {
		s.logger.Warn("cache invalidation failed: " + err.Error())
	}
}

// ComputeKey mirrors computeKey.
func (s *DefaultCacheStrategy) ComputeKey(input any) (string, error) {
	return s.ObjectHash(input)
}

// ComputeTags mirrors computeTags.
func (s *DefaultCacheStrategy) ComputeTags(input any, opts *caching.ComputeTagsOptions) ([]string, error) {
	s.mu.RLock()
	parser := s.parser
	s.mu.RUnlock()
	if parser == nil {
		parser = NewCacheInvalidationParser(nil, nil)
	}
	var entities []caching.EntityReference
	operation := ""
	if opts != nil {
		entities = opts.Entities
		operation = opts.Operation
	}
	if entities == nil {
		entities = parser.ParseObjectForEntities(input, "", false)
	}
	if len(entities) == 0 {
		return []string{}, nil
	}
	events := parser.BuildInvalidationEvents(entities, operation)
	seen := map[string]bool{}
	tags := []string{}
	for _, e := range events {
		for _, k := range e.CacheKeys {
			if !seen[k] {
				seen[k] = true
				tags = append(tags, k)
			}
		}
	}
	return tags, nil
}
