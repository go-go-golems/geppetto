package embeddings

import (
	"context"
	"sync"
)

// DefaultGenerateBatchEmbeddings provides a default implementation of batch processing
// by calling GenerateEmbedding for each text sequentially.
// This can be used by Provider implementations that don't have native batch support.
func DefaultGenerateBatchEmbeddings(ctx context.Context, p Provider, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := p.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = embedding
	}
	return results, nil
}

// ParallelGenerateBatchEmbeddings provides a concurrent implementation of batch processing
// by calling GenerateEmbedding for each text in parallel with a limit on concurrency.
// This can be more efficient than sequential processing for providers with rate limiting.
func ParallelGenerateBatchEmbeddings(ctx context.Context, p Provider, texts []string, maxConcurrency int) ([][]float32, error) {
	if maxConcurrency <= 0 {
		maxConcurrency = 4 // Default concurrency
	}

	total := len(texts)
	results := make([][]float32, total)
	errs := make([]error, total)

	// Use a semaphore to limit concurrency
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, txt string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			embedding, err := p.GenerateEmbedding(ctx, txt)
			results[idx] = embedding
			errs[idx] = err
		}(i, text)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}
