---
Title: Image Attachment Flow Analysis
Ticket: 001-ADD-OCR-VERB
Status: active
Topics:
    - backend
    - cli
    - llm-providers
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Seed Turn builder uses NewUserMultimodalBlock when images present
    - Path: pinocchio/pkg/cmds/images.go
      Note: Path->payload conversion helper
    - Path: pinocchio/pkg/cmds/run/context.go
      Note: RunContext now carries image paths
ExternalSources: []
Summary: 'Analysis of image attachment flow from CLI to LLM providers. Identified bug: images parsed but not passed to turn builder.'
LastUpdated: 2025-12-24T14:16:22.846393942-05:00
WhatFor: ""
WhenToUse: ""
---


# Image Attachment Flow Analysis

## Executive Summary

**Issue**: Images passed via `--images` flag are parsed correctly but never reach LLM providers. The LLM responds with "I'm unable to view images" when images are provided.

**Root Cause**: Image paths are extracted from CLI settings in `pinocchio/pkg/cmds/cmd.go` but never passed to the turn builder. The `buildInitialTurnFromBlocks` function uses `NewUserTextBlock` (which doesn't support images) instead of `NewUserMultimodalBlock`.

**Status**: Analysis complete. Fix required: pass image paths through call chain and convert to turn payload format.

## Current Flow (Broken)

### 1. CLI Parsing ✅
- **Location**: `pinocchio/pkg/cmds/cmdlayers/helpers.go:80-82`
- **Status**: Working correctly
- **Implementation**: `--images` flag defined as `ParameterTypeFileList`
- **Result**: `HelpersSettings.Images []*parameters.FileData`

### 2. Image Path Extraction ✅
- **Location**: `pinocchio/pkg/cmds/cmd.go:207-211`
- **Status**: Working correctly
- **Implementation**: Extracts paths from `helpersSettings.Images`
- **Result**: `imagePaths []string` created
- **Problem**: `imagePaths` never used after extraction

### 3. Turn Building ❌
- **Location**: `pinocchio/pkg/cmds/cmd.go:57-68` (`buildInitialTurnFromBlocks`)
- **Status**: **BUG** — images not included
- **Implementation**: Uses `NewUserTextBlock(userPrompt)` which doesn't support images
- **Result**: Turn created without image blocks

### 4. Provider Handling ✅
- **Location**: 
  - `geppetto/pkg/steps/ai/claude/helpers.go:101-118`
  - `geppetto/pkg/steps/ai/openai/helpers.go:186-213`
- **Status**: Working correctly (but never receives images)
- **Implementation**: Reads `PayloadKeyImages` from block payload
- **Expected Format**: `[]map[string]any{"media_type": string, "content": []byte|string}`

## Required Fix

### Data Flow (Fixed)

```
CLI: --images file.png
  ↓
HelpersSettings.Images []*parameters.FileData
  ↓
imagePaths []string (extracted)
  ↓ [MISSING: pass through call chain]
buildInitialTurn(vars) 
  ↓ [MISSING: accept imagePaths]
buildInitialTurnFromBlocksRendered(sp, blocks, up, vars)
  ↓ [MISSING: accept imagePaths]
buildInitialTurnFromBlocks(sp, blocks, up)
  ↓ [MISSING: accept imagePaths, convert to payload format]
NewUserMultimodalBlock(text, images []map[string]any)
  ↓
Turn with PayloadKeyImages
  ↓
Provider handlers (Claude/OpenAI)
  ↓
LLM receives images ✅
```

### Implementation Steps

1. **Create helper function** to convert image paths → turn payload format:
   ```go
   func imagePathsToTurnPayload(imagePaths []string) ([]map[string]any, error)
   ```
   - Load images using `conversation.NewImageContentFromFile` (or create turn-specific loader)
   - Convert `ImageContent` → `map[string]any{"media_type": string, "content": []byte}`

2. **Modify function signatures** to accept image paths:
   - `buildInitialTurn(vars, imagePaths []string)`
   - `buildInitialTurnFromBlocksRendered(sp, blocks, up, vars, imagePaths []string)`
   - `buildInitialTurnFromBlocks(sp, blocks, up, imagePaths []string)`

3. **Update turn building logic**:
   - If `len(imagePaths) > 0`: use `NewUserMultimodalBlock(text, images)`
   - Otherwise: use `NewUserTextBlock(text)` (backward compatible)

4. **Pass image paths** from `RunIntoWriter` through call chain

## Code References

### Image Parsing
```80:82:pinocchio/pkg/cmds/cmdlayers/helpers.go
			parameters.NewParameterDefinition(
				"images",
				parameters.ParameterTypeFileList,
				parameters.WithHelp("Images to display"),
```

### Image Extraction (Unused)
```207:211:pinocchio/pkg/cmds/cmd.go
	// Create image paths from helper settings
	imagePaths := make([]string, len(helpersSettings.Images))
	for i, img := range helpersSettings.Images {
		imagePaths[i] = img.Path
	}
```

### Turn Building (Needs Fix)
```57:68:pinocchio/pkg/cmds/cmd.go
func buildInitialTurnFromBlocks(systemPrompt string, blocks []turns.Block, userPrompt string) *turns.Turn {
	t := &turns.Turn{}
	if strings.TrimSpace(systemPrompt) != "" {
		turns.AppendBlock(t, turns.NewSystemTextBlock(systemPrompt))
	}
	if len(blocks) > 0 {
		turns.AppendBlocks(t, blocks...)
	}
	if strings.TrimSpace(userPrompt) != "" {
		turns.AppendBlock(t, turns.NewUserTextBlock(userPrompt))
	}
	return t
}
```

### Available Multimodal Constructor
```24:37:geppetto/pkg/turns/helpers_blocks.go
// NewUserMultimodalBlock creates a user block with text and optional images.
// images is a slice of maps with keys: "media_type" (string), and either "url" (string) or "content" ([]byte/base64).
func NewUserMultimodalBlock(text string, images []map[string]any) Block {
	payload := map[string]any{PayloadKeyText: text}
	if len(images) > 0 {
		payload[PayloadKeyImages] = images
	}
	return Block{
		ID:      uuid.NewString(),
		Kind:    BlockKindUser,
		Role:    RoleUser,
		Payload: payload,
	}
}
```

### Provider Image Handling (Claude)
```101:118:geppetto/pkg/steps/ai/claude/helpers.go
				// optional images from payload
				if imgs, ok := b.Payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
					for _, img := range imgs {
						mediaType, _ := img["media_type"].(string)
						if raw, ok := img["content"]; ok && raw != nil {
							var base64Content string
							switch rv := raw.(type) {
							case []byte:
								base64Content = base64.StdEncoding.EncodeToString(rv)
							case string:
								base64Content = rv
							}
							if base64Content != "" {
								parts = append(parts, api.NewImageContent(mediaType, base64Content))
							}
						}
					}
				}
```

### Image Loading Function (Reusable)
```154:187:geppetto/pkg/conversation/message.go
func newImageContentFromLocalFile(path string) (*ImageContent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %v", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	if fileInfo.Size() > 20*1024*1024 {
		return nil, fmt.Errorf("image size exceeds 20MB limit")
	}

	mediaType := getMediaTypeFromExtension(filepath.Ext(path))
	if mediaType == "" {
		return nil, fmt.Errorf("unsupported image format: %s", filepath.Ext(path))
	}

	return &ImageContent{
		ImageContent: content,
		ImageName:    fileInfo.Name(),
		MediaType:    mediaType,
		Detail:       ImageDetailAuto,
	}, nil
}
```

## Format Specifications

### Turn Payload Format
```go
[]map[string]any{
	{
		"media_type": "image/png",  // string, required
		"content": []byte{...},      // []byte or string (base64), required for local files
		"url": "https://...",        // string, optional (for URLs)
	}
}
```

### Provider Expectations
- **Claude**: Accepts `content` as `[]byte` or `string` (base64), auto-converts
- **OpenAI**: Accepts `content` as `[]byte` or `string` (base64), creates data URL

## Testing

### Test Command
```bash
go run ./cmd/pinocchio code professional --images ~/Downloads/image\ \(6\).png "Describe this image"
```

### Expected Behavior (After Fix)
- LLM receives and processes images
- Response includes image analysis/description
- Works with both Claude and OpenAI providers

### Current Behavior
- Command executes successfully
- LLM responds: "I'm unable to view images"
- Images not included in turn blocks

## Related Files

- `pinocchio/pkg/cmds/cmdlayers/helpers.go` — CLI parameter definition
- `pinocchio/pkg/cmds/cmd.go` — Image extraction and turn building
- `geppetto/pkg/turns/helpers_blocks.go` — Block constructors
- `geppetto/pkg/steps/ai/claude/helpers.go` — Claude provider handler
- `geppetto/pkg/steps/ai/openai/helpers.go` — OpenAI provider handler
- `geppetto/pkg/conversation/message.go` — Image loading utilities

## Next Steps

1. ✅ Analysis complete
2. ⏳ Implement fix: pass image paths through call chain
3. ⏳ Create helper function for image path → turn payload conversion
4. ⏳ Update turn building to use `NewUserMultimodalBlock` when images present
5. ⏳ Test with both Claude and OpenAI providers
6. ⏳ Verify backward compatibility (commands without images still work)
