package helpers

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
)

// Callable is a type representing any callable function
type Callable interface{}

// CallFunctionFromJson calls a function with arguments provided as JSON
// TODO(manuel, 2024-01-12) make this take a ctx
func CallFunctionFromJson(f Callable, jsonArgs interface{}) ([]reflect.Value, error) {
	funcVal := reflect.ValueOf(f)
	funcType := funcVal.Type()

	// Check if the function is indeed callable
	if funcType.Kind() != reflect.Func {
		return nil, errors.Errorf("provided callable is not a function")
	}

	// Marshal the provided arguments back to JSON to work with individual arguments
	argsJson, err := json.Marshal(jsonArgs)
	if err != nil {
		return nil, err
	}

	// Prepare to unmarshal arguments into a slice of reflect.Value
	var args []reflect.Value

	// If there's only one argument, handle it separately
	if funcType.NumIn() == 1 {
		argType := funcType.In(0)
		argPtr := reflect.New(argType)

		if err := json.Unmarshal(argsJson, argPtr.Interface()); err != nil {
			return nil, err
		}

		args = append(args, argPtr.Elem())
	} else {
		// Unmarshal JSON into a slice of interfaces
		var rawArgs []interface{}
		if err := json.Unmarshal(argsJson, &rawArgs); err != nil {
			return nil, err
		}

		// Convert each argument to reflect.Value
		for i, rawArg := range rawArgs {
			argType := funcType.In(i)
			argValue := reflect.New(argType).Elem()

			argJson, err := json.Marshal(rawArg)
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(argJson, argValue.Addr().Interface()); err != nil {
				return nil, err
			}

			args = append(args, argValue)
		}
	}

	// Call the function with the prepared arguments
	return funcVal.Call(args), nil
}

// GetFunctionParametersJsonSchema generates a JSON Schema for the arguments of the given function
func GetFunctionParametersJsonSchema(reflector *jsonschema.Reflector, f Callable) (*jsonschema.Schema, error) {
	// Get the type of the function
	funcVal := reflect.ValueOf(f)
	funcType := funcVal.Type()

	// Check if the function is indeed callable
	if funcType.Kind() != reflect.Func {
		return nil, errors.Errorf("provided callable is not a function")
	}

	// Handle the case of a single parameter separately
	if funcType.NumIn() == 1 {
		singleParamType := funcType.In(0)
		singleParamInstance := reflect.New(singleParamType).Elem().Interface()
		return reflector.Reflect(singleParamInstance), nil
	}

	// Prepare a schema for multiple function parameters
	schema := &jsonschema.Schema{
		Type:  "array",
		Items: &jsonschema.Schema{},
	}

	// Create a slice to hold schemas for each parameter
	paramSchemas := make([]*jsonschema.Schema, 0, funcType.NumIn())

	// Loop over the function's input parameters
	for i := 0; i < funcType.NumIn(); i++ {
		paramType := funcType.In(i)
		paramInstance := reflect.New(paramType).Elem().Interface()
		paramSchema := reflector.Reflect(paramInstance)
		paramSchemas = append(paramSchemas, paramSchema)
	}

	// Use PrefixItems to define schemas for each parameter in the array
	schema.PrefixItems = paramSchemas

	return schema, nil
}

// SimplifiedJsonSchemaProperty represents a simplified property in the JSON Schema
type SimplifiedJsonSchemaProperty struct {
	Type        string                                   `json:"type"`
	Description string                                   `json:"description,omitempty"`
	Required    bool                                     `json:"-"`
	Properties  map[string]*SimplifiedJsonSchemaProperty `json:"properties,omitempty"`
	Items       *SimplifiedJsonSchemaProperty            `json:"items,omitempty"`
}

// SimplifiedJsonSchema represents a simplified root JSON Schema
type SimplifiedJsonSchema struct {
	Type        string                                   `json:"type"`
	Description string                                   `json:"description,omitempty"`
	Properties  map[string]*SimplifiedJsonSchemaProperty `json:"properties"`
	Required    []string                                 `json:"required,omitempty"`
}

// getParameterName attempts to get the parameter name from debug information
// Returns an empty string if the name cannot be determined
func getParameterName(f Callable, index int) string {
	t := reflect.TypeOf(f)
	if t.Kind() != reflect.Func {
		return ""
	}
	return t.In(index).Name()
}

// GetSimplifiedFunctionParametersJsonSchema generates a simplified JSON Schema for the arguments of the given function
func GetSimplifiedFunctionParametersJsonSchema(reflector *jsonschema.Reflector, f Callable) (*SimplifiedJsonSchema, error) {
	// Get the type of the function
	funcVal := reflect.ValueOf(f)
	funcType := funcVal.Type()

	// Check if the function is indeed callable
	if funcType.Kind() != reflect.Func {
		return nil, errors.Errorf("provided callable is not a function")
	}

	// Create the root schema
	schema := &SimplifiedJsonSchema{
		Type:       "object",
		Properties: make(map[string]*SimplifiedJsonSchemaProperty),
		Required:   []string{},
	}

	// Handle the case of a single parameter
	if funcType.NumIn() == 1 {
		singleParamType := funcType.In(0)
		fullSchema := reflector.Reflect(reflect.New(singleParamType).Elem().Interface())

		// Convert the full schema to our simplified version
		for i := fullSchema.Properties.Oldest(); i != nil; i = i.Next() {
			name, prop := i.Key, i.Value
			simplified := &SimplifiedJsonSchemaProperty{
				Type:        string(prop.Type),
				Description: prop.Description,
			}

			// Check if this property is required
			for _, req := range fullSchema.Required {
				if req == name {
					schema.Required = append(schema.Required, name)
					break
				}
			}

			// Handle nested objects
			if prop.Type == "object" && prop.Properties != nil {
				simplified.Properties = make(map[string]*SimplifiedJsonSchemaProperty)
				for i := prop.Properties.Oldest(); i != nil; i = i.Next() {
					subName, subProp := i.Key, i.Value
					simplified.Properties[subName] = &SimplifiedJsonSchemaProperty{
						Type:        string(subProp.Type),
						Description: subProp.Description,
					}
				}
			}

			// Handle arrays
			if prop.Type == "array" && prop.Items != nil {
				simplified.Items = &SimplifiedJsonSchemaProperty{
					Type: string(prop.Items.Type),
				}
			}

			schema.Properties[name] = simplified
		}

		return schema, nil
	}

	// For multiple parameters, create a property for each parameter
	for i := 0; i < funcType.NumIn(); i++ {
		paramType := funcType.In(i)
		paramName := getParameterName(f, i)
		if paramName == "" {
			paramName = fmt.Sprintf("param%d", i)
		}

		fullSchema := reflector.Reflect(reflect.New(paramType).Elem().Interface())

		simplified := &SimplifiedJsonSchemaProperty{
			Type:        string(fullSchema.Type),
			Description: fullSchema.Description,
		}

		// Check if this parameter is required
		if len(fullSchema.Required) > 0 {
			schema.Required = append(schema.Required, paramName)
		}

		schema.Properties[paramName] = simplified
	}

	return schema, nil
}
