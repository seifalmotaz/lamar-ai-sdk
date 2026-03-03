// Package tool provides type-safe tool definitions for AI function calling.
//
// Tools allow AI models to request execution of external functions with structured
// inputs and outputs. The tool package provides a type-safe way to define tools
// using Go structs with jsonschema tags for automatic schema generation.
//
// Example:
//
//	type WeatherInput struct {
//	    Location string `json:"location" jsonschema:"required,description=City name"`
//	    Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit"`
//	}
//
//	type WeatherOutput struct {
//	    Temperature float64 `json:"temperature"`
//	    Condition   string   `json:"condition"`
//	}
//
//	weatherTool := tool.NewTool("get_weather", "Get current weather",
//	    func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
//	        return WeatherOutput{Temperature: 22.5, Condition: "sunny"}, nil
//	    },
//	)
package tool
