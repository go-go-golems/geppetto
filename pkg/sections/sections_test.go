package sections

import (
	"testing"

	embeddingsconfig "github.com/go-go-golems/geppetto/pkg/embeddings/config"
)

// TestCreateGeppettoSectionsExposesCohereEmbeddingAPIKey verifies the
// advertised --embeddings-type cohere CLI choice has a credential flag in a
// registered section. Fields ending in -api-key are decoded into
// APISettings.APIKeys by the existing wildcard tag and embeddings slug loop.
func TestCreateGeppettoSectionsExposesCohereEmbeddingAPIKey(t *testing.T) {
	sections, err := CreateGeppettoSections()
	if err != nil {
		t.Fatalf("CreateGeppettoSections: %v", err)
	}

	for _, section := range sections {
		if section.GetSlug() != embeddingsconfig.EmbeddingsSlug {
			continue
		}
		if _, ok := section.GetDefinitions().Get("cohere-api-key"); !ok {
			t.Fatal("embeddings section does not expose cohere-api-key")
		}
		return
	}

	t.Fatal("CreateGeppettoSections did not register the embeddings section")
}
