package runner

import "errors"

var (
	ErrRunnerNil              = errors.New("runner is nil")
	ErrRuntimeStepSettingsNil = errors.New("runtime step settings are nil")
	ErrPromptAndSeedEmpty     = errors.New("prompt and seed turn are both empty")
)
