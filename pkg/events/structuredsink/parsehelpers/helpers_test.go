package parsehelpers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testPayload struct {
	Name  string `yaml:"name"`
	Notes string `yaml:"notes,omitempty"`
}

func TestDebounceConfig_SanitizeEnabledDefaultsToTrue(t *testing.T) {
	cfg := DebounceConfig{}

	require.True(t, cfg.SanitizeEnabled())
}

func TestYAMLControllerFinalBytes_SanitizesByDefault(t *testing.T) {
	ctrl := NewDebouncedYAML[testPayload](DebounceConfig{})

	parsed, err := ctrl.FinalBytes([]byte("```yaml\nname:test\nnotes:hello\n```"))
	require.NoError(t, err)
	require.Equal(t, "test", parsed.Name)
	require.Equal(t, "hello", parsed.Notes)
}

func TestYAMLControllerFinalBytes_CanDisableSanitize(t *testing.T) {
	ctrl := NewDebouncedYAML[testPayload](DebounceConfig{}.WithSanitizeYAML(false))

	parsed, err := ctrl.FinalBytes([]byte("```yaml\nname:test\nnotes:hello\n```"))
	require.Error(t, err)
	require.Nil(t, parsed)
}

func TestYAMLControllerFeedBytes_SanitizesSnapshotsByDefault(t *testing.T) {
	ctrl := NewDebouncedYAML[testPayload](DebounceConfig{
		SnapshotOnNewline: true,
	})

	parsed, err := ctrl.FeedBytes([]byte("name:test\nnotes:hello\n"))
	require.NoError(t, err)
	require.Equal(t, "test", parsed.Name)
	require.Equal(t, "hello", parsed.Notes)
}

func TestYAMLControllerFeedBytes_CanDisableSanitize(t *testing.T) {
	ctrl := NewDebouncedYAML[testPayload](DebounceConfig{
		SnapshotOnNewline: true,
	}.WithSanitizeYAML(false))

	parsed, err := ctrl.FeedBytes([]byte("name:test\nnotes:hello\n"))
	require.Error(t, err)
	require.Nil(t, parsed)
}

func TestYAMLControllerFinalBytes_PreservesValidYAML(t *testing.T) {
	ctrl := NewDebouncedYAML[testPayload](DebounceConfig{})

	parsed, err := ctrl.FinalBytes([]byte("```yaml\nname: test\nnotes: hello\n```"))
	require.NoError(t, err)
	require.Equal(t, "test", parsed.Name)
	require.Equal(t, "hello", parsed.Notes)
}
