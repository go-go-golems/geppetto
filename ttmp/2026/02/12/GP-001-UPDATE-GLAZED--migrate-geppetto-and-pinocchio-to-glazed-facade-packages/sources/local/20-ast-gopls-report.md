# Glazed Migration Analyzer Report

- Generated: `2026-02-12T08:36:35-05:00`
- Repo root: `/home/manuel/workspaces/2026-02-11/geppetto-glazed-bump`
- Modules: `geppetto, pinocchio`
- gopls enrichment: `true` (max calls: `60`, timeout: `12s`)

## Summary

- Go files scanned: `209`
- Legacy import hits: `83`
- Legacy selector hits: `815`
- Legacy tag hits: `229`
- Signature hotspots: `57`
- Hotspots enriched with gopls: `57`

## Legacy Imports

- `github.com/go-go-golems/glazed/pkg/cmds/layers`
- `github.com/go-go-golems/glazed/pkg/cmds/middlewares`
- `github.com/go-go-golems/glazed/pkg/cmds/parameters`
- `github.com/go-go-golems/glazed/pkg/cmds/parsedlayers`

## Top files by legacy selector usage

| File | Count |
| --- | ---: |
| `pinocchio/cmd/pinocchio/cmds/openai/transcribe.go` | 96 |
| `geppetto/cmd/examples/generic-tool-calling/main.go` | 54 |
| `pinocchio/pkg/cmds/cmdlayers/helpers.go` | 53 |
| `pinocchio/cmd/pinocchio/cmds/catter/cmds/print.go` | 52 |
| `geppetto/cmd/llm-runner/main.go` | 51 |
| `geppetto/pkg/layers/layers.go` | 51 |
| `pinocchio/cmd/examples/simple-redis-streaming-inference/main.go` | 37 |
| `geppetto/cmd/examples/simple-streaming-inference/main.go` | 33 |
| `geppetto/cmd/examples/openai-tools/main.go` | 31 |
| `geppetto/cmd/examples/middleware-inference/main.go` | 26 |
| `pinocchio/cmd/pinocchio/cmds/catter/cmds/stats.go` | 25 |
| `pinocchio/cmd/pinocchio/cmds/helpers/md-extract.go` | 23 |
| `pinocchio/cmd/pinocchio/cmds/kagi/summarize.go` | 22 |
| `pinocchio/cmd/pinocchio/cmds/kagi/enrich.go` | 21 |
| `pinocchio/cmd/web-chat/main.go` | 18 |
| `geppetto/cmd/examples/simple-inference/main.go` | 17 |
| `pinocchio/cmd/pinocchio/cmds/openai/openai.go` | 17 |
| `pinocchio/pkg/cmds/helpers/parse-helpers.go` | 17 |
| `pinocchio/cmd/examples/simple-chat/main.go` | 15 |
| `pinocchio/pkg/redisstream/redis_layer.go` | 15 |

## Legacy Tag Keys

- `glazed.default`: `4`
- `glazed.help`: `4`
- `glazed.layer`: `7`
- `glazed.parameter`: `214`

## Signature Hotspots

| Function | Location | gopls refs | Notes |
| --- | --- | ---: | --- |
| `SimpleAgentCmd.RunIntoWriter` | `pinocchio/cmd/agents/simple-chat-agent/main.go:89:26` | 0 |  |
| `TestClaudeToolsCommand.RunIntoWriter` | `geppetto/cmd/examples/claude-tools/main.go:100:34` | 0 |  |
| `GenericToolCallingCommand.RunIntoWriter` | `geppetto/cmd/examples/generic-tool-calling/main.go:283:37` | 0 |  |
| `MiddlewareInferenceCommand.RunIntoWriter` | `geppetto/cmd/examples/middleware-inference/main.go:108:38` | 0 |  |
| `TestOpenAIToolsCommand.RunIntoWriter` | `geppetto/cmd/examples/openai-tools/main.go:173:34` | 0 |  |
| `TestCommand.RunIntoWriter` | `pinocchio/cmd/examples/simple-chat/main.go:88:23` | 0 |  |
| `SimpleInferenceCommand.RunIntoWriter` | `geppetto/cmd/examples/simple-inference/main.go:91:34` | 0 |  |
| `SimpleRedisStreamingInferenceCommand.RunIntoWriter` | `pinocchio/cmd/examples/simple-redis-streaming-inference/main.go:109:48` | 0 |  |
| `SimpleStreamingInferenceCommand.RunIntoWriter` | `geppetto/cmd/examples/simple-streaming-inference/main.go:125:43` | 0 |  |
| `RunCommand.Run` | `geppetto/cmd/llm-runner/main.go:69:22` | 0 |  |
| `ReportCommand.Run` | `geppetto/cmd/llm-runner/main.go:140:25` | 0 |  |
| `ServeCommand.Run` | `geppetto/cmd/llm-runner/serve.go:43:24` | 0 |  |
| `getMiddlewares` | `pinocchio/cmd/pinocchio/cmds/catter/catter.go:40:6` | 2 |  |
| `CatterPrintCommand.RunIntoGlazeProcessor` | `pinocchio/cmd/pinocchio/cmds/catter/cmds/print.go:139:30` | 0 |  |
| `createFileFilter` | `pinocchio/cmd/pinocchio/cmds/catter/cmds/print.go:204:6` | 2 |  |
| `CatterStatsCommand.RunIntoGlazeProcessor` | `pinocchio/cmd/pinocchio/cmds/catter/cmds/stats.go:95:30` | 0 |  |
| `WithProcessor` | `pinocchio/cmd/pinocchio/cmds/catter/pkg/fileprocessor.go:120:6` | 1 |  |
| `Stats.PrintStats` | `pinocchio/cmd/pinocchio/cmds/catter/pkg/stats.go:139:17` | 1 |  |
| `Stats.printOverviewAndFileTypes` | `pinocchio/cmd/pinocchio/cmds/catter/pkg/stats.go:175:17` | 1 |  |
| `Stats.printDirStructure` | `pinocchio/cmd/pinocchio/cmds/catter/pkg/stats.go:205:17` | 1 |  |
| `Stats.printFullStructure` | `pinocchio/cmd/pinocchio/cmds/catter/pkg/stats.go:233:17` | 1 |  |
| `ClipCommand.Run` | `pinocchio/cmd/pinocchio/cmds/clip.go:55:23` | 0 |  |
| `ExtractMdCommand.RunIntoWriter` | `pinocchio/cmd/pinocchio/cmds/helpers/md-extract.go:75:28` | 0 |  |
| `EnrichWebCommand.RunIntoGlazeProcessor` | `pinocchio/cmd/pinocchio/cmds/kagi/enrich.go:138:28` | 0 |  |
| `FastGPTCommand.RunIntoWriter` | `pinocchio/cmd/pinocchio/cmds/kagi/fastgpt.go:141:26` | 0 |  |
| `SummarizeCommand.RunIntoWriter` | `pinocchio/cmd/pinocchio/cmds/kagi/summarize.go:95:28` | 0 |  |
| `ListEnginesCommand.RunIntoGlazeProcessor` | `pinocchio/cmd/pinocchio/cmds/openai/openai.go:81:30` | 0 |  |
| `EngineInfoCommand.RunIntoGlazeProcessor` | `pinocchio/cmd/pinocchio/cmds/openai/openai.go:201:29` | 0 |  |
| `TranscribeCommand.RunIntoGlazeProcessor` | `pinocchio/cmd/pinocchio/cmds/openai/transcribe.go:248:29` | 0 |  |
| `CountCommand.RunIntoWriter` | `pinocchio/cmd/pinocchio/cmds/tokens/count.go:55:25` | 0 |  |
| `DecodeCommand.RunIntoWriter` | `pinocchio/cmd/pinocchio/cmds/tokens/decode.go:55:25` | 0 |  |
| `EncodeCommand.RunIntoWriter` | `pinocchio/cmd/pinocchio/cmds/tokens/encode.go:57:27` | 0 |  |
| `ListModelsCommand.RunIntoGlazeProcessor` | `pinocchio/cmd/pinocchio/cmds/tokens/list.go:17:29` | 0 |  |
| `ListCodecsCommand.RunIntoGlazeProcessor` | `pinocchio/cmd/pinocchio/cmds/tokens/list.go:89:29` | 0 |  |
| `Command.RunIntoWriter` | `pinocchio/cmd/web-chat/main.go:66:19` | 0 |  |
| `PinocchioCommand.RunIntoWriter` | `pinocchio/pkg/cmds/cmd.go:192:28` | 0 |  |
| `NewHelpersParameterLayer` | `pinocchio/pkg/cmds/cmdlayers/helpers.go:35:6` | 2 |  |
| `ParseGeppettoLayers` | `pinocchio/pkg/cmds/helpers/parse-helpers.go:48:6` | 1 |  |
| `NewEmbeddingsParameterLayer` | `geppetto/pkg/embeddings/config/settings.go:62:6` | 2 |  |
| `NewEmbeddingsApiKeyParameter` | `geppetto/pkg/embeddings/config/settings.go:70:6` | 0 |  |

