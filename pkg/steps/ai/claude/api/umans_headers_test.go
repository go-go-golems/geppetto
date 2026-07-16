package api

import (
	"net/http"
	"testing"
)

func TestClientSetHeadersSupportsUmansDualAuthentication(t *testing.T) {
	client := NewClient("test-api-key", "https://api.code.umans.ai")
	client.SetBearerAuthorization("test-api-key")
	request, err := http.NewRequest(http.MethodPost, "https://api.code.umans.ai/v1/messages", nil)
	if err != nil {
		t.Fatal(err)
	}
	client.setHeaders(request)
	if got := request.Header.Get("x-api-key"); got != "test-api-key" {
		t.Fatalf("x-api-key = %q", got)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer test-api-key" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := request.Header.Get("anthropic-version"); got != defaultAPIVersion {
		t.Fatalf("anthropic-version = %q", got)
	}
}
