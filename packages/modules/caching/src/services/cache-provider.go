package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/hitkey-io/medusago/packages/modules/caching/src/types"
)

// CacheProviderService mirrors CacheProviderService: the providers
// registered under lp_<id>.
type CacheProviderService struct {
	logger    *slog.Logger
	mu        sync.RWMutex
	providers map[string]types.ICachingProviderService
	order     []string
}

// NewCacheProviderService builds an empty registry.
func NewCacheProviderService(logger *slog.Logger) *CacheProviderService {
	if logger == nil {
		logger = slog.Default()
	}
	return &CacheProviderService{logger: logger, providers: map[string]types.ICachingProviderService{}}
}

// Register registers a provider under its id.
func (s *CacheProviderService) Register(id string, p types.ICachingProviderService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.providers[id]; !ok {
		s.order = append(s.order, id)
	}
	s.providers[id] = p
}

// RetrieveProvider mirrors retrieveProvider.
func (s *CacheProviderService) RetrieveProvider(id string) (types.ICachingProviderService, error) {
	s.mu.RLock()
	p, ok := s.providers[id]
	s.mu.RUnlock()
	if !ok {
		msg := fmt.Sprintf("\n Unable to retrieve the caching provider with id: %s\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.", id)
		s.logger.Error(fmt.Sprintf("AwilixResolutionError: Could not resolve '%s%s'.", types.CachingProviderRegistrationPrefix, id))
		return nil, fmt.Errorf("%s", msg)
	}
	return p, nil
}

// Shutdown mirrors shutdown: every provider is shut down.
func (s *CacheProviderService) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	var list []types.ICachingProviderService
	for _, id := range s.order {
		list = append(list, s.providers[id])
	}
	s.mu.RUnlock()
	for _, p := range list {
		if sd, ok := p.(types.Shutdowner); ok {
			_ = sd.Shutdown(ctx)
		}
	}
	return nil
}
