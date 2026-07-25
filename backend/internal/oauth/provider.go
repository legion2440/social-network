package oauth

import (
	"context"
	"errors"
	"sort"
	"strings"
)

const ProviderGitHub = "github"

type Identity struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Username       string
	DisplayName    string
}

type ProviderInfo struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

type Provider interface {
	Name() string
	Label() string
	AuthorizationURL(state string) string
	Exchange(ctx context.Context, code string) (string, error)
	Identity(ctx context.Context, accessToken string) (*Identity, error)
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) (*Registry, error) {
	registry := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, provider := range providers {
		if provider == nil {
			return nil, errors.New("OAuth provider is required")
		}
		name := strings.TrimSpace(provider.Name())
		if name == "" {
			return nil, errors.New("OAuth provider name is required")
		}
		if _, exists := registry.providers[name]; exists {
			return nil, errors.New("duplicate OAuth provider: " + name)
		}
		registry.providers[name] = provider
	}
	return registry, nil
}

func (r *Registry) Get(name string) (Provider, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.providers[strings.TrimSpace(name)]
	return provider, ok
}

func (r *Registry) Available() []ProviderInfo {
	if r == nil {
		return []ProviderInfo{}
	}
	result := make([]ProviderInfo, 0, len(r.providers))
	for _, provider := range r.providers {
		result = append(result, ProviderInfo{Name: provider.Name(), Label: provider.Label()})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
