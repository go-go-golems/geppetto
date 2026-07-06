package api

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/security"
)

const multipleContentTypesExpected = `{"role":"assistant","content":[{"type":"text","text":"Text"},{"type":"image","source":{"type":"base64","media_type":"image/jpeg","data":"base64data"}},{"type":"tool_use","id":"tool1","name":"calculator","input":{"operation":"add","numbers":[1,2]}}]}`

func TestMessageSerialization(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name: "Single TextContent",
			message: Message{
				Role:    "user",
				Content: []Content{NewTextContent("Hello")},
			},
			expected: `{"role":"user","content":[{"type":"text","text":"Hello"}]}`,
		},
		{
			name: "Multiple Content types",
			message: Message{
				Role: "assistant",
				Content: []Content{
					NewTextContent("Text"),
					NewImageContent("image/jpeg", "base64data"),
					NewToolUseContent("tool1", "calculator", `{"operation":"add","numbers":[1,2]}`),
				},
			},
			expected: multipleContentTypesExpected,
		},
		{
			name: "Empty Content",
			message: Message{
				Role:    "user",
				Content: []Content{},
			},
			expected: `{"role":"user","content":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.message)
			if err != nil {
				t.Fatalf("Failed to marshal Message: %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(got))
			}
		})
	}
}

func TestMessageDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Message
		wantErr  bool
	}{
		{
			name:  "Single TextContent",
			input: `{"role":"user","content":[{"type":"text","text":"Hello"}]}`,
			expected: Message{
				Role:    "user",
				Content: []Content{TextContent{BaseContent: BaseContent{Type_: "text"}, Text: "Hello"}},
			},
		},
		{
			name:  "Multiple Content types",
			input: multipleContentTypesExpected,
			expected: Message{
				Role: "assistant",
				Content: []Content{
					TextContent{BaseContent: BaseContent{Type_: "text"}, Text: "Text"},
					ImageContent{BaseContent: BaseContent{Type_: "image"}, Source: ImageSource{Type: "base64", MediaType: "image/jpeg", Data: "base64data"}},
					ToolUseContent{BaseContent: BaseContent{Type_: "tool_use"}, ID: "tool1", Name: "calculator", Input: json.RawMessage(`{"operation":"add","numbers":[1,2]}`)},
				},
			},
		},
		{
			name:    "Unknown Content type",
			input:   `{"role":"user","content":[{"type":"unknown","data":"test"}]}`,
			wantErr: true,
		},
		{
			name:    "Malformed JSON",
			input:   `{"role":"user","content":[{"type":"text","text":"Hello"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Message
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unexpected error status: %v", err)
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Expected %+v, got %+v", tt.expected, got)
				}
			}
		})
	}
}

func TestClientOutboundOptionsDefaultDeny(t *testing.T) {
	client := NewClient("test-key", "http://127.0.0.1:9999")
	err := security.ValidateOutboundURL(client.BaseURL, client.outboundOptions())
	if err == nil {
		t.Fatal("expected default outbound options to reject plain HTTP")
	}
}

func TestClientOutboundOptionsCanAllowLocalHTTP(t *testing.T) {
	client := NewClient("test-key", "http://127.0.0.1:9999")
	client.SetOutboundURLOptions(security.OutboundURLOptions{
		AllowHTTP:          true,
		AllowLocalNetworks: true,
	})

	if err := security.ValidateOutboundURL(client.BaseURL, client.outboundOptions()); err != nil {
		t.Fatalf("expected local HTTP to be allowed after opt-in: %v", err)
	}
}
