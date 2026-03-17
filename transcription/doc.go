package transcription

import "github.com/seifalmotaz/lamar-ai-sdk/provider"

/*
Package transcription provides a high-level API for audio-to-text transcription with AI models.

The transcription package wraps provider.TranscriptionModel implementations with:

  - Functional options pattern for configuration
  - Input validation with fail-fast semantics
  - Default timeouts (120 seconds)
  - Logging and metrics support

# Basic Usage

	package main

	import (
	    "context"
	    "fmt"
	    "os"

	    "github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	    "github.com/seifalmotaz/lamar-ai-sdk/transcription"
	)

	func main() {
	    ctx := context.Background()
	    client := openai.NewProvider(openai.APIKey(os.Getenv("OPENAI_API_KEY")))
	    model := client.Whisper1()

	    audioData, err := os.ReadFile("recording.mp3")
	    if err != nil {
	        panic(err)
	    }

	    result, err := transcription.Transcribe(ctx, model, audioData, "audio/mp3")
	    if err != nil {
	        panic(err)
	    }

	    fmt.Printf("Transcription: %s\n", result.Text)
	    if result.Language != "" {
	        fmt.Printf("Language: %s\n", result.Language)
	    }
	}

# With Language Hint

Specify the language to improve transcription accuracy:

	result, err := transcription.Transcribe(ctx, model, audioData, "audio/mp3",
	    transcription.WithLanguage("en"),
	)

# With Prompt Context

Provide context to improve transcription accuracy for specific terminology:

	result, err := transcription.Transcribe(ctx, model, audioData, "audio/mp3",
	    transcription.WithPrompt("A conversation about machine learning and AI"),
	)

# Custom Timeout

	result, err := transcription.Transcribe(ctx, model, audioData, "audio/mp3",
	    transcription.WithTimeout(180*time.Second),
	)

# Supported Audio Formats

OpenAI Whisper supports various audio formats:
  - MP3 (audio/mp3, audio/mpeg)
  - MP4 (audio/mp4, audio/m4a)
  - WAV (audio/wav)
  - WebM (audio/webm)
  - OGG (audio/ogg)

# Segments

The result may include timestamped segments:

	if len(result.Segments) > 0 {
	    for i, seg := range result.Segments {
	        fmt.Printf("[%d] %.2f-%.2f: %s\n", i, seg.StartSecond, seg.EndSecond, seg.Text)
	    }
	}

# No Timeout (context deadline only)

	result, err := transcription.Transcribe(ctx, model, audioData, "audio/mp3",
	    transcription.WithNoTimeout(),
	)
*/
type _ = provider.TranscriptionModel
