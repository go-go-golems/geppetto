package openai

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

type TranscriptionMode string

const (
	TranscriptionModeBlocking TranscriptionMode = "blocking"
	TranscriptionModeStream   TranscriptionMode = "stream"
)

type TranscriptionOptions struct {
	Mode                   TranscriptionMode
	TimestampGranularities []openai.TranscriptionTimestampGranularity
	ChunkSize              int // For streaming mode
	ProgressCallback       func(float64)
}

type Transcription struct {
	File     string                `json:"file"`
	Response *openai.AudioResponse `json:"response"`
	Error    error                 `json:"error,omitempty"`
}

type StreamingTranscription struct {
	File      string  `json:"file"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
	IsFinal   bool    `json:"is_final"`
	Error     error   `json:"error,omitempty"`
}

type TranscriptionClient struct {
	Client      *openai.Client
	Model       string
	Prompt      string
	Language    string
	Temperature float32
}

func NewTranscriptionClient(apiKey, model, prompt, language string, temperature float32) *TranscriptionClient {
	return &TranscriptionClient{
		Client:      openai.NewClient(apiKey),
		Model:       model,
		Prompt:      prompt,
		Language:    language,
		Temperature: temperature,
	}
}

func (tc *TranscriptionClient) TranscribeFile(ctx context.Context, mp3FilePath string, options TranscriptionOptions) (<-chan interface{}, error) {
	resultChan := make(chan interface{})

	// Set up the audio request
	req := openai.AudioRequest{
		Model:                  tc.Model,
		FilePath:               mp3FilePath,
		Prompt:                 tc.Prompt,
		Temperature:            tc.Temperature,
		Language:               tc.Language,
		Format:                 openai.AudioResponseFormatVerboseJSON,
		TimestampGranularities: options.TimestampGranularities,
	}

	go func() {
		defer close(resultChan)

		if options.Mode == TranscriptionModeStream {
			tc.handleStreamingTranscription(ctx, req, mp3FilePath, options, resultChan)
		} else {
			tc.handleBlockingTranscription(ctx, req, mp3FilePath, resultChan)
		}
	}()

	return resultChan, nil
}

func (tc *TranscriptionClient) handleBlockingTranscription(ctx context.Context, req openai.AudioRequest, mp3FilePath string, out chan<- interface{}) {
	log.Info().Str("file", mp3FilePath).Msg("Transcribing in blocking mode...")

	resp, err := tc.Client.CreateTranscription(ctx, req)
	if err != nil {
		log.Error().Err(err).Str("file", mp3FilePath).Msg("Failed to transcribe")
		out <- Transcription{File: mp3FilePath, Error: err}
		return
	}

	out <- Transcription{File: mp3FilePath, Response: &resp}
}

func (tc *TranscriptionClient) handleStreamingTranscription(ctx context.Context, req openai.AudioRequest, mp3FilePath string, options TranscriptionOptions, out chan<- interface{}) {
	log.Info().Str("file", mp3FilePath).Msg("Transcribing in streaming mode...")

	// Process audio in chunks
	file, err := os.Open(mp3FilePath)
	if err != nil {
		out <- StreamingTranscription{File: mp3FilePath, Error: err}
		return
	}
	defer file.Close()

	buffer := make([]byte, options.ChunkSize)
	var processedBytes int64
	fileInfo, err := file.Stat()
	if err != nil {
		out <- StreamingTranscription{File: mp3FilePath, Error: err}
		return
	}
	totalSize := fileInfo.Size()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := file.Read(buffer)
			if err == io.EOF {
				// Send final transcription
				out <- StreamingTranscription{
					File:      mp3FilePath,
					IsFinal:   true,
					Timestamp: float64(time.Now().UnixNano()) / float64(time.Second),
				}
				return
			}
			if err != nil {
				out <- StreamingTranscription{File: mp3FilePath, Error: err}
				return
			}

			processedBytes += int64(n)
			if options.ProgressCallback != nil {
				options.ProgressCallback(float64(processedBytes) / float64(totalSize))
			}

			// Process chunk
			chunkReq := req
			chunkReq.Reader = bytes.NewReader(buffer[:n])
			resp, err := tc.Client.CreateTranscription(ctx, chunkReq)
			if err != nil {
				out <- StreamingTranscription{File: mp3FilePath, Error: err}
				continue
			}

			out <- StreamingTranscription{
				File:      mp3FilePath,
				Text:      resp.Text,
				Timestamp: float64(time.Now().UnixNano()) / float64(time.Second),
				IsFinal:   false,
			}
		}
	}
}
