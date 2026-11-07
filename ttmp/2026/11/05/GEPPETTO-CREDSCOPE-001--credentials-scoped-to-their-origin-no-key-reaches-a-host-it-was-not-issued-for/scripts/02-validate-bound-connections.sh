#!/usr/bin/env bash
# Run from the Geppetto repository root. Recording transports make these tests
# independent of socket access and of real credentials or API availability.
set -euo pipefail
export GOWORK=off
export GOCACHE=/tmp/geppetto-bound-go-cache
export GOFLAGS=-buildvcs=false
export GOPROXY=off

go test -count=1 ./pkg/steps/ai/connection ./pkg/steps/ai/credscope ./pkg/steps/ai/providers/geminisdk ./pkg/steps/ai/providers/umans

go test -count=1 ./pkg/steps/ai/credentials/oauth -run 'Test(TokenGrants|AuthorizationURL|NewPKCE)'

go test -count=1 ./pkg/embeddings -run 'Test(EmbeddingCredentialPairs|NewSettingsFactoryFromInferenceSettings|OpenAIProviderNewRequest|SupportsOpenAI|CohereProvider_GetModel|ProviderBatch)'

go test -count=1 ./pkg/steps/ai/openai ./pkg/steps/ai/openai_responses ./pkg/steps/ai/claude/api ./pkg/steps/ai/providers/pi ./pkg/steps/ai/providers/openaicodex -run 'Test(OpenChatCompletionStreamRetries|OpenChatCompletionStreamRedacts|OpenChatCompletionStreamDoesNotRetry|ResolveBearerToken|ResolveChatStreamConfig|OpenResponses|ResponsesCredential|ResponsesSlot|ResponsesAPI|ResponsesBase|ResponsesOutbound|ResponsesRunInference|ResolveFirstPartyDefaults|ResolvePinsClientAcrossRedirects|ResolvedSettingsCannot|ClientSendsBound|ResponsesCore|Whisper)'

go test -count=1 ./pkg/inference/engine/factory ./pkg/inference/tokencount/factory -run 'Test(StandardEngineFactory_(Supported|Default|CreateEngine|WithOpenAI)|StandardEngineFactoryBearer|EngineFactoryWith|CounterFactoryWith|NewFromSettings)'

go test -count=1 ./pkg/engineprofiles ./pkg/cli/bootstrap ./pkg/doc

go test -race -count=1 ./pkg/steps/ai/connection ./pkg/steps/ai/credscope ./pkg/steps/ai/providers/geminisdk ./pkg/steps/ai/providers/umans

go test ./... -run '^$'
go build ./...
