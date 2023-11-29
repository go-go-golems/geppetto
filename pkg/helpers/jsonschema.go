package helpers

import (
	"encoding/json"
	"fmt"
	"github.com/invopop/jsonschema"
	"reflect"
)

// Callable is a type representing any callable function
type Callable interface{}

// CallFunctionFromJson calls a function with arguments provided as JSON
func CallFunctionFromJson(f Callable, jsonArgs interface{}) ([]reflect.Value, error) {
	funcVal := reflect.ValueOf(f)
	funcType := funcVal.Type()

	// Check if the function is indeed callable
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided callable is not a function")
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
func GetFunctionParametersJsonSchema(f Callable) (*jsonschema.Schema, error) {
	// Custom reflector configuration
	reflector := &jsonschema.Reflector{
		DoNotReference: true,
	}

	// Get the type of the function
	funcVal := reflect.ValueOf(f)
	funcType := funcVal.Type()

	// Check if the function is indeed callable
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided callable is not a function")
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
