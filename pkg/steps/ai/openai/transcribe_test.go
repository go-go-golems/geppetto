package openai

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/stretchr/testify/require"
)

func TestWhisperUsesBoundConnection(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		t.Run(map[bool]string{false: "multipart", true: "redirect"}[redirect], func(t *testing.T) {
			calls := 0
			c, err := connection.New(connection.Config{Provider: "openai", BaseURL: "https://gateway.example/v1", Routes: Routes(), Auth: connection.StaticBearer("WHISPER-SECRET"), Headers: http.Header{"X-Team": {"team"}}, HTTPClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "gateway.example", req.URL.Host)
				require.Equal(t, "/v1/audio/transcriptions", req.URL.Path)
				require.Equal(t, "Bearer WHISPER-SECRET", req.Header.Get("Authorization"))
				require.Equal(t, "team", req.Header.Get("X-Team"))
				_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
				require.NoError(t, err)
				form, err := multipart.NewReader(req.Body, params["boundary"]).ReadForm(1 << 20)
				require.NoError(t, err)
				defer func() { _ = form.RemoveAll() }()
				require.Equal(t, []string{"whisper-1"}, form.Value["model"])
				require.Len(t, form.File["file"], 1)
				if redirect {
					return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"https://foreign.example/stolen"}}, Body: http.NoBody, Request: req}, nil
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"text":"hello","language":"en","duration":1}`)), Request: req}, nil
			})}})
			require.NoError(t, err)
			client, err := NewTranscriptionClient(c)
			require.NoError(t, err)
			audio := filepath.Join(t.TempDir(), "test.mp3")
			require.NoError(t, os.WriteFile(audio, []byte("ID3dummy audio"), 0o600))
			result, err := client.TranscribeFile(t.Context(), audio)
			require.NoError(t, err)
			if redirect {
				require.Error(t, result.Error)
			} else {
				require.NoError(t, result.Error)
				require.Equal(t, "hello", result.Response.Text)
			}
			require.Equal(t, 1, calls)
		})
	}
}
