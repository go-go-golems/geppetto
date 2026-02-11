# Tasks

## TODO


- [x] Baseline analysis: verify whether helper and streaming bugs still reproduce in current HEAD
- [x] Bug 1 test-first: add regression test showing assistant context loss before reasoning
- [x] Bug 1 fix: preserve all but the final assistant pre-reasoning block
- [x] Bug 1 validation: run focused openai_responses tests and ensure regression test fails before fix then passes after
- [x] Bug 2 test-first: add streaming regression test proving SSE error currently returns success
- [x] Bug 2 fix: return streamErr and suppress final success event when streaming fails
- [x] Bug 1 commit: commit helper+test+diary updates
- [x] Bug 2 validation: run focused openai_responses tests and confirm error propagation semantics
- [ ] Bug 2 commit: commit engine+tests+diary updates
- [ ] Finalize ticket docs: complete analysis, diary, changelog, related file links, and mark tasks done
