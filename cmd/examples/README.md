# Geppetto Example Programs

This directory is split into two tiers.

## Start Here

These are the smallest examples that match the current recommended API surface:

- `runner-simple/`: smallest blocking `pkg/inference/runner` program
- `runner-tools/`: same runner API with one function tool
- `runner-streaming/`: same runner API with event sinks
- `runner-registry/`: runner API with profile-registry runtime selection
- `runner-glazed-full-flags/`: runner API driven by full Geppetto sections and Glazed/Cobra parsing
- `runner-glazed-registry-flags/`: runner API with only profile-registry selection exposed publicly through Glazed; base `StepSettings` stay hidden in app bootstrap
- `inference/`: direct engine/session blocking example
- `streaming-inference/`: direct engine/session streaming example

If you are new to Geppetto, start with the `runner-*` programs first.

## Advanced

These examples are intentionally lower-level or provider-specific:

- `advanced/generic-tool-calling/`
- `advanced/openai-tools/`
- `advanced/claude-tools/`
- `advanced/middleware-inference/`

Use those when you want to study `session.Session`, `enginebuilder.Builder`, provider-native tool behavior, or manual event-router assembly.
