package openai_responses

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
)

func (e *Engine) attachToolsToResponsesRequest(ctx context.Context, t *turns.Turn, reqBody *responsesRequest) error {
	if t == nil || reqBody == nil {
		return nil
	}

	engineTools := tools.AdvertisedToolDefinitionsFromContext(ctx)

	var toolCfg engine.ToolConfig
	if cfg, ok, err := engine.KeyToolConfig.Get(t.Data); err != nil {
		return errors.Wrap(err, "get tool config")
	} else if ok {
		toolCfg = cfg
	}

	if len(engineTools) > 0 && toolCfg.Enabled {
		converted, err := e.PrepareToolsForResponses(engineTools, toolCfg)
		if err != nil {
			return err
		}
		if arr, ok := converted.([]any); ok && len(arr) > 0 {
			reqBody.Tools = arr
			// Responses API: omit tool_choice for function tools to allow model selection
			reqBody.ToolChoice = nil
			if toolCfg.MaxParallelTools > 1 {
				b := true
				reqBody.ParallelToolCalls = &b
			} else if toolCfg.MaxParallelTools == 1 {
				b := false
				reqBody.ParallelToolCalls = &b
			}
			log.Debug().Int("tool_count", len(arr)).Interface("tool_choice", reqBody.ToolChoice).Msg("Responses: tools attached to request")
		}
	}

	if builtins, ok, err := turns.KeyResponsesServerTools.Get(t.Data); err != nil {
		return errors.Wrap(err, "get responses server tools")
	} else if ok && len(builtins) > 0 {
		reqBody.Tools = append(reqBody.Tools, builtins...)
		log.Debug().Int("builtin_tool_count", len(builtins)).Msg("Responses: server-side tools attached to request")
	}

	return nil
}
