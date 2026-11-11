package services

import (
	"fmt"
	"strings"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// SearchProviderService mirrors SearchProviderService: the registered
// providers keyed by identifier (each index selects one).
type SearchProviderService struct {
	order     []string
	providers map[string]searchtypes.Provider
}

// NewSearchProviderService registers the providers (in registration
// order); registering one identifier twice throws.
func NewSearchProviderService(providers []searchtypes.Provider) (*SearchProviderService, error) {
	s := &SearchProviderService{providers: map[string]searchtypes.Provider{}}
	for _, provider := range providers {
		if provider == nil || provider.Identifier() == "" {
			continue
		}
		identifier := provider.Identifier()
		if _, ok := s.providers[identifier]; ok {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`The search provider "%s" is registered twice. Register each provider once; index definitions select it through their "provider" field.`, identifier))
		}
		s.order = append(s.order, identifier)
		s.providers[identifier] = provider
	}
	return s, nil
}

// Retrieve mirrors retrieve.
func (s *SearchProviderService) Retrieve(identifier string) (searchtypes.Provider, error) {
	provider, ok := s.providers[identifier]
	if !ok {
		registered := strings.Join(s.order, ", ")
		if registered == "" {
			registered = "(none)"
		}
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Search provider "%s" is not registered. Registered providers: %s`, identifier, registered))
	}
	return provider, nil
}

// List mirrors list.
func (s *SearchProviderService) List() []searchtypes.Provider {
	out := make([]searchtypes.Provider, len(s.order))
	for i, id := range s.order {
		out[i] = s.providers[id]
	}
	return out
}

// GetDefaultIdentifier mirrors getDefaultIdentifier.
func (s *SearchProviderService) GetDefaultIdentifier(configured string) (string, error) {
	if configured != "" {
		if _, err := s.Retrieve(configured); err != nil {
			return "", err
		}
		return configured, nil
	}
	if len(s.order) == 1 {
		return s.order[0], nil
	}
	if len(s.order) == 0 {
		return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "The Search Module requires at least one provider to be registered")
	}
	return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Several search providers are registered (%s), so "default_provider" must be set, or every index definition must name its provider`, strings.Join(s.order, ", ")))
}
