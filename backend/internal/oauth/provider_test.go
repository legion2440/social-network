package oauth

import (
	"context"
	"testing"
)

type stubProvider struct {
	name  string
	label string
}

func (p stubProvider) Name() string                                        { return p.name }
func (p stubProvider) Label() string                                       { return p.label }
func (p stubProvider) AuthorizationURL(string) string                      { return "" }
func (p stubProvider) Exchange(context.Context, string) (string, error)    { return "", nil }
func (p stubProvider) Identity(context.Context, string) (*Identity, error) { return nil, nil }

func TestRegistryReturnsSortedPublicProviders(t *testing.T) {
	registry, err := NewRegistry(
		stubProvider{name: "zeta", label: "Zeta"},
		stubProvider{name: "alpha", label: "Alpha"},
	)
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	available := registry.Available()
	if len(available) != 2 || available[0].Name != "alpha" || available[1].Name != "zeta" {
		t.Fatalf("unexpected provider list: %+v", available)
	}
}

func TestRegistryRejectsDuplicateProviders(t *testing.T) {
	if _, err := NewRegistry(
		stubProvider{name: "github", label: "GitHub"},
		stubProvider{name: "github", label: "Duplicate"},
	); err == nil {
		t.Fatal("expected duplicate provider error")
	}
}
