package transport

import (
	"errors"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/security"
)

func TestEndpointAppendsOnlyPlainSegments(t *testing.T) {
	opts := security.OutboundURLOptions{}
	got, err := Endpoint("test", "https://rerank.example/api", "/v1/rerank", opts)
	if err != nil || got != "https://rerank.example/api/v1/rerank" {
		t.Fatalf("Endpoint = %q, %v", got, err)
	}
	got, err = Endpoint("test", "https://rerank.example", "completion", opts)
	if err != nil || got != "https://rerank.example/completion" {
		t.Fatalf("Endpoint without a base path = %q, %v", got, err)
	}
	for _, route := range []string{"../admin", "v1/../../x", "v1//rerank", "", "./x", "a b", "x?y=1", "%2e%2e"} {
		if _, err := Endpoint("test", "https://rerank.example/api", route, opts); !errors.Is(err, rerank.ErrInvalidRequest) {
			t.Errorf("route %q: err = %v, want ErrInvalidRequest", route, err)
		}
	}
}
