package openai

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"sync"
)

type Transcription struct {
	File     string                `json:"file"`
	Response *openai.AudioResponse `json:"response"`
	err      error
}

type TranscriptionClient struct {
	client      *openai.Client
	model       string
	prompt      string
	language    string
	temperature float32
}

func NewTranscriptionClient(apiKey, model, prompt, language string, temperature float32) *TranscriptionClient {
	return &TranscriptionClient{
		client:      openai.NewClient(apiKey),
		model:       model,
		prompt:      prompt,
		language:    language,
		temperature: temperature,
	}
}

func (tc *TranscriptionClient) TranscribeFile(mp3FilePath string, out chan<- Transcription, wg *sync.WaitGroup) {
	defer wg.Done()

	// Set up the audio request
	req := openai.AudioRequest{
		Model:       tc.model,
		FilePath:    mp3FilePath,
		Prompt:      tc.prompt,
		Temperature: tc.temperature,
		Language:    tc.language,
		Format:      openai.AudioResponseFormatVerboseJSON,
	}

	log.Info().Str("file", mp3FilePath).Msg("Transcribing...")
	// Call the CreateTranscription method
	resp, err := tc.client.CreateTranscription(context.Background(), req)
	if err != nil {
		log.Printf("Failed to transcribe %s: %v\n", mp3FilePath, err)
		out <- Transcription{File: mp3FilePath, err: err}
		return
	}

	out <- Transcription{File: mp3FilePath, Response: &resp}
}
