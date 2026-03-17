package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	"github.com/seifalmotaz/lamar-ai-sdk/speech"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	text := "Hello! This is a demonstration of the OpenAI text-to-speech API using the Lamar SDK."
	if len(os.Args) > 1 {
		text = os.Args[1]
	}

	outputFile := "output.mp3"
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TTS1HD()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Speech Synthesis")
	fmt.Println("----------------")
	fmt.Printf("Text: %s\n", text)
	fmt.Printf("Voice: nova\n")
	fmt.Printf("Model: tts-1-hd\n")
	fmt.Println()

	result, err := speech.Synthesize(ctx, model, text,
		speech.WithVoice("nova"),
		speech.WithFormat("mp3"),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, result.Audio, 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Audio saved to: %s\n", outputFile)
	fmt.Printf("Size: %d bytes\n", len(result.Audio))
	fmt.Printf("Media type: %s\n", result.MediaType)
	fmt.Println()
	fmt.Printf("You can play it with: afplay %s (macOS) or mpg123 %s (Linux)\n", outputFile, outputFile)
}
