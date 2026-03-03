package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type TestInput struct {
	Name string `json:"name" jsonschema:"required,description=The name"`
	Age  int    `json:"age" jsonschema:"required,minimum=0"`
}

type TestOutput struct {
	Greeting string `json:"greeting"`
}

func TestNewTool(t *testing.T) {
	tool := NewTool("test_tool", "A test tool", func(ctx context.Context, input TestInput) (TestOutput, error) {
		return TestOutput{Greeting: "Hello " + input.Name}, nil
	})

	if tool.Name() != "test_tool" {
		t.Errorf("Expected name %q, got %q", "test_tool", tool.Name())
	}

	if tool.Description() != "A test tool" {
		t.Errorf("Expected description %q, got %q", "A test tool", tool.Description())
	}

	// Verify schema
	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema(), &schema); err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties in schema")
	}

	if len(props) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(props))
	}

	if _, ok := props["name"]; !ok {
		t.Error("Expected 'name' property")
	}

	if _, ok := props["age"]; !ok {
		t.Error("Expected 'age' property")
	}
}

func TestTypedTool_Execute(t *testing.T) {
	tool := NewTool("greet", "Greet someone", func(ctx context.Context, input TestInput) (TestOutput, error) {
		return TestOutput{Greeting: "Hello " + input.Name}, nil
	})

	input := TestInput{Name: "World", Age: 42}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(context.Background(), inputJSON)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var output TestOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	if output.Greeting != "Hello World" {
		t.Errorf("Expected greeting %q, got %q", "Hello World", output.Greeting)
	}
}

func TestTypedTool_ExecuteInvalidJSON(t *testing.T) {
	tool := NewTool("test", "Test tool", func(ctx context.Context, input TestInput) (TestOutput, error) {
		return TestOutput{}, nil
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{invalid json}`))
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	var parseErr *provider.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("Expected ParseError, got %T", err)
	}
}

func TestToDefinition(t *testing.T) {
	tool := NewTool("test", "Test tool", func(ctx context.Context, input TestInput) (TestOutput, error) {
		return TestOutput{}, nil
	})

	def := ToDefinition(tool)

	if def.Name != "test" {
		t.Errorf("Expected name %q, got %q", "test", def.Name)
	}

	if def.Description != "Test tool" {
		t.Errorf("Expected description %q, got %q", "Test tool", def.Description)
	}

	if def.InputSchema == nil {
		t.Error("Expected non-nil InputSchema")
	}
}

func TestToDefinitions(t *testing.T) {
	tool1 := NewTool("tool1", "First tool", func(ctx context.Context, input TestInput) (TestOutput, error) {
		return TestOutput{}, nil
	})
	tool2 := NewTool("tool2", "Second tool", func(ctx context.Context, input TestInput) (TestOutput, error) {
		return TestOutput{}, nil
	})

	defs := ToDefinitions(tool1, tool2)

	if len(defs) != 2 {
		t.Fatalf("Expected 2 definitions, got %d", len(defs))
	}

	if defs[0].Name != "tool1" {
		t.Errorf("Expected name %q, got %q", "tool1", defs[0].Name)
	}

	if defs[1].Name != "tool2" {
		t.Errorf("Expected name %q, got %q", "tool2", defs[1].Name)
	}
}

func TestTypedTool_WithError(t *testing.T) {
	expectedErr := &provider.Error{Code: provider.CodeInvalidInput, Message: "test error"}
	tool := NewTool("fail", "Fail tool", func(ctx context.Context, input TestInput) (TestOutput, error) {
		return TestOutput{}, expectedErr
	})

	input := TestInput{Name: "test", Age: 0}
	inputJSON, _ := json.Marshal(input)

	_, err := tool.Execute(context.Background(), inputJSON)
	if err == nil {
		t.Fatal("Expected error")
	}

	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("Expected provider.Error, got %T", err)
	}
	_ = providerErr
}
