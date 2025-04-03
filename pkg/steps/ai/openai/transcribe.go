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
	TimestampGranularities []openai.TranscriptionTimestampGranularity
	ChunkSize              int // For streaming mode
	ProgressCallback       func(float64)
	Format                 openai.AudioResponseFormat

	// File Processing Options
	MaxDuration float64 // Maximum duration in seconds to process
	StartTime   float64 // Start processing from this timestamp (in seconds)

	// Quality and Performance Options
	Quality          float32 // 0.0 = fastest/lowest quality, 1.0 = slowest/highest quality
	ConcurrentChunks int     // Number of concurrent chunks to process in streaming mode

	// Output Formatting Options
	EnableSpeakerDiarization bool // Include speaker diarization
	MinSpeakers              int  // Minimum number of speakers to detect
	MaxSpeakers              int  // Maximum number of speakers to detect
	AllowProfanity           bool // Include profanity in output

	// Advanced Callback Options
	OnSpeakerChange          func(oldSpeaker, newSpeaker string)
	OnSilence                func(duration float64)
	OnMusic                  func(startTime, endTime float64)
	OnNoise                  func(startTime, endTime float64, level float64)
	DetailedProgressCallback func(ProgressInfo)

	// Error Handling Options
	MaxRetries    int           // Maximum number of retries for failed chunks
	RetryDelay    time.Duration // Delay between retries
	FailFast      bool          // Stop on first error
	ErrorCallback func(error)   // Called when an error occurs

	// Rate Limiting Options
	RequestsPerMinute int           // Maximum requests per minute
	MinRequestGap     time.Duration // Minimum time between requests
	CooldownPeriod    time.Duration // Time to wait when rate limit is hit
}

type TranscriptionOption func(*TranscriptionOptions)

// File Processing Options

func WithMaxDuration(seconds float64) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.MaxDuration = seconds
	}
}

func WithStartTime(seconds float64) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.StartTime = seconds
	}
}

// Quality and Performance Options

func WithQuality(quality float32) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		if quality < 0 {
			quality = 0
		}
		if quality > 1 {
			quality = 1
		}
		o.Quality = quality
	}
}

func WithConcurrentChunks(n int) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		if n < 1 {
			n = 1
		}
		o.ConcurrentChunks = n
	}
}

// Output Formatting Options

func WithSpeakerDiarization() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.EnableSpeakerDiarization = true
	}
}

func WithSpeakerLimits(min_ int, max_ int) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		if min_ < 1 {
			min_ = 1
		}
		if max_ < min_ {
			max_ = min_
		}
		o.MinSpeakers = min_
		o.MaxSpeakers = max_
	}
}

func WithProfanity() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.AllowProfanity = true
	}
}

func WithWordTimestamps() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.TimestampGranularities = append(o.TimestampGranularities, openai.TranscriptionTimestampGranularityWord)
	}
}

func WithSegmentTimestamps() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.TimestampGranularities = append(o.TimestampGranularities, openai.TranscriptionTimestampGranularitySegment)
	}
}

func WithJSONFormat() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Format = openai.AudioResponseFormatJSON
	}
}

func WithVerboseJSONFormat() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Format = openai.AudioResponseFormatVerboseJSON
	}
}

func WithTextFormat() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Format = openai.AudioResponseFormatText
	}
}

func WithSRTFormat() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Format = openai.AudioResponseFormatSRT
	}
}

func WithVTTFormat() TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Format = openai.AudioResponseFormatVTT
	}
}

func WithChunkSize(size int) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.ChunkSize = size
	}
}

func WithProgressCallback(callback func(float64)) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.ProgressCallback = callback
	}
}

// Advanced Callback Options

func WithSpeakerChangeCallback(callback func(oldSpeaker, newSpeaker string)) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.OnSpeakerChange = callback
	}
}

func WithSilenceCallback(callback func(duration float64)) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.OnSilence = callback
	}
}

func WithMusicCallback(callback func(startTime, endTime float64)) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.OnMusic = callback
	}
}

func WithNoiseCallback(callback func(startTime, endTime float64, level float64)) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.OnNoise = callback
	}
}

func WithDetailedProgress(callback func(ProgressInfo)) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.DetailedProgressCallback = callback
	}
}

// Error Handling Options

func WithMaxRetries(retries int) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		if retries < 0 {
			retries = 0
		}
		o.MaxRetries = retries
	}
}

func WithRetryDelay(delay time.Duration) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		if delay < 0 {
			delay = 0
		}
		o.RetryDelay = delay
	}
}

func WithFailFast(failFast bool) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.FailFast = failFast
	}
}

func WithErrorCallback(callback func(error)) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.ErrorCallback = callback
	}
}

// Rate Limiting Options

func WithRequestsPerMinute(rpm int) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		if rpm < 1 {
			rpm = 1
		}
		o.RequestsPerMinute = rpm
	}
}

func WithMinRequestGap(gap time.Duration) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		if gap < 0 {
			gap = 0
		}
		o.MinRequestGap = gap
	}
}

func WithCooldownPeriod(period time.Duration) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		if period < 0 {
			period = 0
		}
		o.CooldownPeriod = period
	}
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

type SpeakerInfo struct {
	ID       string
	Duration float64
}

type ProgressInfo struct {
	BytesProcessed    int64
	TotalBytes        int64
	PercentComplete   float64
	CurrentSpeaker    *SpeakerInfo
	ElapsedTime       time.Duration
	EstimatedTimeLeft time.Duration
}

type TranscriptionClient struct {
	client      *openai.Client
	model       string
	prompt      string
	language    string
	temperature float32
}

type ClientOption func(*TranscriptionClient)

func WithModel(model string) ClientOption {
	return func(c *TranscriptionClient) {
		c.model = model
	}
}

func WithPrompt(prompt string) ClientOption {
	return func(c *TranscriptionClient) {
		c.prompt = prompt
	}
}

func WithLanguage(language string) ClientOption {
	return func(c *TranscriptionClient) {
		c.language = language
	}
}

func WithTemperature(temperature float32) ClientOption {
	return func(c *TranscriptionClient) {
		c.temperature = temperature
	}
}

func NewTranscriptionClient(apiKey string, opts ...ClientOption) *TranscriptionClient {
	client := &TranscriptionClient{
		client:      openai.NewClient(apiKey),
		model:       "whisper-1",
		temperature: 0,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

func (tc *TranscriptionClient) TranscribeFile(ctx context.Context, mp3FilePath string, opts ...TranscriptionOption) (*Transcription, error) {
	options := &TranscriptionOptions{
		Format:           openai.AudioResponseFormatVerboseJSON, // Default format
		Quality:          0.5,                                   // Default middle quality
		ConcurrentChunks: 1,                                     // Default single chunk
	}
	for _, opt := range opts {
		opt(options)
	}

	// Handle start time and max duration by modifying the audio file if needed
	if options.StartTime > 0 || options.MaxDuration > 0 {
		// TODO: Implement audio trimming using ffmpeg or similar
		log.Warn().Msg("Start time and max duration options are not implemented yet")
	}

	// Set up the audio request
	req := openai.AudioRequest{
		Model:                  tc.model,
		FilePath:               mp3FilePath,
		Prompt:                 tc.prompt,
		Temperature:            tc.temperature,
		Language:               tc.language,
		Format:                 options.Format,
		TimestampGranularities: options.TimestampGranularities,
	}

	log.Info().
		Str("file", mp3FilePath).
		Float32("quality", options.Quality).
		Bool("speaker_diarization", options.EnableSpeakerDiarization).
		Int("min_speakers", options.MinSpeakers).
		Int("max_speakers", options.MaxSpeakers).
		Bool("allow_profanity", options.AllowProfanity).
		Msg("Transcribing in blocking mode...")

	resp, err := tc.client.CreateTranscription(ctx, req)
	if err != nil {
		log.Error().Err(err).Str("file", mp3FilePath).Msg("Failed to transcribe")
		return &Transcription{File: mp3FilePath, Error: err}, nil
	}

	return &Transcription{File: mp3FilePath, Response: &resp}, nil
}

func (tc *TranscriptionClient) TranscribeFileStreaming(ctx context.Context, mp3FilePath string, opts ...TranscriptionOption) (<-chan StreamingTranscription, error) {
	options := &TranscriptionOptions{
		ChunkSize:        1024 * 1024,                           // Default 1MB chunk size
		Format:           openai.AudioResponseFormatVerboseJSON, // Default format
		Quality:          0.5,                                   // Default middle quality
		ConcurrentChunks: 1,                                     // Default single chunk
	}
	for _, opt := range opts {
		opt(options)
	}

	// Handle start time and max duration by modifying the audio file if needed
	if options.StartTime > 0 || options.MaxDuration > 0 {
		// TODO: Implement audio trimming using ffmpeg or similar
		log.Warn().Msg("Start time and max duration options are not implemented yet")
	}

	resultChan := make(chan StreamingTranscription)

	// Set up the audio request
	req := openai.AudioRequest{
		Model:                  tc.model,
		FilePath:               mp3FilePath,
		Prompt:                 tc.prompt,
		Temperature:            tc.temperature,
		Language:               tc.language,
		Format:                 options.Format,
		TimestampGranularities: options.TimestampGranularities,
	}

	log.Info().
		Str("file", mp3FilePath).
		Float32("quality", options.Quality).
		Int("concurrent_chunks", options.ConcurrentChunks).
		Bool("speaker_diarization", options.EnableSpeakerDiarization).
		Int("min_speakers", options.MinSpeakers).
		Int("max_speakers", options.MaxSpeakers).
		Bool("allow_profanity", options.AllowProfanity).
		Msg("Transcribing in streaming mode...")

	go func() {
		defer close(resultChan)
		if options.ConcurrentChunks > 1 {
			tc.handleConcurrentStreamingTranscription(ctx, req, mp3FilePath, options, resultChan)
		} else {
			tc.handleStreamingTranscription(ctx, req, mp3FilePath, options, resultChan)
		}
	}()

	return resultChan, nil
}

func (tc *TranscriptionClient) handleConcurrentStreamingTranscription(
	ctx context.Context,
	req openai.AudioRequest,
	mp3FilePath string,
	options *TranscriptionOptions,
	out chan<- StreamingTranscription,
) {
	// TODO: Implement concurrent chunk processing
	// This would involve:
	// 1. Splitting the file into chunks
	// 2. Processing chunks concurrently with a worker pool
	// 3. Maintaining order of results
	// 4. Handling errors appropriately
	log.Warn().Msg("Concurrent chunk processing is not implemented yet")
	tc.handleStreamingTranscription(ctx, req, mp3FilePath, options, out)
}

func (tc *TranscriptionClient) handleStreamingTranscription(
	ctx context.Context,
	req openai.AudioRequest,
	mp3FilePath string,
	options *TranscriptionOptions,
	out chan<- StreamingTranscription,
) {
	log.Info().Str("file", mp3FilePath).Msg("Transcribing in streaming mode...")

	// Process audio in chunks
	file, err := os.Open(mp3FilePath)
	if err != nil {
		tc.handleError(err, options, out, mp3FilePath)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Just log the error since we're already returning
			log.Warn().Err(err).Str("file", mp3FilePath).Msg("Failed to close file")
		}
	}()

	buffer := make([]byte, options.ChunkSize)
	var processedBytes int64
	fileInfo, err := file.Stat()
	if err != nil {
		tc.handleError(err, options, out, mp3FilePath)
		return
	}
	totalSize := fileInfo.Size()

	// Rate limiting setup
	var lastRequestTime time.Time
	if options.RequestsPerMinute > 0 {
		requestGap := time.Minute / time.Duration(options.RequestsPerMinute)
		if options.MinRequestGap > requestGap {
			requestGap = options.MinRequestGap
		}
		rateLimiter := time.NewTicker(requestGap)
		defer rateLimiter.Stop()
	}

	startTime := time.Now()
	var currentSpeaker *SpeakerInfo

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := file.Read(buffer)
			if err == io.EOF {
				out <- StreamingTranscription{
					File:      mp3FilePath,
					IsFinal:   true,
					Timestamp: float64(time.Now().UnixNano()) / float64(time.Second),
				}
				return
			}
			if err != nil {
				tc.handleError(err, options, out, mp3FilePath)
				if options.FailFast {
					return
				}
				continue
			}

			processedBytes += int64(n)

			// Update progress
			if options.DetailedProgressCallback != nil {
				elapsed := time.Since(startTime)
				bytesPerSecond := float64(processedBytes) / elapsed.Seconds()
				remainingBytes := totalSize - processedBytes
				estimatedTimeLeft := time.Duration(float64(remainingBytes)/bytesPerSecond) * time.Second

				options.DetailedProgressCallback(ProgressInfo{
					BytesProcessed:    processedBytes,
					TotalBytes:        totalSize,
					PercentComplete:   float64(processedBytes) / float64(totalSize),
					CurrentSpeaker:    currentSpeaker,
					ElapsedTime:       elapsed,
					EstimatedTimeLeft: estimatedTimeLeft,
				})
			} else if options.ProgressCallback != nil {
				options.ProgressCallback(float64(processedBytes) / float64(totalSize))
			}

			// Process chunk with retries
			var resp openai.AudioResponse
			var transcriptionErr error
			for retry := 0; retry <= options.MaxRetries; retry++ {
				if retry > 0 {
					time.Sleep(options.RetryDelay)
				}

				// Rate limiting
				if options.RequestsPerMinute > 0 {
					timeSinceLastRequest := time.Since(lastRequestTime)
					if timeSinceLastRequest < options.MinRequestGap {
						time.Sleep(options.MinRequestGap - timeSinceLastRequest)
					}
				}

				chunkReq := req
				chunkReq.Reader = bytes.NewReader(buffer[:n])
				resp, transcriptionErr = tc.client.CreateTranscription(ctx, chunkReq)
				lastRequestTime = time.Now()

				if transcriptionErr == nil {
					break
				}

				if options.ErrorCallback != nil {
					options.ErrorCallback(transcriptionErr)
				}

				if retry == options.MaxRetries {
					tc.handleError(transcriptionErr, options, out, mp3FilePath)
					if options.FailFast {
						return
					}
					break
				}
			}

			if transcriptionErr != nil {
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

func (tc *TranscriptionClient) handleError(err error, options *TranscriptionOptions, out chan<- StreamingTranscription, filePath string) {
	if options.ErrorCallback != nil {
		options.ErrorCallback(err)
	}
	out <- StreamingTranscription{File: filePath, Error: err}
}
