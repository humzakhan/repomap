package analyzer

import (
	"context"
	"testing"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{Content: "mock"}, nil
}
func (m *mockProvider) EstimateCost(inputTokens, outputTokens int, modelID string) float64 {
	return 0
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	p := &mockProvider{name: "test-provider"}

	reg.Register(p)

	got, err := reg.Get("test-provider")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name() != "test-provider" {
		t.Errorf("expected test-provider, got %s", got.Name())
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing provider")
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockProvider{name: "a"})
	reg.Register(&mockProvider{name: "b"})

	names := reg.List()
	if len(names) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(names))
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["a"] || !found["b"] {
		t.Errorf("expected providers a and b, got %v", names)
	}
}
