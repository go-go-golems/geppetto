package geppetto

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
)

type hostOnlyTestBearerSource struct{}

func (hostOnlyTestBearerSource) BearerToken(context.Context, credentials.Request) (string, error) {
	// Engine construction must not ask the source for a bearer. Returning an
	// empty value avoids placing even test credential material in this package.
	return "", nil
}

func writeOpenAIProfileWithoutStaticKey(t *testing.T) string {
	t.Helper()
	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	if err := os.WriteFile(profilePath, []byte(`slug: bearer-source-test
profiles:
  assistant:
    inference_settings:
      api:
        api_keys: {}
      chat:
        api_type: openai
        engine: gpt-4o-mini
`), 0o644); err != nil {
		t.Fatalf("WriteFile profiles.yaml: %v", err)
	}
	return profilePath
}

func TestEngineBuilderUsesHostBearerSourceWithoutStaticKey(t *testing.T) {
	rt := newJSRuntime(t, Options{BearerTokenSource: hostOnlyTestBearerSource{}})
	if err := rt.vm.Set("profilePath", writeOpenAIProfileWithoutStaticKey(t)); err != nil {
		t.Fatalf("set profilePath: %v", err)
	}

	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("assistant");
		const engine = gp.engine().inference(settings).build();
		if (!engine || engine.name !== "inferenceSettings") throw new Error("expected engine wrapper");
		if (Object.prototype.hasOwnProperty.call(gp, "bearerTokenSource")) throw new Error("source must remain host-only");
		if (Object.prototype.hasOwnProperty.call(engine, "bearerTokenSource")) throw new Error("engine must not expose source");
		if (engine.metadata && Object.prototype.hasOwnProperty.call(engine.metadata, "bearerTokenSource")) throw new Error("metadata must not expose source");
	`)
}

func TestEngineBuilderWithoutHostBearerSourceRequiresStaticKey(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	if err := rt.vm.Set("profilePath", writeOpenAIProfileWithoutStaticKey(t)); err != nil {
		t.Fatalf("set profilePath: %v", err)
	}

	_, err := rt.vm.RunString(`
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("assistant");
		gp.engine().inference(settings).build();
	`)
	if err == nil {
		t.Fatal("engine build without a host source or static key succeeded")
	}
	if !strings.Contains(err.Error(), "missing API key openai-api-key") {
		t.Fatalf("engine build returned unexpected error: %v", err)
	}
}
