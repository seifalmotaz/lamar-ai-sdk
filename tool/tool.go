package tool

import (
	"context"
	"encoding/json"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/schema"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// Tool represents a callable tool that can be used by AI models.
type Tool interface {
	// Name returns the tool's unique identifier.
	Name() string

	// Description returns a human-readable description of what the tool does.
	Description() string

	// InputSchema returns the JSON Schema for the tool's input parameters.
	InputSchema() json.RawMessage

	// Execute invokes the tool with the given input and returns the result.
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

// typedTool is a type-safe implementation of Tool.
type typedTool[In, Out any] struct {
	name        string
	description string
	inputSchema json.RawMessage
	fn          func(ctx context.Context, input In) (Out, error)
}

// NewTool creates a type-safe tool from a function.
// The input type In must be a struct with json and jsonschema tags for schema generation.
//
// Example:
//
//	weatherTool := tool.NewTool[WeatherInput, WeatherOutput](
//	    "get_weather",
//	    "Get current weather for a location",
//	    func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
//	        return WeatherOutput{Temperature: 22.5, Condition: "sunny"}, nil
//	    },
//	)
func NewTool[In, Out any](name, description string, fn func(ctx context.Context, input In) (Out, error)) Tool {
	sch := schema.FromStruct[In]()
	schemaBytes, err := json.Marshal(sch)
	if err != nil {
		schemaBytes = json.RawMessage("{}")
	}

	return &typedTool[In, Out]{
		name:        name,
		description: description,
		inputSchema: json.RawMessage(schemaBytes),
		fn:          fn,
	}
}

func (t *typedTool[In, Out]) Name() string {
	return t.name
}

func (t *typedTool[In, Out]) Description() string {
	return t.description
}

func (t *typedTool[In, Out]) InputSchema() json.RawMessage {
	return t.inputSchema
}

func (t *typedTool[In, Out]) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var in In
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, &provider.ParseError{Field: "input", Err: err}
	}

	out, err := t.fn(ctx, in)
	if err != nil {
		return nil, err
	}

	result, err := json.Marshal(out)
	if err != nil {
		return nil, &provider.ParseError{Field: "output", Err: err}
	}

	return json.RawMessage(result), nil
}

// ToDefinition converts a Tool to provider.ToolDefinition for use in generate options.
func ToDefinition(t Tool) provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		InputSchema: t.InputSchema(),
	}
}

// ToDefinitions converts multiple Tools to []provider.ToolDefinition.
func ToDefinitions(tools ...Tool) []provider.ToolDefinition {
	defs := make([]provider.ToolDefinition, len(tools))
	for i, t := range tools {
		defs[i] = ToDefinition(t)
	}
	return defs
}
