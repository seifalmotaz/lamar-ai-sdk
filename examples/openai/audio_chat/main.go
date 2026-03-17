// Package main demonstrates audio content in chat with GPT-4o.
// This example shows how to send audio input along with text in a message.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	// Read audio file
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <audio_file>")
		fmt.Println("Example: go run main.go sample.mp3")
		fmt.Println("\nSupported formats: wav, mp3, webm, m4a, etc.")
		os.Exit(1)
	}

	audioPath := os.Args[1]
	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		fmt.Printf("Error reading audio file: %v\n", err)
		os.Exit(1)
	}

	// Determine media type from file extension
	mediaType := detectMediaType(audioPath)
	fmt.Printf("Audio file: %s (%s, %d bytes)\n", audioPath, mediaType, len(audioData))

	// Create provider and model
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4oAudioPreview()

	// Create message with audio and text content
	messages := []provider.Message{
		provider.UserMessageWithContent(
			provider.Audio(audioData, mediaType),
			provider.Text("What is being said in this audio? Please provide a detailed transcription."),
		),
	}

	fmt.Println("\nSending audio to GPT-4o-audio-preview...")
	fmt.Println("-------------------------")

	// Generate response
	result, err := generate.Generate(context.Background(), model, "",
		generate.Messages(messages...),
		generate.MaxTokens(500),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Response:")
	fmt.Println(result.Text)
	fmt.Println("-------------------------")
	usage := result.Usage()
	fmt.Printf("Tokens: %d prompt + %d completion = %d total\n",
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens)
}

// detectMediaType returns the media type based on file extension.
func detectMediaType(path string) string {
	// Find the last dot in the path
	lastDot := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			lastDot = i
			break
		}
		if path[i] == '/' {
			break // Stop at directory separator
		}
	}

	if lastDot == -1 || lastDot == len(path)-1 {
		return "audio/wav" // Default fallback
	}

	ext := path[lastDot+1:]

	switch ext {
	case "wav":
		return "audio/wav"
	case "mp3":
		return "audio/mp3"
	case "webm":
		return "audio/webm"
	case "m4a":
		return "audio/m4a"
	case "mp4":
		return "audio/mp4"
	case "mpeg":
		return "audio/mpeg"
	case "oga":
		return "audio/oga"
	default:
		return "audio/wav" // Default fallback
	}
}
