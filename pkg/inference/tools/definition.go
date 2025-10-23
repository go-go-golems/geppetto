package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/rs/zerolog/log"
)

// ToolDefinition represents a tool that can be called by AI models
type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  *jsonschema.Schema `json:"parameters"`
	Function    ToolFunc           `json:"-"` // Type-safe function wrapper
	Examples    []ToolExample      `json:"examples,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	Version     string             `json:"version,omitempty"` // For provider compatibility
}

// ToolFunc wraps the actual function with validation and fast execution
type ToolFunc struct {
	Fn          interface{}                                        `json:"-"` // The actual function
	executor    func([]byte) (interface{}, error)                  `json:"-"` // Pre-compiled executor (no context)
	executorCtx func(context.Context, []byte) (interface{}, error) `json:"-"` // Context-aware executor
	inputType   reflect.Type                                       `json:"-"` // Cached input type
	outputType  reflect.Type                                       `json:"-"` // Cached output type
}

// NewToolFromFunc creates a ToolDefinition from a Go function
func NewToolFromFunc(name, description string, fn interface{}) (*ToolDefinition, error) {
	funcType := reflect.TypeOf(fn)
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}

	// Validate function signature
	if funcType.NumOut() == 0 || funcType.NumOut() > 2 {
		return nil, fmt.Errorf("function must return (result) or (result, error)")
	}
	if funcType.NumOut() == 2 {
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		if !funcType.Out(1).Implements(errorType) {
			return nil, fmt.Errorf("second return value must be an error")
		}
	}

	// Generate JSON schema from function parameters (supports optional context.Context first param)
	schema, err := generateSchemaFromFunc(funcType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	// Create executors
	executor := createExecutor(fn, funcType)
	executorCtx := createExecutorWithContext(fn, funcType)

	// Determine input and output types, supporting optional context first param
	var inType reflect.Type
	switch funcType.NumIn() {
	case 0:
		inType = nil
	case 1:
		inType = funcType.In(0)
		if inType == reflect.TypeOf((*context.Context)(nil)).Elem() {
			inType = nil // no JSON input
		}
	default:
		// If first is context and second is struct, set inType to second
		if funcType.In(0) == reflect.TypeOf((*context.Context)(nil)).Elem() {
			inType = funcType.In(1)
		} else {
			inType = funcType.In(0)
		}
	}

	toolFunc := ToolFunc{
		Fn:          fn,
		executor:    executor,
		executorCtx: executorCtx,
		inputType:   inType,
		outputType:  funcType.Out(0),
	}

	return &ToolDefinition{
		Name:        name,
		Description: description,
		Parameters:  schema,
		Function:    toolFunc,
	}, nil
}

// Execute calls the tool function with the provided arguments
func (tf *ToolFunc) Execute(args []byte) (interface{}, error) {
	if tf.executor == nil {
		return nil, fmt.Errorf("tool function not properly initialized")
	}
	return tf.executor(args)
}

// ExecuteWithContext executes the tool function with context support when available.
func (tf *ToolFunc) ExecuteWithContext(ctx context.Context, args []byte) (interface{}, error) {
	if tf.executorCtx != nil {
		return tf.executorCtx(ctx, args)
	}
	return tf.Execute(args)
}

type ToolExample struct {
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output"`
	Description string                 `json:"description"`
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID       string        `json:"id"`
	Result   interface{}   `json:"result"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
	Retries  int           `json:"retries,omitempty"`
}

// ToolError represents an error that occurred during tool execution
type ToolError struct {
	ToolName string      `json:"tool_name"`
	ToolID   string      `json:"tool_id,omitempty"`
	Type     string      `json:"type"` // "validation", "execution", "timeout", "not_found"
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool error [%s]: %s", e.Type, e.Message)
}

// generateSchemaFromFunc creates a JSON schema from a function's parameters
func generateSchemaFromFunc(funcType reflect.Type) (*jsonschema.Schema, error) {
	if funcType.NumIn() == 0 {
		return &jsonschema.Schema{
			Type: "object",
		}, nil
	}

	// Support signatures:
	//   func(Input) (Result, error)
	//   func(context.Context, Input) (Result, error)
	var inputType reflect.Type
	switch funcType.NumIn() {
	case 1:
		if funcType.In(0) == reflect.TypeOf((*context.Context)(nil)).Elem() {
			// No JSON input; empty object
			return &jsonschema.Schema{Type: "object"}, nil
		}
		inputType = funcType.In(0)
	case 2:
		if funcType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
			return nil, fmt.Errorf("two-arg tool function must be (context.Context, Input)")
		}
		inputType = funcType.In(1)
	default:
		return nil, fmt.Errorf("function must take exactly one parameter (Input) or (context.Context, Input)")
	}

	// Create an instance of the type to properly generate schema with properties
	// This matches the approach used in the old GetFunctionParametersJsonSchema
	inputInstance := reflect.New(inputType).Elem().Interface()

	reflector := jsonschema.Reflector{
		// Expand definitions inline instead of using $refs
		DoNotReference: true,
	}
	schema := reflector.Reflect(inputInstance)

	// Ensure the root schema has type "object" for OpenAI compatibility
	if schema.Type == "" && schema.Ref == "" {
		schema.Type = "object"
	}

	return schema, nil
}

// createExecutor creates a pre-compiled executor for the function
func createExecutor(fn interface{}, funcType reflect.Type) func([]byte) (interface{}, error) {
	funcValue := reflect.ValueOf(fn)

	return func(args []byte) (interface{}, error) {
		log.Debug().
			Str("executor", "noctx").
			Int("num_in", funcType.NumIn()).
			Int("num_out", funcType.NumOut()).
			Str("func_type", funcType.String()).
			Int("args_len", len(args)).
			Str("args", string(args)).
			Msg("tools: createExecutor invoked")

		if funcType.NumIn() == 0 {
			// No parameters
			log.Debug().Msg("tools: calling function with no parameters")
			results := funcValue.Call([]reflect.Value{})
			log.Debug().
				Int("results_len", len(results)).
				Msg("tools: function call completed, extracting results")
			return extractResults(results)
		}

		// One parameter (could be context or input)
		if funcType.NumIn() == 1 {
			in := funcType.In(0)
			log.Debug().
				Str("param_type", in.String()).
				Bool("is_context", in == reflect.TypeOf((*context.Context)(nil)).Elem()).
				Msg("tools: handling single parameter function")

			if in == reflect.TypeOf((*context.Context)(nil)).Elem() {
				// Context-only inputless function: call with Background
				log.Debug().Msg("tools: calling context-only function with Background context")
				results := funcValue.Call([]reflect.Value{reflect.ValueOf(context.Background())})
				log.Debug().
					Int("results_len", len(results)).
					Msg("tools: context-only function call completed")
				return extractResults(results)
			}

			log.Debug().
				Str("input_type", in.String()).
				Msg("tools: creating input instance for unmarshaling")
			input := reflect.New(in).Interface()

			log.Debug().
				Str("input_ptr_type", reflect.TypeOf(input).String()).
				Msg("tools: unmarshaling JSON args into input")
			if err := json.Unmarshal(args, input); err != nil {
				log.Error().
					Err(err).
					Str("input_type", in.String()).
					Str("args", string(args)).
					Msg("tools: failed to unmarshal arguments")
				return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
			}

			log.Debug().
				Str("input_value_type", reflect.ValueOf(input).Elem().Type().String()).
				Msg("tools: calling function with unmarshaled input")
			results := funcValue.Call([]reflect.Value{reflect.ValueOf(input).Elem()})
			log.Debug().
				Int("results_len", len(results)).
				Msg("tools: single param function call completed")
			return extractResults(results)
		}

		// Fallback for unexpected signatures
		// Add diagnostic info
		inCount := funcType.NumIn()
		var inTypes []string
		for i := 0; i < inCount; i++ {
			inTypes = append(inTypes, funcType.In(i).String())
		}
		log.Error().
			Int("num_in", inCount).
			Strs("input_types", inTypes).
			Msg("tools: unsupported function signature")
		return nil, fmt.Errorf("unsupported tool function signature: numIn=%d, inTypes=%v", inCount, inTypes)
	}
}

// createExecutorWithContext returns an executor that passes context when supported by the function signature.
func createExecutorWithContext(fn interface{}, funcType reflect.Type) func(context.Context, []byte) (interface{}, error) {
	funcValue := reflect.ValueOf(fn)
	return func(ctx context.Context, args []byte) (interface{}, error) {
		log.Debug().
			Str("executor", "withctx").
			Int("num_in", funcType.NumIn()).
			Int("num_out", funcType.NumOut()).
			Str("func_type", funcType.String()).
			Int("args_len", len(args)).
			Str("args", string(args)).
			Msg("tools: createExecutorWithContext invoked")

		switch funcType.NumIn() {
		case 0:
			log.Debug().Msg("tools: calling function with no parameters (ignoring context)")
			results := funcValue.Call([]reflect.Value{})
			log.Debug().
				Int("results_len", len(results)).
				Msg("tools: no-param function call completed")
			return extractResults(results)
		case 1:
			in := funcType.In(0)
			log.Debug().
				Str("param_type", in.String()).
				Bool("is_context", in == reflect.TypeOf((*context.Context)(nil)).Elem()).
				Msg("tools: handling single parameter function with context")

			if in == reflect.TypeOf((*context.Context)(nil)).Elem() {
				log.Debug().Msg("tools: calling context-only function with provided context")
				results := funcValue.Call([]reflect.Value{reflect.ValueOf(ctx)})
				log.Debug().
					Int("results_len", len(results)).
					Msg("tools: context-only function call completed")
				return extractResults(results)
			}

			log.Debug().
				Str("input_type", in.String()).
				Msg("tools: creating input instance for unmarshaling (no context passed)")
			input := reflect.New(in).Interface()

			log.Debug().
				Str("input_ptr_type", reflect.TypeOf(input).String()).
				Msg("tools: unmarshaling JSON args into input")
			if err := json.Unmarshal(args, input); err != nil {
				log.Error().
					Err(err).
					Str("input_type", in.String()).
					Str("args", string(args)).
					Msg("tools: failed to unmarshal arguments")
				return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
			}

			log.Debug().
				Str("input_value_type", reflect.ValueOf(input).Elem().Type().String()).
				Msg("tools: calling function with unmarshaled input (no context)")
			results := funcValue.Call([]reflect.Value{reflect.ValueOf(input).Elem()})
			log.Debug().
				Int("results_len", len(results)).
				Msg("tools: single param function call completed")
			return extractResults(results)
		case 2:
			// Expect (context.Context, Input)
			log.Debug().
				Str("param0_type", funcType.In(0).String()).
				Str("param1_type", funcType.In(1).String()).
				Bool("param0_is_context", funcType.In(0) == reflect.TypeOf((*context.Context)(nil)).Elem()).
				Msg("tools: handling two-parameter function")

			if funcType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
				log.Error().
					Str("param0_type", funcType.In(0).String()).
					Msg("tools: first parameter is not context.Context")
				return nil, fmt.Errorf("unsupported two-arg tool function signature")
			}

			in := funcType.In(1)
			log.Debug().
				Str("input_type", in.String()).
				Msg("tools: creating input instance for two-param function")
			input := reflect.New(in).Interface()

			log.Debug().
				Str("input_ptr_type", reflect.TypeOf(input).String()).
				Msg("tools: unmarshaling JSON args for two-param function")
			if err := json.Unmarshal(args, input); err != nil {
				log.Error().
					Err(err).
					Str("input_type", in.String()).
					Str("args", string(args)).
					Msg("tools: failed to unmarshal arguments for two-param function")
				return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
			}

			log.Debug().
				Str("input_value_type", reflect.ValueOf(input).Elem().Type().String()).
				Msg("tools: calling two-param function with context and input")
			results := funcValue.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(input).Elem()})
			log.Debug().
				Int("results_len", len(results)).
				Msg("tools: two-param function call completed")
			return extractResults(results)
		default:
			inCount := funcType.NumIn()
			var inTypes []string
			for i := 0; i < inCount; i++ {
				inTypes = append(inTypes, funcType.In(i).String())
			}
			log.Error().
				Int("num_in", inCount).
				Strs("input_types", inTypes).
				Msg("tools: unsupported function signature with context")
			return nil, fmt.Errorf("unsupported tool function signature: numIn=%d, inTypes=%v", inCount, inTypes)
		}
	}
}

// extractResults extracts the result and error from function call results
func extractResults(results []reflect.Value) (interface{}, error) {
	log.Debug().
		Int("results_count", len(results)).
		Msg("tools: extractResults called")

	if len(results) == 1 {
		result := results[0].Interface()
		log.Debug().
			Str("result_type", reflect.TypeOf(result).String()).
			Msg("tools: returning single result")
		return result, nil
	}

	if len(results) == 2 {
		result := results[0].Interface()
		errInterface := results[1].Interface()

		log.Debug().
			Str("result_type", reflect.TypeOf(result).String()).
			Bool("error_is_nil", errInterface == nil).
			Msg("tools: processing two results")

		if errInterface == nil {
			log.Debug().Msg("tools: no error, returning result")
			return result, nil
		}

		if err, ok := errInterface.(error); ok {
			log.Debug().
				Err(err).
				Msg("tools: returning result with error")
			return result, err
		}

		log.Error().
			Str("error_type", reflect.TypeOf(errInterface).String()).
			Interface("error_value", errInterface).
			Msg("tools: unexpected error type")
		return result, fmt.Errorf("unexpected error type: %T", errInterface)
	}

	log.Error().
		Int("results_count", len(results)).
		Msg("tools: unexpected number of return values")
	return nil, fmt.Errorf("unexpected number of return values: %d", len(results))
}
