package get_conversation

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"strings"
	"text/template"
	"time"
)

type Renderer struct {
	WithMetadata bool
	Concise      bool
	RenameRoles  map[string]string
}

type TemplateData struct {
	Title         string
	CreateTime    string
	URL           string
	Concise       bool
	WithMetadata  bool
	Conversations []ConversationData
}

type ConversationData struct {
	ID              string
	Role            string
	AuthorMetadata  map[string]interface{}
	ContentType     string
	Status          string
	EndTurn         bool
	Weight          float64
	Recipient       string
	Parts           []string
	Children        []string
	MessageMetadata map[string]interface{}
	Skip            bool
}

func (r *Renderer) PrintConversation(url string, response ServerResponseData, linearConversation []Conversation) {
	var buf bytes.Buffer

	const tpl = `
# {{.Title}}
Created at: {{.CreateTime}}
URL: {{.URL}}

{{range .Conversations -}}
{{  template "messageDetails" (list $ .) -}}
{{end -}}
`

	const messageDetailsTemplate = `
{{- $ := index . 0 -}}{{- $topLevel := $ -}}
{{ with (index . 1) -}}
{{if not .Skip -}}
{{if not $.Concise -}}
### Message Details:

{{template "verboseMessageDetails" (list $topLevel .)}}
{{else -}}
**{{.Role}}**: {{ range .Parts -}}
{{.}}
{{end }}
{{end -}}
---
{{end }}
{{end -}}
`

	const verboseMessageDetailsSubTemplate = `
{{- $ := index . 0 -}}
{{ with (index . 1) -}}
- **ID**: {{.ID}}
- **Author Role**: {{.Role}}
{{if $.WithMetadata}}{{template "authorMetadata" .}}{{end -}}
- **Content Type**: {{.ContentType}}
- **Status**: {{.Status}}
- **End Turn**: {{.EndTurn}}
- **Weight**: {{.Weight}}
- **Recipient**: {{.Recipient}}
{{if .Children -}}
- **Children IDs**:{{range .Children }}
  - {{.}}
{{- end -}}
{{end }}
- **Parts**: {{range .Parts}}
  - {{.}}
{{end -}}
{{if $.WithMetadata}}{{template "messageMetadata" .}}{{end -}}
{{end -}}
`

	const authorMetadataSubTemplate = `- **Author Metadata**: {{range $key, $value := .AuthorMetadata}}
  - {{$key}}: {{$value}}
{{- end}}
`

	const messageMetadataSubTemplate = `- **Message Metadata**: {{range $key, $value := .MessageMetadata}}
  - {{$key}}: {{$value}}
{{- end}}
`

	// Parsing the templates
	t := template.Must(template.New("messageDetails").Funcs(sprig.FuncMap()).Parse(messageDetailsTemplate))
	t, _ = t.New("verboseMessageDetails").Parse(verboseMessageDetailsSubTemplate)
	t, _ = t.New("authorMetadata").Parse(authorMetadataSubTemplate)
	t, _ = t.New("messageMetadata").Parse(messageMetadataSubTemplate)
	t, _ = t.New("conversation").Parse(tpl)
	//
	data := TemplateData{
		Title:        response.Title,
		CreateTime:   time.Unix(int64(response.CreateTime), 0).Format(time.RFC3339),
		URL:          url,
		Concise:      r.Concise,
		WithMetadata: r.WithMetadata,
	}

	for _, conversation := range linearConversation {
		parts := conversation.Message.Content.Parts
		if len(parts) == 0 {
			continue
		}

		partContent := strings.Join(parts, "\n")
		if strings.TrimSpace(partContent) == "" {
			continue
		}

		role := conversation.Message.Author.Role
		if r.RenameRoles != nil {
			if newRole, ok := r.RenameRoles[role]; ok {
				role = newRole
			}
		}

		convoData := ConversationData{
			ID:              conversation.Message.ID,
			Role:            role,
			AuthorMetadata:  conversation.Message.Author.Metadata,
			ContentType:     conversation.Message.Content.ContentType,
			Status:          conversation.Message.Status,
			EndTurn:         conversation.Message.EndTurn,
			Weight:          conversation.Message.Weight,
			Recipient:       conversation.Message.Recipient,
			Parts:           parts,
			Children:        conversation.Children,
			MessageMetadata: conversation.Message.Metadata,
			Skip:            false,
		}
		data.Conversations = append(data.Conversations, convoData)
	}

	if err := t.Execute(&buf, data); err != nil {
		fmt.Println("Error executing template:", err)
		return
	}

	fmt.Println(buf.String())
}
