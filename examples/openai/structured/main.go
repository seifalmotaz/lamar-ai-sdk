// Package main demonstrates structured output with OpenAI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-sdk/generate"
	"github.com/seifalmotaz/lamar-sdk/providers/openai"
	"github.com/seifalmotaz/lamar-sdk/stream"
)

type Person struct {
	Name string `json:"name" jsonschema:"required,description=The person's full name"`
	Age  int    `json:"age" jsonschema:"required,minimum=0,maximum=150,description=Age in years"`
	City string `json:"city,omitempty" jsonschema:"description=City of residence"`
	Job  string `json:"job,omitempty" jsonschema:"description=Occupation"`
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	// GPT5Mini returns LanguageModel which supports both generate and stream
	model := client.GPT5Mini()

	// Non-streaming structured output
	fmt.Println("GenerateObject example:")
	fmt.Println("-----------------------")

	result, err := generate.GenerateObject[Person](context.Background(), model,
		"Generate a random fictional software developer from San Francisco.",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Name: %s\n", result.Object.Name)
	fmt.Printf("Age: %d\n", result.Object.Age)
	fmt.Printf("City: %s\n", result.Object.City)
	fmt.Printf("Job: %s\n", result.Object.Job)
	fmt.Printf("Tokens: %d\n", result.Usage.TotalTokens)

	// Streaming structured output
	fmt.Println("\nStreamObject example:")
	fmt.Println("----------------------")

	streamResult := stream.StreamObject[Person](context.Background(), model,
		"Generate a random fictional teacher from New York.",
	)

	// Consume stream parts
	for part := range streamResult.Stream() {
		switch part.Type {
		case "text-delta":
			fmt.Print(part.Delta)
		case "object":
			fmt.Printf("\nPartial object: %+v\n", part.Object)
		case "finish":
			fmt.Println("\n(Stream complete)")
		}
	}

	// Get final object
	obj, err := streamResult.Object()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nFinal object:\n")
	fmt.Printf("  Name: %s\n", obj.Name)
	fmt.Printf("  Age: %d\n", obj.Age)
	fmt.Printf("  City: %s\n", obj.City)
	fmt.Printf("  Job: %s\n", obj.Job)
}
