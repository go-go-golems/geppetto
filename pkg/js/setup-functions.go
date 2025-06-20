package js

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/rs/zerolog/log"
)

// SetupConversation creates a setup function for conversation handling
func SetupConversation() SetupFunction {
	return func(vm *goja.Runtime, engine *RuntimeEngine) {
		log.Debug().Msg("Setting up conversation")
		err := RegisterConversation(vm)
		if err != nil {
			log.Error().Err(err).Msg("Failed to register conversation")
		}
	}
}

// SetupEmbeddings creates a setup function for embeddings
func SetupEmbeddings(stepSettings *settings.StepSettings) SetupFunction {
	return func(vm *goja.Runtime, engine *RuntimeEngine) {
		log.Debug().Msg("Setting up embeddings")
		factory := embeddings.NewSettingsFactoryFromStepSettings(stepSettings)
		provider, err := factory.NewProvider()
		if err != nil {
			log.Error().Err(err).Msg("Failed to create embeddings provider")
			return
		}

		err = RegisterEmbeddings(vm, "embeddings", provider, engine.Loop)
		if err != nil {
			log.Error().Err(err).Msg("Failed to register embeddings")
		}
	}
}
