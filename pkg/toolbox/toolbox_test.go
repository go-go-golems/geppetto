package toolbox

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test function for weather tool
type WeatherResult struct {
	City        string  `json:"city" jsonschema:"description=The city name"`
	Temperature float64 `json:"temperature" jsonschema:"description=Temperature in Celsius"`
	Condition   string  `json:"condition" jsonschema:"description=Weather condition"`
	Humidity    int     `json:"humidity" jsonschema:"description=Humidity percentage"`
}

// GetWeather is a test weather function
// jsonschema:description Get weather information for a city
func GetWeather(city string) WeatherResult {
	// Return fake weather data
	return WeatherResult{
		City:        city,
		Temperature: 22.5,
		Condition:   "Sunny",
		Humidity:    65,
	}
}

// Add is a simple addition function for testing
// jsonschema:description Add two numbers together
func Add(a, b int) int {
	return a + b
}

// Greet creates a greeting message
// jsonschema:description Create a greeting message for a person
func Greet(name string) string {
	return "Hello, " + name + "!"
}

func TestNewRealToolbox(t *testing.T) {
	tb := NewRealToolbox()
	assert.NotNil(t, tb)
	assert.Empty(t, tb.GetToolNames())
	assert.Empty(t, tb.GetToolDescriptions())
}

func TestRegisterTool(t *testing.T) {
	tb := NewRealToolbox()

	// Test registering a valid function
	err := tb.RegisterTool("get_weather", GetWeather)
	require.NoError(t, err)

	// Check that tool was registered
	assert.True(t, tb.HasTool("get_weather"))
	assert.Contains(t, tb.GetToolNames(), "get_weather")

	descriptions := tb.GetToolDescriptions()
	require.Len(t, descriptions, 1)
	assert.Equal(t, "get_weather", descriptions[0].Name)
	assert.NotNil(t, descriptions[0].Parameters)

	// Test registering non-function should fail
	err = tb.RegisterTool("not_a_function", "this is not a function")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a function")
}

func TestExecuteTool(t *testing.T) {
	tb := NewRealToolbox()

	// Register tools
	require.NoError(t, tb.RegisterTool("get_weather", GetWeather))
	require.NoError(t, tb.RegisterTool("add", Add))
	require.NoError(t, tb.RegisterTool("greet", Greet))

	ctx := context.Background()

	t.Run("execute weather tool", func(t *testing.T) {
		result, err := tb.ExecuteTool(ctx, "get_weather", map[string]interface{}{
			"city": "San Francisco",
		})
		require.NoError(t, err)

		weatherResult, ok := result.(WeatherResult)
		require.True(t, ok)
		assert.Equal(t, "San Francisco", weatherResult.City)
		assert.Equal(t, 22.5, weatherResult.Temperature)
		assert.Equal(t, "Sunny", weatherResult.Condition)
		assert.Equal(t, 65, weatherResult.Humidity)
	})

	t.Run("execute add tool", func(t *testing.T) {
		result, err := tb.ExecuteTool(ctx, "add", map[string]interface{}{
			"a": 5,
			"b": 3,
		})
		require.NoError(t, err)

		sum, ok := result.(int)
		require.True(t, ok)
		assert.Equal(t, 8, sum)
	})

	t.Run("execute greet tool", func(t *testing.T) {
		result, err := tb.ExecuteTool(ctx, "greet", map[string]interface{}{
			"name": "Alice",
		})
		require.NoError(t, err)

		greeting, ok := result.(string)
		require.True(t, ok)
		assert.Equal(t, "Hello, Alice!", greeting)
	})

	t.Run("execute non-existent tool", func(t *testing.T) {
		_, err := tb.ExecuteTool(ctx, "non_existent", map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestMultipleRegistrations(t *testing.T) {
	tb := NewRealToolbox()

	// Register multiple tools
	tools := map[string]interface{}{
		"get_weather": GetWeather,
		"add":         Add,
		"greet":       Greet,
	}

	for name, fn := range tools {
		err := tb.RegisterTool(name, fn)
		require.NoError(t, err)
	}

	// Check all tools are registered
	toolNames := tb.GetToolNames()
	assert.Len(t, toolNames, 3)
	for name := range tools {
		assert.Contains(t, toolNames, name)
		assert.True(t, tb.HasTool(name))
	}

	descriptions := tb.GetToolDescriptions()
	assert.Len(t, descriptions, 3)
}

func TestUpdateExistingTool(t *testing.T) {
	tb := NewRealToolbox()

	// Register a tool
	err := tb.RegisterTool("greet", Greet)
	require.NoError(t, err)

	initialDescriptions := tb.GetToolDescriptions()
	require.Len(t, initialDescriptions, 1)

	// Register a tool with the same name (should update, not add)
	err = tb.RegisterTool("greet", Add) // Different function but same name
	require.NoError(t, err)

	updatedDescriptions := tb.GetToolDescriptions()
	assert.Len(t, updatedDescriptions, 1) // Should still be 1, not 2
	assert.Equal(t, "greet", updatedDescriptions[0].Name)
}
