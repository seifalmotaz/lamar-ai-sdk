package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/seifalmotaz/lamar-sdk/providers/openai"
	"github.com/seifalmotaz/lamar-sdk/transcription"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <audio-file>")
		fmt.Println("Example: go run main.go recording.mp3")
		os.Exit(1)
	}

	audioFile := os.Args[1]
	audioData, err := os.ReadFile(audioFile)
	if err != nil {
		fmt.Printf("Error reading audio file: %v\n", err)
		os.Exit(1)
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.Whisper1()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	mediaType := detectMediaType(audioFile)
	result, err := transcription.Transcribe(ctx, model, audioData, mediaType)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Transcription Result")
	fmt.Println("--------------------")
	fmt.Printf("Text: %s\n", result.Text)
	if result.Language != "" {
		fmt.Printf("Language: %s\n", result.Language)
	}
	if result.Duration > 0 {
		fmt.Printf("Duration: %.2f seconds\n", result.Duration)
	}
	if len(result.Segments) > 0 {
		fmt.Printf("\nSegments (%d):\n", len(result.Segments))
		for i, seg := range result.Segments {
			fmt.Printf("  [%d] %.2f-%.2f: %s\n", i+1, seg.StartSecond, seg.EndSecond, seg.Text)
		}
	}
}

func detectMediaType(filename string) string {
	if len(filename) < 4 {
		return "audio/mp3"
	}
	ext := filename[len(filename)-4:]
	switch ext {
	case ".mp3":
		return "audio/mp3"
	case ".mp4":
		return "audio/mp4"
	case ".m4a":
		return "audio/m4a"
	case ".wav":
		return "audio/wav"
	case ".webm":
		return "audio/webm"
	case ".oga", ".ogg":
		return "audio/ogg"
	default:
		return "audio/mp3"
	}
}
