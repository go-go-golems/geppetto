# Tasks

## Implemented (2025-12-24)

- [x] **Thread images through run context**: added `ImagePaths` + `WithImagePaths` in `pinocchio/pkg/cmds/run/context.go`
- [x] **Convert image paths to provider payload**: implemented `imagePathsToTurnImages` in `pinocchio/pkg/cmds/images.go`
- [x] **Attach images to initial user block**: updated seed turn building in `pinocchio/pkg/cmds/cmd.go` to use `turns.NewUserMultimodalBlock`
- [x] **Wire CLI `--images` end-to-end**: `RunIntoWriter` passes images via `run.WithImagePaths(...)`
- [x] **Unit tests**: added tests in `pinocchio/pkg/cmds/images_test.go`
- [x] **Build & test**:
  - [x] `pinocchio`: `make build && make test`
  - [x] `geppetto`: `make build && make test`
- [x] **Manual verification**: `go run ./cmd/pinocchio code professional --images … \"Describe this image\"` now produces an actual image description

## Superseded draft (kept for history)

### Task 1: Create helper function to convert image paths to turn payload format
- [ ] **File**: `geppetto/pkg/turns/helpers_blocks.go`
- [ ] **Function**: `ImagePathsToTurnPayload(imagePaths []string) ([]map[string]any, error)`
- [ ] **Purpose**: Convert file paths to turn payload format expected by `NewUserMultimodalBlock`
- [ ] **Implementation**:
  - Import `geppetto/pkg/conversation` package to reuse `NewImageContentFromFile`
  - For each path, load image using `conversation.NewImageContentFromFile(path)`
  - Convert `ImageContent` to `map[string]any{"media_type": string, "content": []byte}`
  - Handle errors (file not found, unsupported format, size limits)
  - Return `[]map[string]any` ready for `NewUserMultimodalBlock`
- [ ] **Tests**: Add unit tests for the helper function
  - Test with valid image files (PNG, JPEG)
  - Test with invalid paths
  - Test with unsupported formats
  - Test with files exceeding size limit

### Task 2: Modify buildInitialTurnFromBlocks to accept image paths
- [ ] **File**: `pinocchio/pkg/cmds/cmd.go`
- [ ] **Function**: `buildInitialTurnFromBlocks(systemPrompt string, blocks []turns.Block, userPrompt string, imagePaths []string) *turns.Turn`
- [ ] **Changes**:
  - Add `imagePaths []string` parameter
  - If `len(imagePaths) > 0`:
    - Call `turns.ImagePathsToTurnPayload(imagePaths)` to convert paths to payload format
    - Use `turns.NewUserMultimodalBlock(userPrompt, images)` instead of `turns.NewUserTextBlock(userPrompt)`
  - If `len(imagePaths) == 0`: use `turns.NewUserTextBlock(userPrompt)` (backward compatible)
- [ ] **Error handling**: Handle errors from `ImagePathsToTurnPayload` appropriately

### Task 3: Update buildInitialTurnFromBlocksRendered to pass image paths
- [ ] **File**: `pinocchio/pkg/cmds/cmd.go`
- [ ] **Function**: `buildInitialTurnFromBlocksRendered(systemPrompt string, blocks []turns.Block, userPrompt string, vars map[string]interface{}, imagePaths []string) (*turns.Turn, error)`
- [ ] **Changes**:
  - Add `imagePaths []string` parameter
  - Pass `imagePaths` to `buildInitialTurnFromBlocks` call
- [ ] **Note**: This function already returns error, so error handling is in place

### Task 4: Update buildInitialTurn to accept and pass image paths
- [ ] **File**: `pinocchio/pkg/cmds/cmd.go`
- [ ] **Function**: `(g *PinocchioCommand) buildInitialTurn(vars map[string]interface{}, imagePaths []string) (*turns.Turn, error)`
- [ ] **Changes**:
  - Add `imagePaths []string` parameter
  - Pass `imagePaths` to `buildInitialTurnFromBlocksRendered` call
- [ ] **Note**: Change return type from `*turns.Turn` to `(*turns.Turn, error)` to propagate errors

### Task 5: Update RunIntoWriter to pass image paths to buildInitialTurn
- [ ] **File**: `pinocchio/pkg/cmds/cmd.go`
- [ ] **Function**: `(g *PinocchioCommand) RunIntoWriter(...)`
- [ ] **Location**: Lines 207-211 (where `imagePaths` are extracted)
- [ ] **Changes**:
  - After extracting `imagePaths` from `helpersSettings.Images`, pass them to `buildInitialTurn`
  - Update all calls to `buildInitialTurn` to include `imagePaths` parameter
  - Handle errors returned from `buildInitialTurn`
- [ ] **Locations to update**:
  - Line 242: `buildInitialTurn` call in `PrintPrompt` branch
  - Line 375: `buildInitialTurn` call in `runEngineAndCollectMessages`
  - Line 521: `buildInitialTurnFromBlocksRendered` call in `runChat` (for seed turn)

### Task 6: Update RunWithOptions to pass image paths through run context
- [ ] **File**: `pinocchio/pkg/cmds/cmd.go`
- [ ] **Function**: `(g *PinocchioCommand) RunWithOptions(...)`
- [ ] **Consideration**: May need to add `ImagePaths []string` to `run.RunContext` struct
- [ ] **Alternative**: Pass image paths directly to `runBlocking` and `runChat` functions
- [ ] **Decision needed**: Should image paths be part of `RunContext` or passed as separate parameter?

### Task 7: Update runBlocking and runChat to handle image paths
- [ ] **File**: `pinocchio/pkg/cmds/cmd.go`
- [ ] **Functions**: 
  - `runBlocking(ctx context.Context, rc *run.RunContext) (*turns.Turn, error)`
  - `runChat(ctx context.Context, rc *run.RunContext) (*turns.Turn, error)`
- [ ] **Changes**:
  - Ensure image paths are available when calling `buildInitialTurn`
  - If using `RunContext`, extract from there
  - If passing directly, add parameter to these functions

### Task 8: Test the fix with Claude provider
- [ ] **Test command**: `go run ./cmd/pinocchio code professional --images ~/Downloads/image\ \(6\).png "Describe this image"`
- [ ] **Expected**: LLM should receive and process the image
- [ ] **Verify**: Response includes image analysis/description
- [ ] **Provider**: Set to use Claude (check provider settings)

### Task 9: Test the fix with OpenAI provider
- [ ] **Test command**: `go run ./cmd/pinocchio code professional --images ~/Downloads/image\ \(6\).png "Describe this image"`
- [ ] **Expected**: LLM should receive and process the image
- [ ] **Verify**: Response includes image analysis/description
- [ ] **Provider**: Set to use OpenAI (check provider settings)

### Task 10: Verify backward compatibility
- [ ] **Test command**: `go run ./cmd/pinocchio code professional "Describe something"` (no --images flag)
- [ ] **Expected**: Command works exactly as before
- [ ] **Verify**: No errors, normal text-only interaction

### Task 11: Test with multiple images
- [ ] **Test command**: `go run ./cmd/pinocchio code professional --images image1.png --images image2.png "Compare these images"`
- [ ] **Expected**: Both images are passed to LLM
- [ ] **Verify**: Response references both images

### Task 12: Test error handling
- [ ] **Test cases**:
  - Invalid file path: `--images /nonexistent/file.png`
  - Unsupported format: `--images file.txt`
  - File too large: `--images huge-file.png` (if > 20MB)
- [ ] **Expected**: Appropriate error messages, command fails gracefully

### Task 13: Update documentation
- [ ] **File**: Update relevant README or documentation files
- [ ] **Content**: Document `--images` flag usage and supported formats
- [ ] **Note**: Flag already exists, just needs to be documented as working

## Implementation Order

1. **Task 1** - Create helper function (foundation)
2. **Task 2** - Modify `buildInitialTurnFromBlocks` (core logic)
3. **Task 3** - Update `buildInitialTurnFromBlocksRendered` (call chain)
4. **Task 4** - Update `buildInitialTurn` (call chain)
5. **Task 5** - Update `RunIntoWriter` (entry point)
6. **Task 6-7** - Update run functions if needed (may not be necessary if passing directly)
7. **Task 8-12** - Testing
8. **Task 13** - Documentation

## Notes

- **Dependency consideration**: Task 1 imports `geppetto/pkg/conversation` into `geppetto/pkg/turns`. This is acceptable as both are in geppetto package.
- **Error propagation**: Ensure errors from image loading are properly propagated and user-friendly
- **Backward compatibility**: Must maintain compatibility with commands that don't use `--images` flag
- **Code review**: Pay special attention to error handling and edge cases
