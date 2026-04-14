package ollama

import "encoding/json"

type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	Stream    bool          `json:"stream,omitempty"`
	Format    any           `json:"format,omitempty"`
	Tools     []Tool        `json:"tools,omitempty"`
	KeepAlive string        `json:"keep_alive,omitempty"`
	Options   *ChatOptions  `json:"options,omitempty"`
}

type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Images    []string   `json:"images,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type ChatOptions struct {
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	TopK             *int     `json:"top_k,omitempty"`
	Seed             *int     `json:"seed,omitempty"`
	NumPredict       *int     `json:"num_predict,omitempty"`
	NumCtx           *int     `json:"num_ctx,omitempty"`
	NumKeep          *int     `json:"num_keep,omitempty"`
	Mirostat         *int     `json:"mirostat,omitempty"`
	MirostatEta      *float64 `json:"mirostat_eta,omitempty"`
	MirostatTau      *float64 `json:"mirostat_tau,omitempty"`
	RepeatLastN      *int     `json:"repeat_last_n,omitempty"`
	RepeatPenalty    *float64 `json:"repeat_penalty,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	TFSZ             *float64 `json:"tfs_z,omitempty"`
	TypicalP         *float64 `json:"typical_p,omitempty"`
	MinP             *float64 `json:"min_p,omitempty"`
	Stop             []string `json:"stop,omitempty"`
}

type ChatResponse struct {
	Model      string          `json:"model"`
	CreatedAt  string          `json:"created_at"`
	Message    ResponseMessage `json:"message"`
	Done       bool            `json:"done"`
	DoneReason string          `json:"done_reason,omitempty"`

	PromptEvalCount int `json:"prompt_eval_count,omitempty"`
	EvalCount       int `json:"eval_count,omitempty"`

	TotalDuration      int64 `json:"total_duration,omitempty"`
	LoadDuration       int64 `json:"load_duration,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalDuration       int64 `json:"eval_duration,omitempty"`
}

type ResponseMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Thinking  string     `json:"thinking,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ChatChunk struct {
	Model      string          `json:"model"`
	CreatedAt  string          `json:"created_at"`
	Message    ResponseMessage `json:"message"`
	Done       bool            `json:"done"`
	DoneReason string          `json:"done_reason,omitempty"`

	PromptEvalCount int `json:"prompt_eval_count,omitempty"`
	EvalCount       int `json:"eval_count,omitempty"`
}

type EmbedRequest struct {
	Model     string `json:"model"`
	Input     string `json:"input"`
	Truncate  *bool  `json:"truncate,omitempty"`
	KeepAlive string `json:"keep_alive,omitempty"`
}

type EmbedBatchRequest struct {
	Model     string   `json:"model"`
	Input     []string `json:"input"`
	Truncate  *bool    `json:"truncate,omitempty"`
	KeepAlive string   `json:"keep_alive,omitempty"`
}

type EmbedResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float64 `json:"embeddings"`
	PromptEvalCount int         `json:"prompt_eval_count,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ModelsListResponse struct {
	Models []ModelInfo `json:"models"`
}

type ModelInfo struct {
	Name     string `json:"name"`
	Modified string `json:"modified_at"`
	Size     int64  `json:"size"`
	Digest   string `json:"digest"`
}

type ShowModelRequest struct {
	Name string `json:"name"`
}

type ShowModelResponse struct {
	License    string       `json:"license,omitempty"`
	Modelfile  string       `json:"modelfile,omitempty"`
	Parameters string       `json:"parameters,omitempty"`
	Template   string       `json:"template,omitempty"`
	Details    ModelDetails `json:"details,omitempty"`
}

type ModelDetails struct {
	Format            string   `json:"format,omitempty"`
	Family            string   `json:"family,omitempty"`
	ParameterSize     string   `json:"parameter_size,omitempty"`
	QuantizationLevel string   `json:"quantization_level,omitempty"`
	ParentModel       string   `json:"parent_model,omitempty"`
	Families          []string `json:"families,omitempty"`
}
