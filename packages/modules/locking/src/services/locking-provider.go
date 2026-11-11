package services

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/types"
)

// LockingProviderService mirrors LockingProviderService: the registered
// providers, keyed by `lp_<id>`.
type LockingProviderService struct {
	mu        sync.RWMutex
	providers map[string]lockingtypes.ILockingProvider
	order     []string
	logger    *slog.Logger
}

// NewLockingProviderService builds an empty provider registry.
func NewLockingProviderService(logger *slog.Logger) *LockingProviderService {
	if logger == nil {
		logger = slog.Default()
	}
	return &LockingProviderService{providers: map[string]lockingtypes.ILockingProvider{}, logger: logger}
}

// Register registers a provider under `lp_<id>`.
func (s *LockingProviderService) Register(id string, provider lockingtypes.ILockingProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := types.LockingProviderRegistrationPrefix + id
	if _, ok := s.providers[key]; !ok {
		s.order = append(s.order, key)
	}
	s.providers[key] = provider
}

// RetrieveProviderRegistration mirrors retrieveProviderRegistration.
func (s *LockingProviderService) RetrieveProviderRegistration(providerID string) (lockingtypes.ILockingProvider, error) {
	s.mu.RLock()
	p, ok := s.providers[types.LockingProviderRegistrationPrefix+providerID]
	s.mu.RUnlock()
	if !ok {
		key := types.LockingProviderRegistrationPrefix + providerID
		s.logger.Error(fmt.Sprintf("AwilixResolutionError: Could not resolve '%s'.", key))
		return nil, errors.New("\n Unable to retrieve the locking provider with id: " + providerID + "\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.")
	}
	return p, nil
}

// Shutdown mirrors shutdown: every provider holding resources is shut down.
func (s *LockingProviderService) Shutdown() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[lockingtypes.ILockingProvider]bool{}
	var errs []error
	for _, key := range s.order {
		p := s.providers[key]
		if seen[p] {
			continue
		}
		seen[p] = true
		if sd, ok := p.(lockingtypes.Shutdowner); ok {
			if err := sd.Shutdown(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
