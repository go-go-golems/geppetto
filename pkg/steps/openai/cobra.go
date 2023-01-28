package openai

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"math"
	"time"
)

func NewClientSettingsFromCobra(cmd *cobra.Command) (*ClientSettings, error) {
	apiKey := viper.GetString("openai-api-key")
	if apiKey == "" {
		log.Fatal().Msg("openai-api-key is not set")
	}

	timeout, err := cmd.Flags().GetInt("timeout")
	if err != nil {
		return nil, err
	}
	organization, err := cmd.Flags().GetString("organization")
	if err != nil {
		return nil, err
	}
	userAgent, err := cmd.Flags().GetString("user-agent")
	if err != nil {
		return nil, err
	}
	baseUrl, err := cmd.Flags().GetString("base-url")
	if err != nil {
		return nil, err
	}
	defaultEngine, err := cmd.Flags().GetString("default-engine")
	if err != nil {
		return nil, err
	}

	clientSettings := &ClientSettings{
		APIKey: &apiKey,
	}
	timeoutDuration := time.Duration(timeout) * time.Second
	clientSettings.Timeout = &timeoutDuration
	if organization != "" {
		clientSettings.Organization = &organization
	}
	if userAgent != "" {
		clientSettings.UserAgent = &userAgent
	}
	if baseUrl != "" {
		clientSettings.BaseURL = &baseUrl
	}
	if defaultEngine != "" {
		clientSettings.DefaultEngine = &defaultEngine
	}

	return clientSettings, nil
}

// TODO(manuel, 2023-01-28) - we have GenericStepFactory now
func NewCompletionStepSettingsFromCobra(cmd *cobra.Command) (*CompletionStepSettings, error) {
	clientSettings, err := NewClientSettingsFromCobra(cmd)
	if err != nil {
		return nil, err
	}

	factorySettings := &CompletionStepSettings{}
	if cmd.Flags().Changed("engine") {
		engine, err := cmd.Flags().GetString("engine")
		if err != nil {
			return nil, err
		}
		factorySettings.Engine = &engine
	}
	if cmd.Flags().Changed("max-response-tokens") {
		maxResponseTokens, err := cmd.Flags().GetInt("max-response-tokens")
		if err != nil {
			return nil, err
		}
		factorySettings.MaxResponseTokens = &maxResponseTokens
	}
	if cmd.Flags().Changed("temperature") {
		temperature, err := cmd.Flags().GetFloat64("temperature")
		if err != nil {
			return nil, err
		}
		if !math.IsNaN(temperature) {
			temperaturef32 := float32(temperature)
			factorySettings.Temperature = &temperaturef32
		}
	}
	if cmd.Flags().Changed("top-p") {
		topP, err := cmd.Flags().GetFloat64("top-p")
		if err != nil {
			return nil, err
		}
		if !math.IsNaN(topP) {
			topPf32 := float32(topP)
			factorySettings.TopP = &topPf32
		}
	}
	if cmd.Flags().Changed("stop") {
		stop, err := cmd.Flags().GetStringSlice("stop")
		if err != nil {
			return nil, err
		}
		factorySettings.Stop = stop
	}
	if cmd.Flags().Changed("log-probabilities") {
		logProbabilities, err := cmd.Flags().GetInt("log-probabilities")
		if err != nil {
			return nil, err
		}
		if logProbabilities > 0 {
			factorySettings.LogProbs = &logProbabilities
		} else {
			factorySettings.LogProbs = nil
		}
	}
	if cmd.Flags().Changed("n") {
		n, err := cmd.Flags().GetInt("n")
		if err != nil {
			return nil, err
		}
		factorySettings.N = &n
	}
	if cmd.Flags().Changed("stream") {
		stream, err := cmd.Flags().GetBool("stream")
		if err != nil {
			return nil, err
		}
		factorySettings.Stream = stream
	}

	factorySettings.ClientSettings = clientSettings

	return factorySettings, nil
}
