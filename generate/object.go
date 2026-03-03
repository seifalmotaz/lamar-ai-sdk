package generate

import (
	"context"
	"encoding/json"
	"time"

	"github.com/seifalmotaz/lamar-sdk/internal/schema"
	"github.com/seifalmotaz/lamar-sdk/provider"
)

// ObjectResult contains the result of a structured object generation.
type ObjectResult[T any] struct {
	Object       T
	FinishReason provider.FinishReason
	Usage        provider.Usage
}

// GenerateObject generates a structured object from a model using JSON schema.
// It extracts the schema from type T using jsonschema tags and configures
// the model to respond with structured JSON output.
//
// Example:
//
//	type Person struct {
//	    Name string `json:"name" jsonschema:"required,description=The person's name"`
//	    Age  int    `json:"age" jsonschema:"required,minimum=0"`
//	}
//
//	result, err := generate.GenerateObject[Person](ctx, model, "Generate a random person")
func GenerateObject[T any](ctx context.Context, model provider.Generator, prompt string, opts ...Option) (*ObjectResult[T], error) {
	if model == nil {
		return nil, provider.ErrInvalidModel
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if prompt == "" && len(cfg.Messages) == 0 {
		return nil, provider.ErrInvalidPrompt
	}

	// Extract schema from type T
	sch := schema.FromStruct[T]()
	schemaBytes, err := json.Marshal(sch)
	if err != nil {
		return nil, &provider.ParseError{Field: "schema", Err: err}
	}

	// Build request with JSON schema response format
	req := &provider.GenerateRequest{
		Prompt:   prompt,
		Messages: cfg.Messages,
		System:   cfg.System,
		Config: provider.Config{
			MaxTokens:     cfg.MaxTokens,
			Temperature:   cfg.Temperature,
			TopP:          cfg.TopP,
			TopK:          cfg.TopK,
			StopSequences: cfg.StopSequences,
			Seed:          cfg.Seed,
			ResponseFormat: &provider.ResponseFormat{
				Type:       "json_schema",
				JSONSchema: json.RawMessage(schemaBytes),
			},
		},
	}

	// Apply timeout
	genCtx := ctx
	if cfg.Timeout != nil {
		if *cfg.Timeout > 0 {
			var cancel context.CancelFunc
			genCtx, cancel = context.WithTimeout(ctx, *cfg.Timeout)
			defer cancel()
		}
		// *Timeout == 0 means no timeout
	}

	start := time.Now()
	cfg.logDebug("generate_object: starting", "provider", model.Provider(), "model", model.ModelID())

	result, err := model.Generate(genCtx, req)
	if err != nil {
		cfg.logError("generate_object: failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
		cfg.recordMetrics(ctx, model, time.Since(start), err)
		return nil, err
	}

	cfg.logDebug("generate_object: completed", "provider", model.Provider(), "model", model.ModelID(), "tokens", result.Usage.TotalTokens)
	cfg.recordMetrics(ctx, model, time.Since(start), nil)
	cfg.recordTokens(ctx, model, result.Usage)

	// Parse the JSON response into type T
	var obj T
	if err := json.Unmarshal([]byte(result.Text), &obj); err != nil {
		return nil, &provider.ParseError{Field: "object", Err: err}
	}

	return &ObjectResult[T]{
		Object:       obj,
		FinishReason: result.FinishReason,
		Usage:        result.Usage,
	}, nil
}
