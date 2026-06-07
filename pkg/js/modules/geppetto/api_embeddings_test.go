package geppetto

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddingsBuilderFromRegistryProfileExposesModel(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	if err := os.WriteFile(profilePath, []byte(`slug: embeddings-test
profiles:
  assistant:
    inference_settings:
      api:
        api_keys:
          openai-api-key: dummy-key
      embeddings:
        type: openai
        engine: text-embedding-3-small
        dimensions: 4
`), 0o644); err != nil {
		t.Fatalf("WriteFile profiles.yaml: %v", err)
	}
	rt := newJSRuntime(t, Options{})
	if err := rt.vm.Set("profilePath", profilePath); err != nil {
		t.Fatalf("set profilePath: %v", err)
	}
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("assistant");
		const embedder = gp.embeddings(settings);
		const model = embedder.model();
		if (model.name !== "text-embedding-3-small") throw new Error("wrong model name: " + model.name);
		if (model.dimensions !== 4) throw new Error("wrong model dims: " + model.dimensions);
	`)
}

func TestEmbeddingsBuilderUsesEmbeddingLocalCredentials(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	if err := os.WriteFile(profilePath, []byte(`slug: embeddings-local-credentials-test
profiles:
  assistant:
    inference_settings:
      embeddings:
        type: openai
        engine: text-embedding-3-small
        dimensions: 4
        api_keys:
          openai-api-key: dummy-key
`), 0o644); err != nil {
		t.Fatalf("WriteFile profiles.yaml: %v", err)
	}
	rt := newJSRuntime(t, Options{})
	if err := rt.vm.Set("profilePath", profilePath); err != nil {
		t.Fatalf("set profilePath: %v", err)
	}
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("assistant");
		const embedder = gp.embeddings(settings);
		const model = embedder.model();
		if (model.name !== "text-embedding-3-small") throw new Error("wrong model name: " + model.name);
		if (model.dimensions !== 4) throw new Error("wrong model dims: " + model.dimensions);
	`)
}

func TestEmbeddingsBuilderRejectsMissingSettings(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		let ok = false;
		try { gp.embeddings(); } catch (err) { ok = String(err).includes("requires"); }
		if (!ok) throw new Error("expected embeddings() to reject missing settings");
	`)
}
