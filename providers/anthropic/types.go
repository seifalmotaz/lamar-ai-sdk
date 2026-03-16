package anthropic

import "encoding/json"

type MessageRequest struct {
	Model         string           `json:"model"`
	MaxTokens     int              `json:"max_tokens"`
	Messages      []Message        `json:"messages"`
	System        []SystemBlock    `json:"system,omitempty"`
	Temperature   *float64         `json:"temperature,omitempty"`
	TopP          *float64         `json:"top_p,omitempty"`
	TopK          *int             `json:"top_k,omitempty"`
	StopSequences []string         `json:"stop_sequences,omitempty"`
	Tools         []Tool           `json:"tools,omitempty"`
	ToolChoice    *ToolChoice      `json:"tool_choice,omitempty"`
	Metadata      *Metadata        `json:"metadata,omitempty"`
	Stream        bool             `json:"stream,omitempty"`
	Thinking      *ThinkingRequest `json:"thinking,omitempty"`
	MCPServers    []MCPServerAPI   `json:"mcp_servers,omitempty"`
	Container     *ContainerAPI    `json:"container,omitempty"`
}

type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type SystemBlock struct {
	Type         string           `json:"type"`
	Text         string           `json:"text"`
	CacheControl *CacheControlAPI `json:"cache_control,omitempty"`
}

type CacheControlAPI struct {
	Type string `json:"type"`
	TTL  string `json:"ttl,omitempty"`
}

type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Source    *ContentSource  `json:"source,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	ID        string          `json:"id,omitempty"`
}

type ContentSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type ToolChoice struct {
	Type                   string `json:"type"`
	Name                   string `json:"name,omitempty"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

type ThinkingRequest struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

type MessageResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason,omitempty"`
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        UsageResponse  `json:"usage"`
}

type UsageResponse struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

type StreamEvent struct {
	Type         string           `json:"type"`
	Message      *MessageResponse `json:"message,omitempty"`
	Index        int              `json:"index,omitempty"`
	Delta        json.RawMessage  `json:"delta,omitempty"`
	ContentBlock *ContentBlock    `json:"content_block,omitempty"`
	Error        *ErrorResponse   `json:"error,omitempty"`
	Usage        *UsageResponse   `json:"usage,omitempty"`
}

type StreamDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

type StreamMessageDelta struct {
	StopReason string        `json:"stop_reason,omitempty"`
	Usage      UsageResponse `json:"usage,omitempty"`
}

type ErrorResponse struct {
	Type  string      `json:"type"`
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type MCPServerAPI struct {
	Type               string                   `json:"type"`
	Name               string                   `json:"name"`
	URL                string                   `json:"url,omitempty"`
	AuthorizationToken string                   `json:"authorization_token,omitempty"`
	ToolConfiguration  *MCPToolConfigurationAPI `json:"tool_configuration,omitempty"`
}

type MCPToolConfigurationAPI struct {
	Enabled      bool     `json:"enabled,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

type ContainerAPI struct {
	ID     string           `json:"id,omitempty"`
	Skills []ContainerSkill `json:"skills,omitempty"`
}
