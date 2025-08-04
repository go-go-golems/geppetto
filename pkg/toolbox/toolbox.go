package toolbox

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/inference"
	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
)

// RealToolbox implements the Toolbox interface using reflection-based tool execution
type RealToolbox struct {
	tools         map[string]interface{}
	descriptions  []inference.ToolDescription
	reflector     *jsonschema.Reflector
}

// NewRealToolbox creates a new real toolbox instance
func NewRealToolbox() *RealToolbox {
	return &RealToolbox{
		tools:        make(map[string]interface{}),
		descriptions: []inference.ToolDescription{},
		reflector:    &jsonschema.Reflector{},
	}
}

// RegisterTool registers a tool function with the toolbox
// The function parameter should be a Go function that will be called when the tool is executed
func (rt *RealToolbox) RegisterTool(name string, function interface{}) error {
	// Validate that function is actually a function
	funcType := reflect.TypeOf(function)
	if funcType.Kind() != reflect.Func {
		return errors.Errorf("tool %s is not a function", name)
	}

	// Generate JSON schema for the function parameters
	schema, err := helpers.GetFunctionParametersJsonSchema(rt.reflector, function)
	if err != nil {
		return errors.Wrapf(err, "failed to generate schema for tool %s", name)
	}

	// Convert schema to map for the tool description
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal schema for tool %s", name)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return errors.Wrapf(err, "failed to unmarshal schema for tool %s", name)
	}

	// Store the tool function
	rt.tools[name] = function

	// Create tool description
	description := inference.ToolDescription{
		Name:        name,
		Description: schema.Description,
		Parameters:  schemaMap,
	}

	// Check if this tool already exists and update it, otherwise append
	found := false
	for i, existing := range rt.descriptions {
		if existing.Name == name {
			rt.descriptions[i] = description
			found = true
			break
		}
	}
	if !found {
		rt.descriptions = append(rt.descriptions, description)
	}

	return nil
}

// ExecuteTool executes a tool with the given name and arguments
func (rt *RealToolbox) ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	// Find the tool function
	tool, exists := rt.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %q not found", name)
	}

	// Convert arguments to the format expected by helpers.CallFunctionFromJson
	convertedArgs, err := rt.convertArgumentsForFunction(tool, arguments)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert arguments for tool %s", name)
	}

	// Call the function using the helpers function
	results, err := helpers.CallFunctionFromJson(tool, convertedArgs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute tool %s", name)
	}

	// Handle the results
	if len(results) == 0 {
		return nil, nil
	}

	if len(results) == 1 {
		return results[0].Interface(), nil
	}

	// Multiple return values - return as slice
	var resultValues []interface{}
	for _, result := range results {
		resultValues = append(resultValues, result.Interface())
	}
	return resultValues, nil
}

// convertArgumentsForFunction converts map arguments to the format expected by CallFunctionFromJson
func (rt *RealToolbox) convertArgumentsForFunction(function interface{}, arguments map[string]interface{}) (interface{}, error) {
	funcType := reflect.TypeOf(function)
	
	// For single parameter functions
	if funcType.NumIn() == 1 {
		// If we only have one parameter, we need to figure out what it should be
		paramType := funcType.In(0)
		
		// If the parameter is a struct, we marshal the arguments map and return it 
		// so it can be unmarshaled into the struct
		if paramType.Kind() == reflect.Struct {
			return arguments, nil
		}
		
		// If it's a simple type, look for a single argument value
		// For simple types, we expect the arguments map to have one key-value pair
		if len(arguments) == 1 {
			for _, v := range arguments {
				return v, nil
			}
		}
		
		// If no single value found, return the map itself
		return arguments, nil
	}
	
	// For multiple parameter functions, we need to convert to an ordered slice
	// This is tricky because we need to know the parameter names
	// For now, we'll assume the arguments are provided in a way that can be converted to a slice
	// This might need refinement based on actual usage patterns
	
	// Try to extract values in some reasonable order
	values := make([]interface{}, 0, len(arguments))
	for _, v := range arguments {
		values = append(values, v)
	}
	
	return values, nil
}

// GetToolDescriptions returns descriptions of all available tools
func (rt *RealToolbox) GetToolDescriptions() []inference.ToolDescription {
	return rt.descriptions
}

// GetToolNames returns the names of all registered tools
func (rt *RealToolbox) GetToolNames() []string {
	names := make([]string, 0, len(rt.tools))
	for name := range rt.tools {
		names = append(names, name)
	}
	return names
}

// HasTool returns true if the tool with the given name is registered
func (rt *RealToolbox) HasTool(name string) bool {
	_, exists := rt.tools[name]
	return exists
}

// Ensure RealToolbox implements Toolbox interface
var _ inference.Toolbox = (*RealToolbox)(nil)
