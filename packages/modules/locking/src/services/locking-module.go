// Package services mirrors @medusajs/locking/src/services.
package services

import (
	"context"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
)

// LockingModuleService mirrors LockingModuleService: every call goes to the
// provider named in the args, or to the default provider.
type LockingModuleService struct {
	providerService   *LockingProviderService
	defaultProviderID string
}

var _ lockingtypes.ILockingModule = (*LockingModuleService)(nil)

// NewLockingModuleService builds the service over the registered providers.
func NewLockingModuleService(providers *LockingProviderService, defaultProviderID string) *LockingModuleService {
	return &LockingModuleService{providerService: providers, defaultProviderID: defaultProviderID}
}

// DefaultProviderID is the id of the default provider.
func (s *LockingModuleService) DefaultProviderID() string { return s.defaultProviderID }

func (s *LockingModuleService) provider(id string) (lockingtypes.ILockingProvider, error) {
	if id == "" {
		id = s.defaultProviderID
	}
	return s.providerService.RetrieveProviderRegistration(id)
}

// OnApplicationShutdown mirrors __hooks.onApplicationShutdown.
func (s *LockingModuleService) OnApplicationShutdown(ctx context.Context) error {
	return s.providerService.Shutdown()
}

// Execute mirrors execute.
func (s *LockingModuleService) Execute(ctx context.Context, keys []string, job lockingtypes.Job, args lockingtypes.ExecuteArgs) (any, error) {
	p, err := s.provider(args.Provider)
	if err != nil {
		return nil, err
	}
	return p.Execute(ctx, keys, job, args)
}

// Acquire mirrors acquire.
func (s *LockingModuleService) Acquire(ctx context.Context, keys []string, args lockingtypes.AcquireArgs) error {
	p, err := s.provider(args.Provider)
	if err != nil {
		return err
	}
	return p.Acquire(ctx, keys, args)
}

// Release mirrors release.
func (s *LockingModuleService) Release(ctx context.Context, keys []string, args lockingtypes.ReleaseArgs) (bool, error) {
	p, err := s.provider(args.Provider)
	if err != nil {
		return false, err
	}
	return p.Release(ctx, keys, args)
}

// ReleaseAll mirrors releaseAll.
func (s *LockingModuleService) ReleaseAll(ctx context.Context, args lockingtypes.ReleaseArgs) error {
	p, err := s.provider(args.Provider)
	if err != nil {
		return err
	}
	return p.ReleaseAll(ctx, args)
}
