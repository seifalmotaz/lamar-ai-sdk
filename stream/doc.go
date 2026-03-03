// Package stream provides streaming text generation from AI models.
//
// The stream package provides a channel-based API for consuming streaming
// responses from AI models that support the Streamer interface.
//
// The Stream function returns a Result that provides both a channel for
// real-time part consumption and thread-safe methods for waiting until
// completion and retrieving the full result.
//
// Example:
//
//	result := stream.Stream(ctx, model, "Tell me a story")
//
//	// Consume stream in real-time
//	for part := range result.Stream() {
//	    switch p := part.(type) {
//	    case provider.StreamTextPart:
//	        fmt.Print(p.Delta)
//	    case provider.StreamErrorPart:
//	        log.Error(p.Error)
//	    }
//	}
//
//	// Or wait for completion and get full text
//	text, err := result.Text()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(text)
package stream
