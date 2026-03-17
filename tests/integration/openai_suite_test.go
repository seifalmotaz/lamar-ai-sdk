package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/embed"
	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/internal/contract"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	"github.com/seifalmotaz/lamar-ai-sdk/stream"
)

func getTestOpenAIModels(t *testing.T) []contract.ModelWithCapabilities {
	t.Helper()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	client := openai.NewProvider(openai.APIKey(apiKey))

	return []contract.ModelWithCapabilities{
		contract.ModelWithCaps(
			client.GPT4o(),
			contract.CapTextGeneration,
			contract.CapObjectGeneration,
			contract.CapToolCalls,
			contract.CapVision,
		),
	}
}

func timeout() time.Duration {
	if timeout := os.Getenv("TEST_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			return d
		}
	}
	return 60 * time.Second
}

func TestOpenAI_FeatureSuite_Generate(t *testing.T) {
	models := getTestOpenAIModels(t)

	contract.RunGeneratorTests(t, models, map[string]contract.GeneratorTestFunc{
		"BasicGenerate": func(t *testing.T, ctx context.Context, model provider.Generator) {
			result, err := generate.Generate(ctx, model, "Say 'hello'",
				generate.MaxTokens(10),
			)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if result.Text() == "" {
				t.Error("Expected non-empty text")
			}

			if result.Usage().TotalTokens == 0 {
				t.Error("Expected non-zero token usage")
			}
		},
		"GenerateWithSystem": func(t *testing.T, ctx context.Context, model provider.Generator) {
			result, err := generate.Generate(ctx, model, "What is 2+2?",
				generate.System("You are a helpful math teacher. Be brief."),
				generate.MaxTokens(20),
			)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if result.Text() == "" {
				t.Error("Expected non-empty text")
			}
		},
		"GenerateWithTemperature": func(t *testing.T, ctx context.Context, model provider.Generator) {
			result, err := generate.Generate(ctx, model, "Say 'test'",
				generate.MaxTokens(5),
				generate.Temperature(0.1),
			)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if result.Text() == "" {
				t.Error("Expected non-empty text")
			}
		},
	})
}

func TestOpenAI_FeatureSuite_Stream(t *testing.T) {
	models := getTestOpenAIModels(t)

	contract.RunStreamerTests(t, models, map[string]contract.StreamerTestFunc{
		"BasicStream": func(t *testing.T, ctx context.Context, model provider.Streamer) {
			result := stream.Stream(ctx, model, "Count from 1 to 3",
				stream.MaxTokens(20),
			)

			partCount := contract.AssertStreamHasParts(t, result, 1)
			if partCount < 1 {
				t.Fatal("Expected at least 1 stream part")
			}

			text, err := result.Text()
			if err != nil {
				t.Fatalf("Text() failed: %v", err)
			}
			if text == "" {
				t.Error("Expected non-empty text")
			}
		},
		"StreamWithUsage": func(t *testing.T, ctx context.Context, model provider.Streamer) {
			result := stream.Stream(ctx, model, "Hello",
				stream.MaxTokens(10),
			)

			for range result.Stream() {
			}

			usage, err := result.Usage()
			if err != nil {
				t.Fatalf("Usage() failed: %v", err)
			}
			contract.RequirePositiveUsage(t, usage)
		},
	})
}

func TestOpenAI_FeatureSuite_Embedding(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	embedder := client.TextEmbedding3Small()

	models := []contract.ModelWithCapabilities{
		contract.ModelWithCaps(embedder, contract.CapEmbedding),
	}

	contract.RunEmbedderTests(t, models, map[string]contract.EmbedderTestFunc{
		"BasicEmbed": func(t *testing.T, ctx context.Context, model provider.EmbeddingModel) {
			result, err := embed.Embed(ctx, model, "Hello, world!")
			if err != nil {
				t.Fatalf("Embed failed: %v", err)
			}
			if len(result.Embedding) == 0 {
				t.Error("Expected non-empty embedding")
			}
			if result.Usage.TotalTokens == 0 {
				t.Error("Expected non-zero token usage")
			}
		},
		"EmbedBatch": func(t *testing.T, ctx context.Context, model provider.EmbeddingModel) {
			texts := []string{"Hello", "World", "Test"}
			result, err := embed.EmbedBatch(ctx, model, texts)
			if err != nil {
				t.Fatalf("EmbedBatch failed: %v", err)
			}
			if len(result.Embeddings) != len(texts) {
				t.Errorf("Expected %d embeddings, got %d", len(texts), len(result.Embeddings))
			}
			for i, emb := range result.Embeddings {
				if len(emb) == 0 {
					t.Errorf("Embedding %d is empty", i)
				}
			}
		},
	})
}

func TestOpenAI_FeatureSuite_ObjectGeneration(t *testing.T) {
	models := getTestOpenAIModels(t)

	tests := []contract.TestCase{
		{
			Name:         "GenerateObject_Basic",
			Capabilities: []contract.TestCapability{contract.CapObjectGeneration},
			Run: func(t *testing.T, m provider.Model) {
				generator, ok := m.(provider.Generator)
				if !ok {
					t.Skip("Model does not implement Generator")
				}

				ctx, cancel := context.WithTimeout(context.Background(), timeout())
				defer cancel()

				type Person struct {
					Name string `json:"name" jsonschema:"required"`
					Age  int    `json:"age" jsonschema:"required,minimum=0"`
				}

				result, err := generate.GenerateObject[Person](ctx, generator,
					"Generate a fictional person with name and age.",
					generate.MaxTokens(50),
				)
				if err != nil {
					t.Fatalf("GenerateObject failed: %v", err)
				}
				if result.Object.Name == "" {
					t.Error("Expected non-empty name")
				}
				if result.Object.Age < 0 {
					t.Error("Age should not be negative")
				}
			},
		},
	}

	contract.RunWithModels(t, models, tests)
}
