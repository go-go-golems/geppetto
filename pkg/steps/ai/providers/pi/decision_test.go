package pi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/geppetto/pkg/decision"
)

// Decider sends Pi's credential and models.json headers to the provider's
// base URL + /systemone.
func TestDeciderUsesPiCredentialsAndBaseURL(t *testing.T) {
	var path, auth, team string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, auth, team = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("x-team")
		_, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"model":"djev","answers":{"q":{"type":"noul","noul":0.9}},"usage":{"input_tokens":5,"output_tokens":1}}`)
	}))
	defer srv.Close()
	a, err := LoadAgentDir(agentDir(t, srv.URL+"/v1"), env(map[string]string{"LUNA_KEY": "test-key", "TEAM": "t1"}), WithLocalProviders())
	require.NoError(t, err)

	d, err := a.Decider(context.Background(), "lunaroute/djev", DecisionOptions{})
	require.NoError(t, err)
	assert.Equal(t, decision.Model{Provider: "systemone", Name: "djev"}, d.Model())
	r, err := d.Decide(context.Background(), decision.Request{State: "s", Questions: map[string]decision.Question{"q": {Type: decision.Noul, Instructions: "?"}}})
	require.NoError(t, err)
	assert.Equal(t, 0.9, r.Answers["q"].Probability)
	assert.Equal(t, "/v1/systemone", path)
	assert.Equal(t, "Bearer test-key", auth)
	assert.Equal(t, "t1", team)
}

func TestDeciderErrors(t *testing.T) {
	a, err := LoadAgentDir(agentDir(t, "https://gw.example/v1"), env(map[string]string{}))
	require.NoError(t, err)
	_, err = a.Decider(context.Background(), "lunaroute/nope", DecisionOptions{})
	assert.ErrorIs(t, err, ErrUnknownModel)

	// No credential: $LUNA_KEY is unset.
	_, err = a.Decider(context.Background(), "lunaroute/djev", DecisionOptions{})
	require.Error(t, err)

	// Without WithLocalProviders, a plain-HTTP provider URL is refused.
	b, err := LoadAgentDir(agentDir(t, "http://127.0.0.1:1/v1"), env(map[string]string{"LUNA_KEY": "k", "TEAM": "t"}))
	require.NoError(t, err)
	_, err = b.Decider(context.Background(), "lunaroute/djev", DecisionOptions{})
	assert.ErrorIs(t, err, decision.ErrInvalidRequest)

	// A non-LunaRoute model without a baseUrl must not fall back to
	// LunaRoute's endpoint, which would send that provider's credential there.
	c, err := LoadAgentDir(agentDir(t, "https://gw.example/v1"), env(map[string]string{}))
	require.NoError(t, err)
	_, err = c.Decider(context.Background(), "amazon-bedrock/claude", DecisionOptions{})
	assert.ErrorIs(t, err, decision.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "no baseUrl")

	// Blank or "/" base URLs normalize to empty in systemone.New and must be
	// treated as missing here too, not as an explicit endpoint.
	for _, u := range []string{" ", "/", " // "} {
		dir := agentDir(t, "https://gw.example/v1")
		writeFile(t, dir, "models.json", map[string]any{"providers": map[string]any{"fireworks": map[string]any{
			"baseUrl": u, "api": "openai-completions", "apiKey": "fw-key",
			"models": []any{map[string]any{"id": "djev"}},
		}}})
		d, err := LoadAgentDir(dir, env(map[string]string{}))
		require.NoError(t, err)
		_, err = d.Decider(context.Background(), "fireworks/djev", DecisionOptions{})
		require.ErrorIs(t, err, decision.ErrInvalidRequest, "baseUrl %q", u)
		assert.Contains(t, err.Error(), "no baseUrl", "baseUrl %q", u)
	}
}
