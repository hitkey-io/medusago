// Package services mirrors @medusajs/caching/src/services.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/graphql"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/types"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/utils"
)

const oneHourInSecond = 60 * 60

// call is an ongoing request shared by identical concurrent requests
// (ongoingRequests).
type call struct {
	done   chan struct{}
	result any
	err    error
}

// CachingModuleService mirrors CachingModuleService.
type CachingModuleService struct {
	providers         *CacheProviderService
	defaultProviderID string
	strategy          *utils.DefaultCacheStrategy
	logger            *slog.Logger
	ttl               int

	mu              sync.Mutex
	ongoingRequests map[string]*call
}

var _ caching.Module = (*CachingModuleService)(nil)

// NewCachingModuleService mirrors the constructor (ttl one hour by
// default).
func NewCachingModuleService(providers *CacheProviderService, defaultProviderID string, hasher func(string) string, ttl *int, logger *slog.Logger) *CachingModuleService {
	if logger == nil {
		logger = slog.Default()
	}
	s := &CachingModuleService{providers: providers, defaultProviderID: defaultProviderID, logger: logger, ttl: oneHourInSecond, ongoingRequests: map[string]*call{}}
	if ttl != nil {
		s.ttl = *ttl
	}
	s.strategy = utils.NewDefaultCacheStrategy(s, hasher, logger)
	return s
}

// OnApplicationStart mirrors onApplicationStart: the strategy gets the
// merged schema of the loaded modules' joiner configs.
func (s *CachingModuleService) OnApplicationStart(joinerConfigs []*joiner.ModuleJoinerConfig, eventBus any) error {
	var schemas []string
	for _, c := range joinerConfigs {
		if c != nil {
			schemas = append(schemas, c.Schema)
		}
	}
	schema, err := graphql.BuildSchema(schemas...)
	if err != nil {
		return err
	}
	s.strategy.OnApplicationStart(schema, joinerConfigs, eventBus)
	return nil
}

// OnApplicationShutdown mirrors onApplicationShutdown.
func (s *CachingModuleService) OnApplicationShutdown(ctx context.Context) error {
	return s.providers.Shutdown(ctx)
}

func sortedJoin(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	sorted := append([]string{}, tags...)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

func (s *CachingModuleService) providersPart(providers []string) string {
	if len(providers) == 0 {
		return s.defaultProviderID
	}
	return strings.Join(providers, ",")
}

// getRequestKey mirrors getRequestKey.
func (s *CachingModuleService) getRequestKey(key string, tags, providers []string) string {
	return key + "|" + sortedJoin(tags) + "|" + s.providersPart(providers)
}

// getClearRequestKey mirrors getClearRequestKey.
func (s *CachingModuleService) getClearRequestKey(key string, tags, providers []string) string {
	return "clear:" + s.getRequestKey(key, tags, providers)
}

// dedupe runs fn once for concurrent requests with the same request key.
func (s *CachingModuleService) dedupe(requestKey string, fn func() (any, error)) (any, error) {
	s.mu.Lock()
	if c, ok := s.ongoingRequests[requestKey]; ok {
		s.mu.Unlock()
		<-c.done
		return c.result, c.err
	}
	c := &call{done: make(chan struct{})}
	s.ongoingRequests[requestKey] = c
	s.mu.Unlock()
	c.result, c.err = fn()
	s.mu.Lock()
	delete(s.ongoingRequests, requestKey)
	s.mu.Unlock()
	close(c.done)
	return c.result, c.err
}

func invalidArgument(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, msg)
}

// Get mirrors get({ key, tags, providers }).
func (s *CachingModuleService) Get(ctx context.Context, o caching.GetOptions) (any, error) {
	if o.Key == "" && o.Tags == nil {
		return nil, invalidArgument("Either key or tags must be provided")
	}
	return s.dedupe(s.getRequestKey(o.Key, o.Tags, o.Providers), func() (any, error) {
		return s.performCacheGet(ctx, o.Key, o.Tags, o.Providers), nil
	})
}

// performCacheGet mirrors performCacheGet: the first provider holding the
// data (a failing provider is skipped).
func (s *CachingModuleService) performCacheGet(ctx context.Context, key string, tags, providers []string) any {
	ids := providers
	if len(ids) == 0 {
		ids = []string{s.defaultProviderID}
	}
	for _, id := range ids {
		p, err := s.providers.RetrieveProvider(id)
		if err == nil {
			var res any
			res, err = p.Get(ctx, key, tags)
			if err == nil {
				if res != nil {
					return res
				}
				continue
			}
		}
		s.logger.Warn(fmt.Sprintf("Cache provider %s failed: %s", id, err.Error()))
	}
	return nil
}

// Set mirrors set({ key, data, ttl, tags, computeAutomaticTags, providers,
// options }).
func (s *CachingModuleService) Set(ctx context.Context, o caching.SetOptions) error {
	if o.Key == "" {
		return invalidArgument("[CachingModuleService] Key must be provided")
	}
	var computed []string
	if o.Tags == nil || o.ComputeAutomaticTags {
		c, err := s.strategy.ComputeTags(o.Data, nil)
		if err != nil {
			return err
		}
		computed = c
	}
	tags := computed
	if o.Tags != nil {
		tags = []string{}
		seen := map[string]bool{}
		for _, t := range append(append([]string{}, o.Tags...), computed...) {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	providers := o.Providers
	if len(providers) == 0 {
		providers = []caching.ProviderTTL{{ID: s.defaultProviderID}}
	}
	ids := make([]string, len(providers))
	for i, p := range providers {
		ids[i] = p.ID
	}
	// getRequestKey sorts the tags in place
	sort.Strings(tags)
	_, err := s.dedupe(s.getRequestKey(o.Key, tags, ids), func() (any, error) {
		return nil, s.performCacheSet(ctx, o, tags, providers)
	})
	return err
}

// performCacheSet mirrors performCacheSet.
func (s *CachingModuleService) performCacheSet(ctx context.Context, o caching.SetOptions, tags []string, providers []caching.ProviderTTL) error {
	var wg sync.WaitGroup
	errs := make([]error, len(providers))
	for i, po := range providers {
		p, err := s.providers.RetrieveProvider(po.ID)
		if err != nil {
			errs[i] = err
			continue
		}
		ttl := s.ttl
		switch {
		case po.TTL != nil:
			ttl = *po.TTL
		case o.TTL != nil:
			ttl = *o.TTL
		}
		wg.Add(1)
		go func(i int, p types.ICachingProviderService, ttl int) {
			defer wg.Done()
			errs[i] = p.Set(ctx, types.ProviderSetOptions{Key: o.Key, Data: o.Data, TTL: &ttl, Tags: tags, HasOptions: o.HasOptions, AutoInvalidate: o.AutoInvalidate})
		}(i, p, ttl)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Clear mirrors clear({ key, tags, options, providers }).
func (s *CachingModuleService) Clear(ctx context.Context, o caching.ClearOptions) error {
	if o.Key == "" && o.Tags == nil {
		return invalidArgument("Either key or tags must be provided")
	}
	_, err := s.dedupe(s.getClearRequestKey(o.Key, o.Tags, o.Providers), func() (any, error) {
		return nil, s.performCacheClear(ctx, o)
	})
	return err
}

// performCacheClear mirrors performCacheClear.
func (s *CachingModuleService) performCacheClear(ctx context.Context, o caching.ClearOptions) error {
	ids := o.Providers
	if len(ids) == 0 {
		ids = []string{s.defaultProviderID}
	}
	var wg sync.WaitGroup
	errs := make([]error, len(ids))
	for i, id := range ids {
		p, err := s.providers.RetrieveProvider(id)
		if err != nil {
			errs[i] = err
			continue
		}
		wg.Add(1)
		go func(i int, p types.ICachingProviderService) {
			defer wg.Done()
			errs[i] = p.Clear(ctx, types.ProviderClearOptions{Key: o.Key, Tags: o.Tags, Options: o.Options, AutoInvalidate: o.AutoInvalidate})
		}(i, p)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// ComputeKey mirrors computeKey.
func (s *CachingModuleService) ComputeKey(input any) (string, error) {
	return s.strategy.ComputeKey(input)
}

// ComputeTags mirrors computeTags.
func (s *CachingModuleService) ComputeTags(input any, opts *caching.ComputeTagsOptions) ([]string, error) {
	return s.strategy.ComputeTags(input, opts)
}
