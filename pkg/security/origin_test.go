package security

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestParseOrigin(t *testing.T) {
	cases := map[string]string{
		"https://api.openai.com/v1":          "https://api.openai.com:443",
		"https://API.OpenAI.com:443/v1/":     "https://api.openai.com:443",
		"HTTPS://api.openai.com":             "https://api.openai.com:443",
		"http://127.0.0.1:8080/x":            "http://127.0.0.1:8080",
		"http://localhost":                   "http://localhost:80",
		"https://user:pw@gw.example/v1":      "https://gw.example:443",
		"https://[::1]:9000/":                "https://[::1]:9000",
		"  https://gw.lunaroute.com/v1  ":    "https://gw.lunaroute.com:443",
		"https://gw.example/v1?key=SECRET#f": "https://gw.example:443",
	}
	for in, want := range cases {
		o, err := ParseOrigin(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if o.String() != want {
			t.Errorf("%q: got %s, want %s", in, o, want)
		}
	}
	for _, bad := range []string{"", " ", "/", "//", "gw.example/v1", "ftp://gw.example", "https://", "file:///etc/passwd"} {
		if _, err := ParseOrigin(bad); err == nil {
			t.Errorf("%q: want error", bad)
		}
	}
	if MustParseOrigin("https://a.example") == MustParseOrigin("http://a.example") {
		t.Fatal("scheme must be part of the origin")
	}
	if MustParseOrigin("https://a.example") == MustParseOrigin("https://a.example:8443") {
		t.Fatal("port must be part of the origin")
	}
}

// countingTransport records whether a request reached the network side.
type countingTransport struct{ n atomic.Int32 }

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	c.n.Add(1)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error { b.closed = true; return nil }

func TestPinnedTransportRefusesBeforeSending(t *testing.T) {
	next := &countingTransport{}
	tr := &PinnedTransport{Next: next, Allowed: []Origin{MustParseOrigin("https://gw.example")}}

	ok, _ := http.NewRequest(http.MethodGet, "https://gw.example/v1/x", nil)
	if _, err := tr.RoundTrip(ok); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"https://other.example/v1/x", "http://gw.example/v1/x", "https://gw.example:8443/x", "https://gw.example.evil/x"} {
		body := &trackedBody{Reader: strings.NewReader("prompt")}
		req, _ := http.NewRequest(http.MethodPost, target+"?key=SECRET", body)
		_, err := tr.RoundTrip(req)
		if !errors.Is(err, ErrOriginNotAllowed) {
			t.Fatalf("%s: want ErrOriginNotAllowed, got %v", target, err)
		}
		if strings.Contains(err.Error(), "SECRET") {
			t.Fatalf("error leaks the query: %v", err)
		}
		if !body.closed {
			t.Fatalf("%s: body not closed on refusal", target)
		}
	}
	if got := next.n.Load(); got != 1 {
		t.Fatalf("next transport called %d times, want 1 (only the allowed request)", got)
	}
	if _, err := (&PinnedTransport{Next: next}).RoundTrip(ok); !errors.Is(err, ErrOriginNotAllowed) {
		t.Fatalf("empty Allowed must refuse everything, got %v", err)
	}
}

// A redirect to another origin is refused, including custom secret headers
// that net/http would otherwise forward.
func TestPinClientRefusesCrossOriginRedirect(t *testing.T) {
	var leaked atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked.Add(1)
	}))
	defer other.Close()
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/same" {
			http.Redirect(w, r, "/final", http.StatusTemporaryRedirect)
			return
		}
		if r.URL.Path == "/final" {
			_, _ = io.WriteString(w, "final")
			return
		}
		http.Redirect(w, r, other.URL+"/steal", http.StatusTemporaryRedirect)
	}))
	defer gw.Close()

	c := PinClient(nil, MustParseOrigin(gw.URL))
	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/messages", strings.NewReader("{}"))
	req.Header.Set("x-api-key", "CL-SECRET")
	_, err := c.Do(req)
	if !errors.Is(err, ErrOriginNotAllowed) {
		t.Fatalf("want ErrOriginNotAllowed, got %v", err)
	}
	if leaked.Load() != 0 {
		t.Fatal("redirect target was contacted")
	}

	// Same-origin redirects still work.
	resp, err := c.Get(gw.URL + "/same")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(b) != "final" {
		t.Fatalf("same-origin redirect: got %q", b)
	}
}

func TestPinClientDoesNotModifyBase(t *testing.T) {
	base := &http.Client{}
	pinned := PinClient(base, MustParseOrigin("https://gw.example"))
	if base.Transport != nil {
		t.Fatal("base client modified")
	}
	if pinned == base || pinned.Transport == nil {
		t.Fatal("PinClient must return a copy with a pinned transport")
	}
	d := PinClient(http.DefaultClient, MustParseOrigin("https://gw.example"))
	if http.DefaultClient.Transport != nil || d == http.DefaultClient {
		t.Fatal("http.DefaultClient modified")
	}
	if !Allows([]Origin{MustParseOrigin("https://gw.example")}, &url.URL{Scheme: "https", Host: "GW.example:443"}) {
		t.Fatal("Allows must normalize case and default port")
	}
}
