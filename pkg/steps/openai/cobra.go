package openai

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
