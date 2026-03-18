package engineprofiles

import (
	"context"
	"testing"

	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func mustTestInferenceSettings(t *testing.T, apiType aitypes.ApiType, model string) *aistepssettings.InferenceSettings {
	t.Helper()
	ss, err := aistepssettings.NewInferenceSettings()
	if err != nil {
		t.Fatalf("NewInferenceSettings failed: %v", err)
	}
	ss.Chat.ApiType = &apiType
	ss.Chat.Engine = &model
	return ss
}

func mustUpsertRegistry(t *testing.T, store *InMemoryEngineProfileStore, registry *EngineProfileRegistry) {
	t.Helper()
	if err := store.UpsertRegistry(context.Background(), registry, SaveOptions{Actor: "test", Source: "test"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
}

func mustNewStoreRegistry(t *testing.T, store EngineProfileStore) *StoreRegistry {
	t.Helper()
	registry, err := NewStoreRegistry(store, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewStoreRegistry failed: %v", err)
	}
	return registry
}
