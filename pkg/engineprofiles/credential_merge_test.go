package engineprofiles

import (
	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProfileCredentialOverridesAreAtomic(t *testing.T) {
	for _, section := range []string{"api", "embeddings"} {
		for _, variant := range []string{"key-only", "url-only", "pair", "blank", "model-only"} {
			t.Run(section+"/"+variant, func(t *testing.T) {
				baseKeys, baseURLs := map[string]string{"openai-api-key": "PARENT"}, map[string]string{"openai-base-url": "https://parent.example/v1"}
				keys, urls := map[string]string{}, map[string]string{}
				switch variant {
				case "key-only":
					keys["openai-api-key"] = "LOCAL"
				case "url-only":
					urls["openai-base-url"] = "https://local.example/v1"
				case "pair":
					keys["openai-api-key"] = "LOCAL"
					urls["openai-base-url"] = "https://local.example/v1"
				case "blank":
					keys["openai-api-key"] = " "
					urls["openai-base-url"] = " "
				}
				base, overlay := &settings.InferenceSettings{}, &settings.InferenceSettings{}
				if section == "api" {
					base.API = &settings.APISettings{APIKeys: baseKeys, BaseUrls: baseURLs}
					overlay.API = &settings.APISettings{APIKeys: keys, BaseUrls: urls}
				} else {
					base.Embeddings = &config.EmbeddingsConfig{APIKeys: baseKeys, BaseURLs: baseURLs}
					overlay.Embeddings = &config.EmbeddingsConfig{APIKeys: keys, BaseURLs: urls}
				}
				result, err := MergeInferenceSettings(base, overlay)
				if variant == "key-only" || variant == "url-only" {
					require.Error(t, err)
					require.NotContains(t, err.Error(), "PARENT")
					require.NotContains(t, err.Error(), "LOCAL")
					return
				}
				require.NoError(t, err)
				var gotKeys, gotURLs map[string]string
				if section == "api" {
					gotKeys, gotURLs = result.API.APIKeys, result.API.BaseUrls
				} else {
					gotKeys, gotURLs = result.Embeddings.APIKeys, result.Embeddings.BaseURLs
				}
				wantKey, wantURL := "PARENT", "https://parent.example/v1"
				if variant == "pair" {
					wantKey, wantURL = "LOCAL", "https://local.example/v1"
				}
				require.Equal(t, wantKey, gotKeys["openai-api-key"])
				require.Equal(t, wantURL, gotURLs["openai-base-url"])
				require.Equal(t, "PARENT", baseKeys["openai-api-key"])
				require.Equal(t, "https://parent.example/v1", baseURLs["openai-base-url"])
			})
		}
	}
}
