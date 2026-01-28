package middleware

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"
)

func runSystemPromptMiddleware(t *testing.T, prompt string, seed *turns.Turn) *turns.Turn {
	t.Helper()

	handler := NewSystemPromptMiddleware(prompt)(func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
		return t, nil
	})

	res, err := handler(context.Background(), seed)
	require.NoError(t, err)
	require.NotNil(t, res)
	return res
}

func TestSystemPromptMiddlewareInsertsWhenMissing(t *testing.T) {
	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("hi"))

	res := runSystemPromptMiddleware(t, "system prompt", seed)

	require.Len(t, res.Blocks, 2)
	require.Equal(t, turns.BlockKindSystem, res.Blocks[0].Kind)
	require.Equal(t, "system prompt", res.Blocks[0].Payload[turns.PayloadKeyText])
	require.Equal(t, turns.BlockKindUser, res.Blocks[1].Kind)
}

func TestSystemPromptMiddlewareReplacesExistingSystem(t *testing.T) {
	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewSystemTextBlock("old prompt"))
	turns.AppendBlock(seed, turns.NewUserTextBlock("hi"))

	res := runSystemPromptMiddleware(t, "new prompt", seed)

	require.Len(t, res.Blocks, 2)
	require.Equal(t, turns.BlockKindSystem, res.Blocks[0].Kind)
	require.Equal(t, "new prompt", res.Blocks[0].Payload[turns.PayloadKeyText])
	require.Equal(t, turns.BlockKindUser, res.Blocks[1].Kind)
}

func TestSystemPromptMiddlewareIsIdempotent(t *testing.T) {
	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewSystemTextBlock("steady prompt"))
	turns.AppendBlock(seed, turns.NewUserTextBlock("hi"))

	res := runSystemPromptMiddleware(t, "steady prompt", seed)
	res = runSystemPromptMiddleware(t, "steady prompt", res)

	require.Len(t, res.Blocks, 2)
	require.Equal(t, turns.BlockKindSystem, res.Blocks[0].Kind)
	require.Equal(t, "steady prompt", res.Blocks[0].Payload[turns.PayloadKeyText])
}
