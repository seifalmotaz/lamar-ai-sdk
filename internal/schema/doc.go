// Package schema provides JSON Schema extraction from Go struct types.
//
// This package uses the invopop/jsonschema library to automatically generate
// JSON schemas from Go struct types using struct tags. This enables type-safe
// tool definitions and structured output without hand-writing schemas.
//
// The primary use case is for generating input schemas for AI tools and
// response schemas for structured outputs.
//
// Example:
//
//	type WeatherInput struct {
//	    Location string `json:"location" jsonschema:"required,description=City name"`
//	    Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit"`
//	}
//
//	schema := schema.FromStruct[WeatherInput]()
package schema
