package imageparts

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type ImagePart struct {
	MediaType string
	URL       string
	Data      []byte
	FileID    string
	FileURI   string
	Detail    string
}

func NormalizeImageMap(img map[string]any) (ImagePart, bool, error) {
	if len(img) == 0 {
		return ImagePart{}, false, nil
	}
	part := ImagePart{Detail: NormalizeDetail(firstNonEmptyString(img["detail"]))}
	if url := firstNonEmptyString(img["url"], img["image_url"]); url != "" {
		if mediaType, data, ok, err := DecodeDataURL(url); err != nil {
			return ImagePart{}, false, err
		} else if ok {
			part.MediaType = mediaType
			part.Data = data
			return part, true, nil
		}
		part.URL = url
		if mt := firstNonEmptyString(img["media_type"]); mt != "" {
			part.MediaType = mt
		}
		return part, true, nil
	}
	if fileID := firstNonEmptyString(img["file_id"]); fileID != "" {
		part.FileID = fileID
		return part, true, nil
	}
	if fileURI := firstNonEmptyString(img["file_uri"]); fileURI != "" {
		part.FileURI = fileURI
		part.MediaType = firstNonEmptyString(img["media_type"])
		return part, true, nil
	}
	if raw, ok := img["content"]; ok && raw != nil {
		if dataURL, ok := raw.(string); ok && strings.HasPrefix(strings.TrimSpace(dataURL), "data:") {
			mediaType, data, ok, err := DecodeDataURL(dataURL)
			if err != nil {
				return ImagePart{}, false, err
			}
			if ok {
				part.MediaType = mediaType
				part.Data = data
				return part, true, nil
			}
		}
		mediaType := firstNonEmptyString(img["media_type"])
		if mediaType == "" {
			return ImagePart{}, false, fmt.Errorf("image content requires media_type")
		}
		data, err := contentBytes(raw)
		if err != nil {
			return ImagePart{}, false, err
		}
		if len(data) == 0 {
			return ImagePart{}, false, nil
		}
		part.MediaType = mediaType
		part.Data = data
		return part, true, nil
	}
	return ImagePart{}, false, nil
}

func NormalizeDetail(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "low", "high", "auto", "original":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "auto"
	}
}

func DataURL(mediaType string, data []byte) string {
	if mediaType == "" || len(data) == 0 {
		return ""
	}
	return fmt.Sprintf("data:%s;base64,%s", mediaType, base64.StdEncoding.EncodeToString(data))
}

func DecodeDataURL(value string) (string, []byte, bool, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "data:") {
		return "", nil, false, nil
	}
	comma := strings.Index(value, ",")
	if comma < 0 {
		return "", nil, false, fmt.Errorf("invalid data URL: missing comma")
	}
	meta := strings.TrimPrefix(value[:comma], "data:")
	payload := value[comma+1:]
	parts := strings.Split(meta, ";")
	mediaType := strings.TrimSpace(parts[0])
	if mediaType == "" {
		return "", nil, false, fmt.Errorf("invalid data URL: missing media type")
	}
	isBase64 := false
	for _, p := range parts[1:] {
		if strings.EqualFold(strings.TrimSpace(p), "base64") {
			isBase64 = true
			break
		}
	}
	if !isBase64 {
		return "", nil, false, fmt.Errorf("invalid data URL: only base64 image data is supported")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload))
	if err != nil {
		return "", nil, false, fmt.Errorf("decode data URL image: %w", err)
	}
	return mediaType, data, true, nil
}

func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		switch v := value.(type) {
		case string:
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		case []byte:
			if s := strings.TrimSpace(string(v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func contentBytes(raw any) ([]byte, error) {
	switch v := raw.(type) {
	case []byte:
		return append([]byte(nil), v...), nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, nil
		}
		decoded, err := base64.StdEncoding.DecodeString(s)
		if err == nil {
			return decoded, nil
		}
		return []byte(s), nil
	default:
		return nil, fmt.Errorf("unsupported image content type %T", raw)
	}
}
