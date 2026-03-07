package provider

import (
	"context"
	"encoding/json"
	"time"
)

// Content is a polymorphic type representing different message parts.
// Use type assertion to determine the specific type:
//
//	switch c := content.(type) {
//	case TextContent:
//	    fmt.Println(c.Text)
//	case ImageContent:
//	    // Handle image
//	}
type Content interface {
	content()
}

// TextContent represents a text message part.
type TextContent struct {
	Text string
}

func (TextContent) content() {}

// ImageContent represents an image message part with binary data.
type ImageContent struct {
	Data      []byte
	MediaType string
}

func (ImageContent) content() {}

// AudioContent represents an audio message part with binary data.
type AudioContent struct {
	Data      []byte
	MediaType string
}

func (AudioContent) content() {}

// ToolCallContent represents a tool/function call in a message.
type ToolCallContent struct {
	ID    string
	Name  string
	Input json.RawMessage
}

func (ToolCallContent) content() {}

// ToolResultContent represents the result of a tool/function call.
type ToolResultContent struct {
	ID      string
	Name    string
	Result  json.RawMessage
	IsError bool
}

func (ToolResultContent) content() {}

// ReasoningContent represents reasoning/thinking content (e.g., from O1 models).
type ReasoningContent struct {
	Text string
}

func (ReasoningContent) content() {}

// Text creates a TextContent from a string.
func Text(s string) TextContent {
	return TextContent{Text: s}
}

// Image creates an ImageContent from binary data and media type.
// For URL-based images, use ImageFromURL instead.
func Image(data []byte, mediaType string) ImageContent {
	return ImageContent{Data: data, MediaType: mediaType}
}

// ImageFromURL creates an ImageContent from a URL string.
// The media type is set to "url" to indicate the data contains a URL.
func ImageFromURL(url string) ImageContent {
	return ImageContent{MediaType: "url", Data: []byte(url)}
}

// Audio creates an AudioContent from binary data and media type.
func Audio(data []byte, mediaType string) AudioContent {
	return AudioContent{Data: data, MediaType: mediaType}
}

// NewToolCallContent creates a ToolCallContent with the given parameters.
func NewToolCallContent(id, name string, input json.RawMessage) ToolCallContent {
	return ToolCallContent{ID: id, Name: name, Input: input}
}

// NewToolCallContentFromJSON creates a ToolCallContent by marshaling the input to JSON.
// If marshaling fails, an empty RawMessage is used.
func NewToolCallContentFromJSON(id, name string, input any) ToolCallContent {
	data, _ := json.Marshal(input)
	return ToolCallContent{ID: id, Name: name, Input: data}
}

// NewToolResultContent creates a ToolResultContent with the given parameters.
func NewToolResultContent(id, name string, result json.RawMessage, isError bool) ToolResultContent {
	return ToolResultContent{ID: id, Name: name, Result: result, IsError: isError}
}

// NewToolResultContentFromJSON creates a ToolResultContent by marshaling the result to JSON.
// If marshaling fails, an empty RawMessage is used.
func NewToolResultContentFromJSON(id, name string, result any, isError bool) ToolResultContent {
	data, _ := json.Marshal(result)
	return ToolResultContent{ID: id, Name: name, Result: data, IsError: isError}
}

// MessageRole represents the role of a message sender.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type Message struct {
	Role    MessageRole
	Content []Content
}

func SystemMessage(text string) Message {
	return Message{
		Role:    RoleSystem,
		Content: []Content{Text(text)},
	}
}

func UserMessage(text string) Message {
	return Message{
		Role:    RoleUser,
		Content: []Content{Text(text)},
	}
}

func UserMessageWithContent(content ...Content) Message {
	return Message{
		Role:    RoleUser,
		Content: content,
	}
}

func AssistantMessage(text string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: []Content{Text(text)},
	}
}

func AssistantMessageWithToolCalls(toolCalls ...ToolCallContent) Message {
	content := make([]Content, len(toolCalls))
	for i, tc := range toolCalls {
		content[i] = tc
	}
	return Message{
		Role:    RoleAssistant,
		Content: content,
	}
}

func ToolResultMessage(toolResult ToolResultContent) Message {
	return Message{
		Role:    RoleTool,
		Content: []Content{toolResult},
	}
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

func (u Usage) Add(other Usage) Usage {
	return Usage{
		PromptTokens:     u.PromptTokens + other.PromptTokens,
		CompletionTokens: u.CompletionTokens + other.CompletionTokens,
		TotalTokens:      u.TotalTokens + other.TotalTokens,
	}
}

type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonError         FinishReason = "error"
)

type ToolChoice struct {
	Type     string
	ToolName string
}

func ToolChoiceAuto() ToolChoice {
	return ToolChoice{Type: "auto"}
}

func ToolChoiceNone() ToolChoice {
	return ToolChoice{Type: "none"}
}

func ToolChoiceRequired() ToolChoice {
	return ToolChoice{Type: "required"}
}

func ToolChoiceNamed(toolName string) ToolChoice {
	return ToolChoice{Type: "tool", ToolName: toolName}
}

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

type ToolResult struct {
	ID      string
	Name    string
	Result  json.RawMessage
	IsError bool
}

type ResponseFormat struct {
	Type       string
	JSONSchema json.RawMessage
}

func ResponseFormatText() ResponseFormat {
	return ResponseFormat{Type: "text"}
}

func ResponseFormatJSON() ResponseFormat {
	return ResponseFormat{Type: "json_object"}
}

func ResponseFormatJSONSchema(schema json.RawMessage) ResponseFormat {
	return ResponseFormat{Type: "json_schema", JSONSchema: schema}
}

type Config struct {
	System         string
	MaxTokens      int
	Temperature    float64
	TopP           float64
	TopK           int
	StopSequences  []string
	Tools          []ToolDefinition
	ToolChoice     ToolChoice
	Seed           *int
	ResponseFormat *ResponseFormat
}

type GenerateRequest struct {
	Prompt   string
	Messages []Message
	System   string
	Config   Config
}

type GenerateResult struct {
	Text         string
	Content      []Content
	ToolCalls    []ToolCall
	FinishReason FinishReason
	Usage        Usage
}

type StreamPart interface {
	streamPart()
}

type StreamTextPart struct {
	Delta string
}

func (StreamTextPart) streamPart() {}

type StreamToolCallPart struct {
	ToolCall ToolCall
}

func (StreamToolCallPart) streamPart() {}

type StreamFinishPart struct {
	FinishReason FinishReason
	Usage        Usage
}

func (StreamFinishPart) streamPart() {}

type StreamErrorPart struct {
	Error error
}

func (StreamErrorPart) streamPart() {}

type StreamResult struct {
	Stream       <-chan StreamPart
	Text         func() (string, error)
	Usage        func() (Usage, error)
	FinishReason func() (FinishReason, error)
	Done         <-chan struct{}
	Err          error
}

type EmbedRequest struct {
	Texts []string
}

type EmbedResult struct {
	Embeddings [][]float64
	Usage      Usage
}

type ModelInfo struct {
	Provider     string
	ModelID      string
	Capabilities []Capability
	MaxTokens    int
	ContextSize  int
}

func (m ModelInfo) HasCapability(cap Capability) bool {
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type MetricsCollector interface {
	RecordRequest(ctx context.Context, provider, model string, duration time.Duration, err error)
	RecordTokens(ctx context.Context, provider, model string, prompt, completion int)
	RecordStreamStart(ctx context.Context, provider, model string)
	RecordStreamEvent(ctx context.Context, provider, model, eventType string)
	RecordStreamEnd(ctx context.Context, provider, model string, duration time.Duration)
}

type NoopMetrics struct{}

func (NoopMetrics) RecordRequest(context.Context, string, string, time.Duration, error) {}
func (NoopMetrics) RecordTokens(context.Context, string, string, int, int)              {}
func (NoopMetrics) RecordStreamStart(context.Context, string, string)                   {}
func (NoopMetrics) RecordStreamEnd(context.Context, string, string, time.Duration)      {}
func (NoopMetrics) RecordStreamEvent(context.Context, string, string, string)           {}

// ImageFile represents an image file for input or editing.
type ImageFile struct {
	Data      []byte
	MediaType string
}

// NewImageFile creates an ImageFile from binary data and media type.
func NewImageFile(data []byte, mediaType string) ImageFile {
	return ImageFile{Data: data, MediaType: mediaType}
}

// NewImageFileFromURL creates an ImageFile from a URL.
func NewImageFileFromURL(url string) ImageFile {
	return ImageFile{MediaType: "url", Data: []byte(url)}
}

// ImageRequest is the request for image generation.
type ImageRequest struct {
	Prompt  string
	Files   []ImageFile
	Mask    *ImageFile
	N       int
	Size    string
	Quality string
	Format  string

	// Provider-specific options
	ProviderOptions map[string]any
}

// ImageUsage represents token usage for image generation.
type ImageUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ImageResult contains the result of image generation.
type ImageResult struct {
	Images           [][]byte
	RevisedPrompts   []string
	Usage            ImageUsage
	ProviderMetadata map[string]any
}

// TranscriptSegment represents a segment of transcribed audio.
type TranscriptSegment struct {
	Text        string
	StartSecond float64
	EndSecond   float64
}

// TranscriptionRequest is the request for audio transcription.
type TranscriptionRequest struct {
	Audio     []byte
	MediaType string
	Language  string
	Prompt    string

	// Provider-specific options
	ProviderOptions map[string]any
}

// TranscriptionResult contains the result of audio transcription.
type TranscriptionResult struct {
	Text     string
	Segments []TranscriptSegment
	Language string
	Duration float64
}

// SpeechRequest is the request for text-to-speech synthesis.
type SpeechRequest struct {
	Text         string
	Voice        string
	Format       string
	Speed        float64
	Instructions string

	// Provider-specific options
	ProviderOptions map[string]any
}

// SpeechResult contains the result of speech synthesis.
type SpeechResult struct {
	Audio     []byte
	MediaType string
}
