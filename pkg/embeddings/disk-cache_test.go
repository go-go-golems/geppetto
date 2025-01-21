package embeddings

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTest creates a temporary directory and sets it as HOME
func setupTest(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}
	})
	return tmpDir
}

// MockProvider implements Provider for testing
type MockProvider struct {
	model      EmbeddingModel
	embeddings map[string][]float32
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		model: EmbeddingModel{
			Name:       "test-model",
			Dimensions: 3,
		},
		embeddings: make(map[string][]float32),
	}
}

func (m *MockProvider) GenerateEmbedding(_ context.Context, text string) ([]float32, error) {
	if e, ok := m.embeddings[text]; ok {
		return e, nil
	}
	// Return predictable embedding based on text length
	return []float32{float32(len(text)), 1.0, 2.0}, nil
}

func (m *MockProvider) GetModel() EmbeddingModel {
	return m.model
}

func TestNewDiskCacheProvider(t *testing.T) {
	t.Run("default settings", func(t *testing.T) {
		_ = setupTest(t)
		provider := NewMockProvider()
		cache, err := NewDiskCacheProvider(provider)
		require.NoError(t, err)
		assert.Contains(t, cache.directory, "test-model")
		assert.Equal(t, int64(1<<30), cache.maxSize)
		assert.Equal(t, 10000, cache.maxEntries)
	})

	t.Run("custom settings", func(t *testing.T) {
		tmpDir := setupTest(t)
		customDir := filepath.Join(tmpDir, "custom")
		provider := NewMockProvider()
		cache, err := NewDiskCacheProvider(provider,
			WithDirectory(customDir),
			WithMaxSize(1000),
			WithMaxEntries(10))
		require.NoError(t, err)
		assert.Equal(t, customDir, cache.directory)
		assert.Equal(t, int64(1000), cache.maxSize)
		assert.Equal(t, 10, cache.maxEntries)
	})
}

func TestBasicCacheOperations(t *testing.T) {
	tmpDir := setupTest(t)
	provider := NewMockProvider()
	cache, err := NewDiskCacheProvider(provider, WithDirectory(tmpDir))
	require.NoError(t, err)

	t.Run("store and retrieve", func(t *testing.T) {
		text := "test text"
		embedding, err := cache.GenerateEmbedding(context.Background(), text)
		require.NoError(t, err)
		assert.Equal(t, []float32{9.0, 1.0, 2.0}, embedding)

		// Verify it's cached
		entry, err := cache.GetCachedEntry(text)
		require.NoError(t, err)
		assert.Equal(t, embedding, entry.Embedding)
		assert.Equal(t, text, entry.TextPrefix)
	})

	t.Run("retrieve non-existent", func(t *testing.T) {
		entry, err := cache.GetCachedEntry("non-existent")
		require.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("empty text", func(t *testing.T) {
		embedding, err := cache.GenerateEmbedding(context.Background(), "")
		require.NoError(t, err)
		assert.Equal(t, []float32{0.0, 1.0, 2.0}, embedding)
	})

	t.Run("long text prefix", func(t *testing.T) {
		longText := string(make([]byte, 200))
		_, err := cache.GenerateEmbedding(context.Background(), longText)
		require.NoError(t, err)

		entry, err := cache.GetCachedEntry(longText)
		require.NoError(t, err)
		assert.Equal(t, 100, len(entry.TextPrefix))
	})
}

func TestCacheLimits(t *testing.T) {
	tmpDir := setupTest(t)
	provider := NewMockProvider()
	maxEntries := 3
	cache, err := NewDiskCacheProvider(provider,
		WithDirectory(tmpDir),
		WithMaxEntries(maxEntries))
	require.NoError(t, err)

	t.Run("max entries", func(t *testing.T) {
		// Add entries up to limit
		for i := 0; i < maxEntries+1; i++ {
			text := string(make([]byte, i+1))
			_, err := cache.GenerateEmbedding(context.Background(), text)
			require.NoError(t, err)
			time.Sleep(100 * time.Millisecond) // Ensure different timestamps
		}

		// Verify oldest was removed
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, maxEntries, len(files))
	})
}

func TestCacheFileOperations(t *testing.T) {
	tmpDir := setupTest(t)
	provider := NewMockProvider()
	cache, err := NewDiskCacheProvider(provider, WithDirectory(tmpDir))
	require.NoError(t, err)

	t.Run("file format", func(t *testing.T) {
		text := "test text"
		_, err := cache.GenerateEmbedding(context.Background(), text)
		require.NoError(t, err)

		// Read the cache file directly
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		require.Equal(t, 1, len(files))

		data, err := os.ReadFile(filepath.Join(tmpDir, files[0].Name()))
		require.NoError(t, err)

		var entry DiskCacheEntry
		err = json.Unmarshal(data, &entry)
		require.NoError(t, err)
		assert.Equal(t, text, entry.TextPrefix)
		assert.Equal(t, []float32{9.0, 1.0, 2.0}, entry.Embedding)
	})

	t.Run("corrupted cache file", func(t *testing.T) {
		text := "corrupted"
		_, err := cache.GenerateEmbedding(context.Background(), text)
		require.NoError(t, err)

		// Corrupt the cache file
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		for _, f := range files {
			if f.Name() != "." {
				err = os.WriteFile(filepath.Join(tmpDir, f.Name()), []byte("invalid json"), 0644)
				require.NoError(t, err)
			}
		}

		// Should regenerate embedding
		embedding, err := cache.GenerateEmbedding(context.Background(), text)
		require.NoError(t, err)
		assert.Equal(t, []float32{9.0, 1.0, 2.0}, embedding)
	})
}

func TestClearCache(t *testing.T) {
	tmpDir := setupTest(t)
	provider := NewMockProvider()
	cache, err := NewDiskCacheProvider(provider, WithDirectory(tmpDir))
	require.NoError(t, err)

	t.Run("clear with entries", func(t *testing.T) {
		// Add some entries
		texts := []string{"one", "two", "three"}
		for _, text := range texts {
			_, err := cache.GenerateEmbedding(context.Background(), text)
			require.NoError(t, err)
		}

		// Clear cache
		err = cache.ClearCache()
		require.NoError(t, err)

		// Verify directory is empty
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, 0, len(files))

		// Verify directory still exists
		_, err = os.Stat(tmpDir)
		require.NoError(t, err)
	})
}

func TestGetModel(t *testing.T) {
	tmpDir := setupTest(t)
	provider := NewMockProvider()
	cache, err := NewDiskCacheProvider(provider, WithDirectory(tmpDir))
	require.NoError(t, err)

	model := cache.GetModel()
	assert.Equal(t, provider.GetModel(), model)
}

func TestEdgeCases(t *testing.T) {
	t.Run("zero-length embedding", func(t *testing.T) {
		tmpDir := setupTest(t)
		provider := &MockProvider{
			model: EmbeddingModel{Name: "test", Dimensions: 0},
			embeddings: map[string][]float32{
				"empty": {},
			},
		}
		cache, err := NewDiskCacheProvider(provider, WithDirectory(tmpDir))
		require.NoError(t, err)

		embedding, err := cache.GenerateEmbedding(context.Background(), "empty")
		require.NoError(t, err)
		assert.Empty(t, embedding)
	})

	t.Run("special characters in text", func(t *testing.T) {
		tmpDir := setupTest(t)
		provider := NewMockProvider()
		cache, err := NewDiskCacheProvider(provider, WithDirectory(tmpDir))
		require.NoError(t, err)

		text := "!@#$%^&*()\n\t"
		embedding, err := cache.GenerateEmbedding(context.Background(), text)
		require.NoError(t, err)
		assert.NotNil(t, embedding)

		// Verify it's cached
		entry, err := cache.GetCachedEntry(text)
		require.NoError(t, err)
		assert.Equal(t, embedding, entry.Embedding)
	})
}
