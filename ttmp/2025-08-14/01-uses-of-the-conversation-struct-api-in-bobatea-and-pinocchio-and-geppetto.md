# Uses of the Conversation Struct/API in Bobatea, Pinocchio, and Geppetto

This document catalogs all uses of the conversation abstraction from `github.com/go-go-golems/geppetto/pkg/conversation` across the codebase, focusing on the `@bobatea/` and `@pinocchio/` directories as requested, but also including relevant usage in `@geppetto/` for completeness.

## Overview

The conversation abstraction provides:
- `Message` struct: Represents individual chat messages with content, metadata, and relationships
- `Conversation` type: A slice of `*Message` representing a conversation thread
- Content types: `ChatMessageContent`, `ToolUseContent`, `ToolResultContent`, `ImageContent`, `ErrorContent`
- Roles: `RoleSystem`, `RoleUser`, `RoleAssistant`, `RoleTool`
- Metadata: `LLMMessageMetadata` with usage statistics, engine info, etc.
- Utility functions: `NewNodeID()`, `NewChatMessage()`, etc.

## Usage by Directory

### @bobatea/ Directory

#### bobatea/pkg/chat/model.go
- **Import**: `geppetto_conversation "github.com/go-go-golems/geppetto/pkg/conversation"`
- **Usage**: 
  - Line 915: `id := geppetto_conversation.NewNodeID().String()` - Used for generating unique message IDs for timeline entities
  - **Status**: ✅ **Safe to remove** - This is only using `NewNodeID()` which can be replaced with a local ID generator

#### bobatea/cmd/chat/fake-backend.go
- **Import**: `conversation2 "github.com/go-go-golems/geppetto/pkg/conversation"`
- **Usage**:
  - Line 64: `localID := conversation2.NewNodeID().String()` - ID generation
  - Line 66-67: `conversation2.LLMMessageMetadata{...}` and `conversation2.Usage{...}` - Used for fake event metadata
  - **Status**: ✅ **Safe to remove** - Only used for demo/testing purposes

#### bobatea/pkg/timeline/renderers/llm_text_model.go
- **Usage**: References to `LLMMessageMetadata` in type assertions and field extraction
- **Status**: ⚠️ **Partial dependency** - Uses the struct shape but could be refactored to use local types

### @pinocchio/ Directory

#### pinocchio/pkg/ui/backend.go
- **Import**: `conv "github.com/go-go-golems/geppetto/pkg/conversation"`
- **Usage**:
  - Line 15: Import statement
  - Line 102: `func (e *EngineBackend) SetSeedFromConversation(c conv.Conversation)` - Accepts conversation as input
  - Line 105: `turns.AppendBlocks(t, turns.BlocksFromConversationDelta(c, 0)...)` - Converts conversation to turns
  - **Status**: 🔴 **Active dependency** - This is a key boundary interface

#### pinocchio/pkg/cmds/run/context.go
- **Import**: `geppetto_conversation "github.com/go-go-golems/geppetto/pkg/conversation"`
- **Usage**:
  - Line 44: `ResultConversation []*geppetto_conversation.Message` - Stores conversation results
  - **Status**: 🔴 **Active dependency** - Used for storing final results

#### pinocchio/pkg/chatrunner/chat_runner.go
- **Import**: `geppetto_conversation "github.com/go-go-golems/geppetto/pkg/conversation"`
- **Usage**:
  - Line 41: `manager geppetto_conversation.Manager` - Holds conversation manager instance
  - Line 125: `backend.SetSeedFromConversation(cs.manager.GetConversation())` - Gets conversation from manager
  - Line 179: `conversation_ := cs.manager.GetConversation()` - Retrieves current conversation
  - Line 181: `turns.AppendBlocks(seed, turns.BlocksFromConversationDelta(conversation_, 0)...)` - Converts conversation to blocks
  - Line 195: `conv := turns.BuildConversationFromTurn(updatedTurn)` - Converts turns back to conversation
  - Line 199: `cs.manager.AppendMessages(newMessages...)` - Appends new messages to conversation
  - **Status**: 🔴 **Core dependency** - The entire chat system is built around conversation management

#### pinocchio/cmd/agents/simple-chat-agent/main.go
- **Import**: `"github.com/go-go-golems/geppetto/pkg/conversation/builder"`
- **Usage**: 
  - Line 12: Import of conversation/builder (indirect usage)
  - **Status**: 🟡 **Indirect usage** - May be removable if not actively used

### @geppetto/ Directory (for reference)

#### geppetto/pkg/turns/conv_conversation.go
- **Import**: `"github.com/go-go-golems/geppetto/pkg/conversation"`
- **Usage**:
  - Line 11: `func BuildConversationFromTurn(t *Turn) conversation.Conversation` - Core conversion function
  - Line 15: `msgs := make(conversation.Conversation, 0, len(t.Blocks))`
  - Lines 20, 23, 27: `conversation.NewChatMessage(...)` - Creates conversation messages
  - Lines 34-40: `conversation.ToolUseContent{...}` - Tool use content creation
  - Lines 45-49: `conversation.ToolResultContent{...}` - Tool result content creation
  - Line 62: `func BlocksFromConversationDelta(updated conversation.Conversation, startIdx int) []Block` - Reverse conversion
  - Lines 70-95: Extensive usage of conversation types for conversion
  - **Status**: 🔴 **Critical dependency** - This is the primary conversion layer between turns and conversation

#### geppetto/pkg/steps/ai/claude/
- **Usage**: Multiple files use conversation types for Claude API integration
- **Status**: 🔴 **Active usage** - Core inference engine dependency

#### geppetto/pkg/steps/ai/openai/
- **Usage**: Multiple files use conversation types for OpenAI API integration
- **Status**: 🔴 **Active usage** - Core inference engine dependency

#### geppetto/pkg/inference/toolhelpers/helpers.go
- **Usage**: Tool call extraction and processing using conversation types
- **Status**: 🔴 **Active usage** - Tool integration layer

## Summary of Dependencies

### Critical Dependencies (Cannot be easily removed)
1. **pinocchio/pkg/chatrunner/chat_runner.go** - Core chat system architecture
2. **pinocchio/pkg/cmds/run/context.go** - Run context storage
3. **geppetto/pkg/turns/conv_conversation.go** - Conversion layer between turns and conversation
4. **geppetto/pkg/steps/ai/** - Inference engine implementations

### Boundary Dependencies (Used at system boundaries)
1. **pinocchio/pkg/ui/backend.go** - UI/backend interface
2. **pinocchio/pkg/cmds/run/context.go** - Result storage

### Safe to Remove
1. **bobatea/pkg/chat/model.go** - Only uses NewNodeID()
2. **bobatea/cmd/chat/fake-backend.go** - Demo/testing code
3. **Various documentation files** - Only contain examples

## Migration Strategy

### Phase 1: Remove Safe Dependencies
- Replace `conversation.NewNodeID()` with local UUID generation
- Remove fake backend dependencies
- Update documentation

### Phase 2: Refactor Boundary Interfaces
- Replace `conv.Conversation` parameters with `[]*turns.Block` or `*turns.Turn`
- Update `SetSeedFromConversation` to `SetSeedFromBlocks` or similar
- Refactor result storage to use turns-based structures

### Phase 3: Core Architecture Changes
- Replace conversation manager with turns-based history management
- Update chat runner to work directly with turns instead of conversation
- Refactor inference engines to use turns as primary data structure

### Phase 4: Complete Removal
- Remove all conversation package imports
- Delete conversation package entirely
- Update all dependent systems

## Files Requiring Updates

### High Priority (Core System)
- `pinocchio/pkg/chatrunner/chat_runner.go`
- `pinocchio/pkg/cmds/run/context.go`
- `pinocchio/pkg/ui/backend.go`
- `geppetto/pkg/turns/conv_conversation.go`

### Medium Priority (Inference Engines)
- `geppetto/pkg/steps/ai/claude/*.go`
- `geppetto/pkg/steps/ai/openai/*.go`
- `geppetto/pkg/inference/toolhelpers/*.go`

### Low Priority (Safe to Remove)
- `bobatea/pkg/chat/model.go`
- `bobatea/cmd/chat/fake-backend.go`
- Documentation files
- Example files

## Recommendations

1. **Start with turns-based architecture**: The `turns` package already provides the necessary abstractions
2. **Maintain backward compatibility**: Provide adapter functions during transition
3. **Incremental migration**: Update one system at a time to minimize disruption
4. **Test thoroughly**: Ensure all chat modes (blocking, interactive, chat) work correctly
5. **Update documentation**: Reflect the new architecture in all documentation

The conversation abstraction is deeply integrated into the core architecture, particularly in Pinocchio's chat system. A complete removal will require significant refactoring of the chat runner, UI backend, and inference engines to work directly with the turns-based system instead.