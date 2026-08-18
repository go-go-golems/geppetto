package imageparts

import (
	"encoding/base64"
	"testing"
)

func TestNormalizeImageMapURL(t *testing.T) {
	part, ok, err := NormalizeImageMap(map[string]any{"url": "https://example.com/a.png", "detail": "high"})
	if err != nil || !ok {
		t.Fatalf("NormalizeImageMap ok=%v err=%v", ok, err)
	}
	if part.URL != "https://example.com/a.png" || part.Detail != "high" {
		t.Fatalf("part = %#v", part)
	}
}

func TestNormalizeImageMapDataURL(t *testing.T) {
	part, ok, err := NormalizeImageMap(map[string]any{"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("PNG"))})
	if err != nil || !ok {
		t.Fatalf("NormalizeImageMap ok=%v err=%v", ok, err)
	}
	if part.URL != "" || part.MediaType != "image/png" || string(part.Data) != "PNG" {
		t.Fatalf("part = %#v", part)
	}
}

func TestNormalizeImageMapInlineBytes(t *testing.T) {
	part, ok, err := NormalizeImageMap(map[string]any{"media_type": "image/png", "content": []byte("PNG")})
	if err != nil || !ok {
		t.Fatalf("NormalizeImageMap ok=%v err=%v", ok, err)
	}
	if part.MediaType != "image/png" || string(part.Data) != "PNG" {
		t.Fatalf("part = %#v", part)
	}
}

func TestNormalizeImageMapInlineBase64(t *testing.T) {
	part, ok, err := NormalizeImageMap(map[string]any{"media_type": "image/jpeg", "content": base64.StdEncoding.EncodeToString([]byte("JPEG"))})
	if err != nil || !ok {
		t.Fatalf("NormalizeImageMap ok=%v err=%v", ok, err)
	}
	if part.MediaType != "image/jpeg" || string(part.Data) != "JPEG" {
		t.Fatalf("part = %#v", part)
	}
}

func TestNormalizeImageMapMissingMediaType(t *testing.T) {
	_, _, err := NormalizeImageMap(map[string]any{"content": []byte("PNG")})
	if err == nil {
		t.Fatalf("expected media_type error")
	}
}

func TestNormalizeImageMapFileReferences(t *testing.T) {
	part, ok, err := NormalizeImageMap(map[string]any{"file_id": "file_123"})
	if err != nil || !ok || part.FileID != "file_123" {
		t.Fatalf("file id part=%#v ok=%v err=%v", part, ok, err)
	}
	part, ok, err = NormalizeImageMap(map[string]any{"file_uri": "gs://bucket/a.png", "media_type": "image/png"})
	if err != nil || !ok || part.FileURI != "gs://bucket/a.png" || part.MediaType != "image/png" {
		t.Fatalf("file uri part=%#v ok=%v err=%v", part, ok, err)
	}
}

func TestDataURL(t *testing.T) {
	if got := DataURL("image/png", []byte("PNG")); got != "data:image/png;base64,UE5H" {
		t.Fatalf("DataURL = %q", got)
	}
}

func TestNormalizeImageMapAcceptsGenericNumericContentSlice(t *testing.T) {
	// Shape produced by decoding a YAML/JSON-persisted []byte content value.
	part, ok, err := NormalizeImageMap(map[string]any{
		"media_type": "image/png",
		"content":    []any{80, 78, 71},
	})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if string(part.Data) != "PNG" || part.MediaType != "image/png" {
		t.Fatalf("part = %#v", part)
	}
	if _, _, err := NormalizeImageMap(map[string]any{"media_type": "image/png", "content": []any{"x"}}); err == nil {
		t.Fatalf("expected error for non-numeric content slice")
	}
}
