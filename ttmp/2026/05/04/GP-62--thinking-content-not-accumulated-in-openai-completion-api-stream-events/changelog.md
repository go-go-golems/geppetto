# Changelog

## 2026-05-04

- Initial workspace created


## 2026-05-04

Created full analysis and implementation guide for thinking content accumulation bug. Identified 3 root causes: (1) EventReasoningTextDelta has no Completion field, (2) runtime_debug_feature.go passes ev.Delta as content, (3) JS encoder missing EventReasoningTextDelta case.


## 2026-05-04

Updated design doc: decided to delete EventReasoningTextDelta entirely instead of patching it. Single consumer (runtime_debug_feature.go) will switch to EventThinkingPartial. Simpler fix, removes dead code.


## 2026-05-04

Implementation complete. 4 commits: (1) switch runtime_debug_feature to EventThinkingPartial, (2) remove emissions from OpenAI engine, (3) remove emissions from Responses engine, (4) delete type definitions. All tests pass, all three repos compile.

