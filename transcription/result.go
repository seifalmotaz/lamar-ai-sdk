package transcription

import "github.com/seifalmotaz/lamar-sdk/provider"

type Result struct {
	Text     string
	Segments []provider.TranscriptSegment
	Language string
	Duration float64
}
