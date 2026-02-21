package doc

import (
	"testing"

	"github.com/go-go-golems/glazed/pkg/help"
)

func TestAddDocToHelpSystem_LoadsJSEntries(t *testing.T) {
	hs := help.NewHelpSystem()
	if err := AddDocToHelpSystem(hs); err != nil {
		t.Fatalf("AddDocToHelpSystem failed: %v", err)
	}

	slugs := []string{
		"geppetto-js-api-reference",
		"geppetto-js-api-user-guide",
		"geppetto-js-api-getting-started",
	}

	for _, slug := range slugs {
		section, err := hs.GetSectionWithSlug(slug)
		if err != nil {
			t.Fatalf("expected slug %q to load: %v", slug, err)
		}
		if section == nil {
			t.Fatalf("expected slug %q to resolve to section", slug)
		}
		if section.Title == "" {
			t.Fatalf("expected slug %q to have title", slug)
		}
	}
}
