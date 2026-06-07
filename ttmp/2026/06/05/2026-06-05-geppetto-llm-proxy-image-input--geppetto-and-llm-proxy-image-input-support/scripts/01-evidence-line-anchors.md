---
Title: Evidence Line Anchors
Ticket: 2026-06-05-geppetto-llm-proxy-image-input
Status: active
Topics:
    - geppetto
    - providers
    - openai-compatibility
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: Line-anchored source excerpts for image input support across llm-proxy and Geppetto providers.
LastUpdated: 2026-06-05T18:10:00-04:00
WhatFor: Use as evidence for the image input guide and gap matrix.
WhenToUse: Read when reviewing claims about current multimodal support.
---

# Evidence line anchors

## geppetto/pkg/turns/helpers_blocks.go:20-42
```
    20			Payload: map[string]any{PayloadKeyText: text},
    21		}
    22	}
    23	
    24	// NewUserMultimodalBlock creates a user block with text and optional images.
    25	// images is a slice of maps with keys:
    26	//   - "media_type" (string) for inline content
    27	//   - either "url" (string), "content" ([]byte/base64), or provider-specific "file_id" (string)
    28	//   - optional "detail" for providers that support image detail selection
    29	func NewUserMultimodalBlock(text string, images []map[string]any) Block {
    30		payload := map[string]any{PayloadKeyText: text}
    31		if len(images) > 0 {
    32			payload[PayloadKeyImages] = images
    33		}
    34		return Block{
    35			ID:      uuid.NewString(),
    36			Kind:    BlockKindUser,
    37			Role:    RoleUser,
    38			Payload: payload,
    39		}
    40	}
    41	
    42	// NewAssistantTextBlock returns a Block representing assistant LLM text output.
```

## geppetto/pkg/steps/ai/openai_responses/helpers.go:600-670
```
   600		}
   601		if role == "assistant" {
   602			return parts
   603		}
   604		if imgs, ok := payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
   605			for _, img := range imgs {
   606				if part, ok := responsesImagePartFromMap(img); ok {
   607					parts = append(parts, part)
   608				}
   609			}
   610		}
   611		return parts
   612	}
   613	
   614	func responsesImagePartFromMap(img map[string]any) (responsesContentPart, bool) {
   615		if len(img) == 0 {
   616			return responsesContentPart{}, false
   617		}
   618		detail := normalizeResponsesImageDetail(img["detail"])
   619		if imageURL := firstNonEmptyString(img["url"], img["image_url"]); imageURL != "" {
   620			return responsesContentPart{Type: "input_image", ImageURL: imageURL, Detail: detail}, true
   621		}
   622		if raw, ok := img["content"]; ok && raw != nil {
   623			if dataURL, ok := raw.(string); ok && strings.HasPrefix(strings.TrimSpace(dataURL), "data:") {
   624				return responsesContentPart{Type: "input_image", ImageURL: strings.TrimSpace(dataURL), Detail: detail}, true
   625			}
   626			if mediaType := firstNonEmptyString(img["media_type"]); mediaType != "" {
   627				if base64Content := base64ImageContent(raw); base64Content != "" {
   628					return responsesContentPart{
   629						Type:     "input_image",
   630						ImageURL: fmt.Sprintf("data:%s;base64,%s", mediaType, base64Content),
   631						Detail:   detail,
   632					}, true
   633				}
   634			}
   635		}
   636		if fileID := firstNonEmptyString(img["file_id"]); fileID != "" {
   637			return responsesContentPart{Type: "input_image", FileID: fileID, Detail: detail}, true
   638		}
   639		return responsesContentPart{}, false
   640	}
   641	
   642	func firstNonEmptyString(values ...any) string {
   643		for _, value := range values {
   644			switch v := value.(type) {
   645			case string:
   646				if s := strings.TrimSpace(v); s != "" {
   647					return s
   648				}
   649			case []byte:
   650				if s := strings.TrimSpace(string(v)); s != "" {
   651					return s
   652				}
   653			}
   654		}
   655		return ""
   656	}
   657	
   658	func base64ImageContent(raw any) string {
   659		switch v := raw.(type) {
   660		case []byte:
   661			if len(v) == 0 {
   662				return ""
   663			}
   664			return base64.StdEncoding.EncodeToString(v)
   665		case string:
   666			return strings.TrimSpace(v)
   667		default:
   668			return ""
   669		}
   670	}
```

## geppetto/pkg/steps/ai/openai_responses/helpers_test.go:40-132
```
    40			t.Fatalf("user content must have single input_text part")
    41		}
    42	}
    43	
    44	func TestBuildInputItemsFromTurn_UserMessageWithImageURL(t *testing.T) {
    45		turn := &turns.Turn{Blocks: []turns.Block{
    46			turns.NewUserMultimodalBlock("What is in this image?", []map[string]any{{
    47				"media_type": "image/png",
    48				"url":        "https://example.com/reference.png",
    49			}}),
    50		}}
    51	
    52		got := buildInputItemsFromTurn(turn)
    53		if len(got) != 1 {
    54			t.Fatalf("expected 1 item, got %d", len(got))
    55		}
    56		if got[0].Role != "user" || got[0].Type != "" {
    57			t.Fatalf("expected user role-based message, got type=%q role=%q", got[0].Type, got[0].Role)
    58		}
    59		if len(got[0].Content) != 2 {
    60			t.Fatalf("expected text + image content parts, got %#v", got[0].Content)
    61		}
    62		if got[0].Content[0].Type != "input_text" || got[0].Content[0].Text != "What is in this image?" {
    63			t.Fatalf("unexpected first content part: %#v", got[0].Content[0])
    64		}
    65		if got[0].Content[1].Type != "input_image" {
    66			t.Fatalf("expected second part to be input_image, got %#v", got[0].Content[1])
    67		}
    68		if got[0].Content[1].ImageURL != "https://example.com/reference.png" {
    69			t.Fatalf("expected image URL to round-trip, got %#v", got[0].Content[1])
    70		}
    71		if got[0].Content[1].Detail != "auto" {
    72			t.Fatalf("expected image detail to default to auto, got %#v", got[0].Content[1])
    73		}
    74	}
    75	
    76	func TestBuildInputItemsFromTurn_UserMessageWithInlineImageBytes(t *testing.T) {
    77		turn := &turns.Turn{Blocks: []turns.Block{
    78			turns.NewUserMultimodalBlock("Compare this", []map[string]any{{
    79				"media_type": "image/png",
    80				"content":    []byte("PNG"),
    81				"detail":     "high",
    82			}}),
    83		}}
    84	
    85		got := buildInputItemsFromTurn(turn)
    86		if len(got) != 1 {
    87			t.Fatalf("expected 1 item, got %d", len(got))
    88		}
    89		if len(got[0].Content) != 2 {
    90			t.Fatalf("expected text + image content parts, got %#v", got[0].Content)
    91		}
    92		expectedDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("PNG"))
    93		if got[0].Content[1].Type != "input_image" || got[0].Content[1].ImageURL != expectedDataURL {
    94			t.Fatalf("expected inline bytes to become a data URL, got %#v", got[0].Content[1])
    95		}
    96		if got[0].Content[1].Detail != "high" {
    97			t.Fatalf("expected image detail to preserve valid value, got %#v", got[0].Content[1])
    98		}
    99	}
   100	
   101	func TestBuildInputItemsFromTurn_UserMessageWithMixedTextAndMultipleImages(t *testing.T) {
   102		turn := &turns.Turn{Blocks: []turns.Block{
   103			turns.NewUserMultimodalBlock("Review both screenshots", []map[string]any{
   104				{
   105					"media_type": "image/png",
   106					"url":        "https://example.com/left.png",
   107				},
   108				{
   109					"media_type": "image/jpeg",
   110					"content":    base64.StdEncoding.EncodeToString([]byte("RIGHT")),
   111				},
   112			}),
   113		}}
   114	
   115		got := buildInputItemsFromTurn(turn)
   116		if len(got) != 1 {
   117			t.Fatalf("expected 1 item, got %d", len(got))
   118		}
   119		if len(got[0].Content) != 3 {
   120			t.Fatalf("expected text + two image parts, got %#v", got[0].Content)
   121		}
   122		if got[0].Content[1].Type != "input_image" || got[0].Content[1].ImageURL != "https://example.com/left.png" {
   123			t.Fatalf("unexpected first image part: %#v", got[0].Content[1])
   124		}
   125		expectedRight := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte("RIGHT"))
   126		if got[0].Content[2].Type != "input_image" || got[0].Content[2].ImageURL != expectedRight {
   127			t.Fatalf("unexpected second image part: %#v", got[0].Content[2])
   128		}
   129		if got[0].Content[2].Detail != "auto" {
   130			t.Fatalf("expected second image detail to default to auto, got %#v", got[0].Content[2])
   131		}
   132	}
```

## geppetto/pkg/steps/ai/openai/helpers.go:232-270
```
   232					case turns.BlockKindOther:
   233						role = "assistant"
   234					case turns.BlockKindReasoning:
   235						role = "assistant"
   236					}
   237					// Check for images array in payload to construct MultiContent
   238					var msg ChatCompletionMessage
   239					if imgs, ok := b.Payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
   240						parts := []ChatMessagePart{{Type: chatMessagePartTypeText, Text: text}}
   241						for _, img := range imgs {
   242							mediaType, _ := img["media_type"].(string)
   243							url, _ := img["url"].(string)
   244							// content can be []byte or base64 string
   245							var base64Content string
   246							if raw, ok := img["content"]; ok && raw != nil {
   247								switch rv := raw.(type) {
   248								case []byte:
   249									base64Content = base64.StdEncoding.EncodeToString(rv)
   250								case string:
   251									// assume already base64
   252									base64Content = rv
   253								}
   254							}
   255							imageURL := url
   256							if imageURL == "" && base64Content != "" {
   257								imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, base64Content)
   258							}
   259							parts = append(parts, ChatMessagePart{
   260								Type: chatMessagePartTypeImageURL,
   261								ImageURL: &ChatMessageImageURL{
   262									URL:    imageURL,
   263									Detail: chatImageURLDetailAuto,
   264								},
   265							})
   266						}
   267						msg = ChatCompletionMessage{Role: role, MultiContent: parts}
   268					} else {
   269						msg = ChatCompletionMessage{Role: role, Content: text}
   270					}
```

## geppetto/pkg/steps/ai/claude/helpers.go:226-252
```
   226							text = s
   227						} else if bb, err := json.Marshal(v); err == nil {
   228							text = string(bb)
   229						}
   230					}
   231					parts := []api.Content{}
   232					if text != "" {
   233						parts = append(parts, api.NewTextContent(text))
   234					}
   235					if imgs, ok := b.Payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
   236						for _, img := range imgs {
   237							mediaType, _ := img["media_type"].(string)
   238							if raw, ok := img["content"]; ok && raw != nil {
   239								var base64Content string
   240								switch rv := raw.(type) {
   241								case []byte:
   242									base64Content = base64.StdEncoding.EncodeToString(rv)
   243								case string:
   244									base64Content = rv
   245								}
   246								if base64Content != "" {
   247									parts = append(parts, api.NewImageContent(mediaType, base64Content))
   248								}
   249							}
   250						}
   251					}
   252					if len(parts) > 0 {
```

## geppetto/pkg/steps/ai/gemini/modern_adapter.go:235-253
```
   235			}
   236		}
   237		contents := make([]*moderngenai.Content, 0, len(t.Blocks))
   238		for _, b := range t.Blocks {
   239			content := &moderngenai.Content{}
   240			switch b.Kind {
   241			case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindOther:
   242				content.Role = string(moderngenai.RoleUser)
   243				if txt, ok := blockText(b); ok {
   244					content.Parts = append(content.Parts, moderngenai.NewPartFromText(txt))
   245				}
   246			case turns.BlockKindLLMText:
   247				content.Role = string(moderngenai.RoleModel)
   248				if txt, ok := blockText(b); ok {
   249					content.Parts = append(content.Parts, moderngenai.NewPartFromText(txt))
   250				}
   251			case turns.BlockKindReasoning:
   252				content.Role = string(moderngenai.RoleModel)
   253				part := &moderngenai.Part{Thought: true}
```

## llm-proxy/pkg/openaichat/types.go:60-132
```
    60		if len(req.Messages) == 0 {
    61			return nil, FieldError{Field: "messages", Message: "messages is required", Code: "missing_messages"}
    62		}
    63		for i := range req.Messages {
    64			if err := req.Messages[i].Validate(i); err != nil {
    65				return nil, err
    66			}
    67		}
    68		for i := range req.Tools {
    69			if err := req.Tools[i].Validate(i); err != nil {
    70				return nil, err
    71			}
    72		}
    73		return &req, nil
    74	}
    75	
    76	func (m ChatMessage) Validate(index int) error {
    77		role := strings.TrimSpace(m.Role)
    78		switch role {
    79		case "system", "developer", "user":
    80			if _, err := m.RequiredContentString(index); err != nil {
    81				return err
    82			}
    83		case "assistant":
    84			if len(m.Content) != 0 && string(m.Content) != "null" {
    85				if _, err := m.ContentString(); err != nil {
    86					return withField(err, fmt.Sprintf("messages[%d].content", index))
    87				}
    88			}
    89			for j := range m.ToolCalls {
    90				if err := m.ToolCalls[j].Validate(fmt.Sprintf("messages[%d].tool_calls[%d]", index, j)); err != nil {
    91					return err
    92				}
    93			}
    94			if (len(m.Content) == 0 || string(m.Content) == "null") && len(m.ToolCalls) == 0 {
    95				return FieldError{Field: fmt.Sprintf("messages[%d].content", index), Message: "assistant message must include content or tool_calls", Code: "missing_content"}
    96			}
    97		case "tool":
    98			if strings.TrimSpace(m.ToolCallID) == "" {
    99				return FieldError{Field: fmt.Sprintf("messages[%d].tool_call_id", index), Message: "tool_call_id is required for tool messages", Code: "missing_tool_call_id"}
   100			}
   101			if _, err := m.RequiredContentString(index); err != nil {
   102				return err
   103			}
   104		default:
   105			return FieldError{Field: fmt.Sprintf("messages[%d].role", index), Message: fmt.Sprintf("unsupported message role %q", m.Role), Code: "unsupported_role"}
   106		}
   107		return nil
   108	}
   109	
   110	func (m ChatMessage) RequiredContentString(index int) (string, error) {
   111		if len(m.Content) == 0 || string(m.Content) == "null" {
   112			return "", FieldError{Field: fmt.Sprintf("messages[%d].content", index), Message: "message content is required", Code: "missing_content"}
   113		}
   114		text, err := m.ContentString()
   115		if err != nil {
   116			return "", withField(err, fmt.Sprintf("messages[%d].content", index))
   117		}
   118		return text, nil
   119	}
   120	
   121	func (m ChatMessage) ContentString() (string, error) {
   122		var s string
   123		if err := json.Unmarshal(m.Content, &s); err == nil {
   124			return s, nil
   125		}
   126		var arr []json.RawMessage
   127		if err := json.Unmarshal(m.Content, &arr); err == nil {
   128			return "", FieldError{Field: "content", Message: "message content arrays are not supported in this prototype", Code: "unsupported_content_shape"}
   129		}
   130		return "", FieldError{Field: "content", Message: "message content must be a string", Code: "unsupported_content_shape"}
   131	}
   132	
```

## llm-proxy/pkg/openaichat/mapper.go:20-45
```
    20		if err := attachTools(t, req); err != nil {
    21			return nil, err
    22		}
    23		for i, msg := range req.Messages {
    24			if err := msg.Validate(i); err != nil {
    25				return nil, err
    26			}
    27			switch msg.Role {
    28			case "system", "developer":
    29				text, _ := msg.ContentString()
    30				turns.AppendBlock(t, turns.NewSystemTextBlock(text))
    31			case "user":
    32				text, _ := msg.ContentString()
    33				turns.AppendBlock(t, turns.NewUserTextBlock(text))
    34			case "assistant":
    35				if len(msg.Content) != 0 && string(msg.Content) != "null" {
    36					text, _ := msg.ContentString()
    37					if text != "" {
    38						turns.AppendBlock(t, turns.NewAssistantTextBlock(text))
    39					}
    40				}
    41				for _, tc := range msg.ToolCalls {
    42					turns.AppendBlock(t, turns.NewToolCallBlock(tc.ID, tc.Function.Name, tc.Function.Arguments))
    43				}
    44			case "tool":
    45				text, _ := msg.ContentString()
```

## /home/manuel/go/pkg/mod/google.golang.org/genai@v1.58.0/types.go:1328-1435
```
  1328	type Blob struct {
  1329		// Required. The raw bytes of the data.
  1330		Data []byte `json:"data,omitempty"`
  1331		// Optional. The display name of the blob. Used to provide a label or filename to distinguish
  1332		// blobs. This field is only returned in `PromptMessage` for prompt management. It is
  1333		// used in the Gemini calls only when server-side tools (`code_execution`, `google_search`,
  1334		// and `url_context`) are enabled. This field is not supported in Gemini API.
  1335		DisplayName string `json:"displayName,omitempty"`
  1336		// Required. The IANA standard MIME type of the source data.
  1337		MIMEType string `json:"mimeType,omitempty"`
  1338	}
  1339	
  1340	// Provides metadata for a video, including the start and end offsets for clipping and
  1341	// the frame rate.
  1342	type VideoMetadata struct {
  1343		// Optional. The end offset of the video.
  1344		EndOffset time.Duration `json:"endOffset,omitempty"`
  1345		// Optional. The frame rate of the video sent to the model. If not specified, the default
  1346		// value is 1.0. The valid range is (0.0, 24.0].
  1347		FPS *float64 `json:"fps,omitempty"`
  1348		// Optional. The start offset of the video.
  1349		StartOffset time.Duration `json:"startOffset,omitempty"`
  1350	}
  1351	
  1352	func (c *VideoMetadata) UnmarshalJSON(data []byte) error {
  1353		type Alias VideoMetadata
  1354		aux := &struct {
  1355			EndOffset   string `json:"endOffset,omitempty"`
  1356			StartOffset string `json:"startOffset,omitempty"`
  1357			*Alias
  1358		}{
  1359			Alias: (*Alias)(c),
  1360		}
  1361	
  1362		if err := json.Unmarshal(data, &aux); err != nil {
  1363			return err
  1364		}
  1365	
  1366		if aux.EndOffset != "" {
  1367			d, err := time.ParseDuration(aux.EndOffset)
  1368			if err != nil {
  1369				return err
  1370			}
  1371			c.EndOffset = d
  1372		}
  1373	
  1374		if aux.StartOffset != "" {
  1375			d, err := time.ParseDuration(aux.StartOffset)
  1376			if err != nil {
  1377				return err
  1378			}
  1379			c.StartOffset = d
  1380		}
  1381	
  1382		return nil
  1383	}
  1384	
  1385	func (c *VideoMetadata) MarshalJSON() ([]byte, error) {
  1386		type Alias VideoMetadata
  1387		aux := &struct {
  1388			EndOffset   string `json:"endOffset,omitempty"`
  1389			StartOffset string `json:"startOffset,omitempty"`
  1390			*Alias
  1391		}{
  1392			Alias: (*Alias)(c),
  1393		}
  1394	
  1395		if c.StartOffset != 0 {
  1396			aux.StartOffset = fmt.Sprintf("%.0fs", c.StartOffset.Seconds())
  1397		}
  1398		if c.EndOffset != 0 {
  1399			aux.EndOffset = fmt.Sprintf("%.0fs", c.EndOffset.Seconds())
  1400			if aux.StartOffset == "" {
  1401				aux.StartOffset = "0s"
  1402			}
  1403		}
  1404	
  1405		return json.Marshal(aux)
  1406	}
  1407	
  1408	// A datatype containing media content.
  1409	// Exactly one field within a Part should be set, representing the specific type
  1410	// of content being conveyed. Using multiple fields within the same `Part`
  1411	// instance is considered invalid.
  1412	type Part struct {
  1413		// Optional. Media resolution for the input media.
  1414		MediaResolution *PartMediaResolution `json:"mediaResolution,omitempty"`
  1415		// Optional. The result of executing the ExecutableCode.
  1416		CodeExecutionResult *CodeExecutionResult `json:"codeExecutionResult,omitempty"`
  1417		// Optional. Code generated by the model that is intended to be executed.
  1418		ExecutableCode *ExecutableCode `json:"executableCode,omitempty"`
  1419		// Optional. The URI-based data of the part. This can be used to include files from
  1420		// Google Cloud Storage.
  1421		FileData *FileData `json:"fileData,omitempty"`
  1422		// Optional. A predicted [FunctionCall] returned from the model that contains a string
  1423		// representing the [FunctionDeclaration.Name] with the parameters and their values.
  1424		FunctionCall *FunctionCall `json:"functionCall,omitempty"`
  1425		// Optional. The result output of a [FunctionCall] that contains a string representing
  1426		// the [FunctionDeclaration.Name] and a structured JSON object containing any output
  1427		// from the function call. It is used as context to the model.
  1428		FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
  1429		// Optional. The inline data content of the part. This can be used to include images,
  1430		// audio, or video in a request.
  1431		InlineData *Blob `json:"inlineData,omitempty"`
  1432		// Optional. The text content of the part. When sent from the VSCode Gemini Code Assist
  1433		// extension, references to @mentioned items will be converted to markdown boldface
  1434		// text. For example `@my-repo` will be converted to and sent as `**my-repo**` by the
  1435		// IDE agent.
```

