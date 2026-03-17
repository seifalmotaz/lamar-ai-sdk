package speech

import "github.com/seifalmotaz/lamar-ai-sdk/provider"

/*
Package speech provides a high-level API for text-to-speech synthesis with AI models.

The speech package wraps provider.SpeechModel implementations with:

  - Functional options pattern for configuration
  - Input validation with fail-fast semantics
  - Default timeouts (60 seconds)
  - Logging and metrics support

# Basic Usage

	package main

	import (
	    "context"
	    "fmt"
	    "os"

	    "github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	    "github.com/seifalmotaz/lamar-ai-sdk/speech"
	)

	func main() {
	    ctx := context.Background()
	    client := openai.NewProvider(openai.APIKey(os.Getenv("OPENAI_API_KEY")))
	    model := client.TTS1()

	    result, err := speech.Synthesize(ctx, model, "Hello, world!",
	        speech.WithVoice("alloy"),
	        speech.WithFormat("mp3"),
	    )
	    if err != nil {
	        panic(err)
	    }

	    fmt.Printf("Audio size: %d bytes\n", len(result.Audio))
	    fmt.Printf("Media type: %s\n", result.MediaType)
	}

# Available Voices

OpenAI TTS supports several voices: alloy, echo, fable, onyx, nova, and shimmer.

	result, err := speech.Synthesize(ctx, model, "Hello!",
	    speech.WithVoice("nova"),
	)

# Audio Formats

Supported formats include: mp3, opus, aac, flac, wav, pcm.

	result, err := speech.Synthesize(ctx, model, "Test audio",
	    speech.WithFormat("opus"),
	)

# Speech Speed

Adjust the speed from 0.25 to 4.0 (default is 1.0):

	result, err := speech.Synthesize(ctx, model, "Fast speech",
	    speech.WithSpeed(1.5),
	)

# Custom Timeout

	result, err := speech.Synthesize(ctx, model, "Long text...",
	    speech.WithTimeout(120*time.Second),
	)

# HD Quality

	model := client.TTS1HD() // Higher quality audio
	result, err := speech.Synthesize(ctx, model, "Premium quality audio")

# GPT-4o-mini TTS (with instructions)

	model := client.GPT4oMiniTTS()
	result, err := speech.Synthesize(ctx, model, "Hello!",
	    speech.WithVoice("alloy"),
	    speech.WithInstructions("Speak slowly and clearly"),
	)
*/
type _ = provider.SpeechModel
