package runner

import "errors"

var (
	ErrRunnerNil                   = errors.New("runner is nil")
	ErrRuntimeInferenceSettingsNil = errors.New("runtime inference settings are nil")
	ErrPromptAndSeedEmpty          = errors.New("prompt and seed turn are both empty")
	ErrRequestedToolMissing        = errors.New("requested tool is not registered")
)
