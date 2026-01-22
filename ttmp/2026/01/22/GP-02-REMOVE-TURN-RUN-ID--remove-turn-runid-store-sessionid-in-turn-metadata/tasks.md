# Tasks

## TODO

- [x] Update analysis doc with decisions: `NewSession()` creates `SessionID`; `StartInference` rejects empty seed turn
- [x] Add `session.NewSession()` constructor that generates `SessionID` (uuid)
- [x] Change `Session.StartInference` to fail if there is no seed turn or the seed turn has 0 blocks
- [x] Add `turns.KeyTurnMetaSessionID` (`geppetto.session_id@v1`) in `geppetto/pkg/turns/keys.go`
- [x] Change `Session.Append` to set `KeyTurnMetaSessionID` (instead of `t.RunID`)
- [x] Add `turns.KeyTurnMetaInferenceID` (`geppetto.inference_id@v1`) and set it per-inference (unique per `RunInference` call)
- [x] Decide policy: keep runner best-effort session-id injection (session + tool-loop runner both set when missing)
- [x] Remove `RunID` from `turns.Turn` in `geppetto/pkg/turns/types.go`
- [x] Remove `runID` parameter from `TurnPersister.PersistTurn` (derive from `Turn.Metadata`)
- [x] Update `ToolLoopEngineBuilder` runner to set/read session id via metadata and call new `PersistTurn` signature
- [x] Update provider engines to populate `events.EventMetadata.RunID` from `Turn.Metadata` session id
- [x] Update middleware logging to use metadata session id for log field `run_id`
- [x] Update tests: `geppetto/pkg/inference/session/session_test.go`
- [x] Update tests: `geppetto/pkg/inference/toolhelpers/helpers_test.go`
- [x] Update tests: `geppetto/pkg/turns/serde/serde_test.go`
- [x] Update examples under `geppetto/cmd/examples/*` to stop setting `Turn.RunID`
- [x] Update docs: `geppetto/pkg/doc/topics/08-turns.md` YAML example (remove `run_id:`, show metadata key)
- [x] Run tests: `GOCACHE=/tmp/go-build-cache go test ./... -count=1` (from `geppetto/`)
- [ ] (Optional follow-up) Update any remaining docs mentioning `Turn.RunID` to describe `KeyTurnMetaSessionID` instead
- [x] moments/backend: rename Conversation.RunID -> SessionID and update webchat/router/loops
- [x] moments/backend: rename FindConversationByRunID -> FindConversationBySessionID across sinks/adapters
- [x] moments/backend: replace EventMetadata.RunID usage with SessionID (+ add InferenceID wiring)
- [x] moments/backend: update SEM handler caches from RunID to SessionID and fix tests
