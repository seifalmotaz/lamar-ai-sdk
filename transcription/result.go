package transcription

import "github.com/seifalmotaz/lamar-ai-sdk/provider"

type Result struct {
	Text     string
	Segments []provider.TranscriptSegment
	Language string
	Duration float64
}
