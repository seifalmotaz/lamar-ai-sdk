package provider

import (
	"encoding/json"
	"testing"
)

func TestTextContent(t *testing.T) {
	tc := Text("Hello, world!")

	if tc.Text != "Hello, world!" {
		t.Errorf("Text = %q, want %q", tc.Text, "Hello, world!")
	}

	var _ Content = tc
}

func TestImageContent(t *testing.T) {
	data := []byte("fake-image-data")
	ic := Image(data, "image/png")

	if string(ic.Data) != "fake-image-data" {
		t.Errorf("Data = %v, want %v", ic.Data, data)
	}
	if ic.MediaType != "image/png" {
		t.Errorf("MediaType = %q, want %q", ic.MediaType, "image/png")
	}

	var _ Content = ic
}

func TestImageFromURL(t *testing.T) {
	ic := ImageFromURL("https://example.com/image.png")

	if ic.MediaType != "url" {
		t.Errorf("MediaType = %q, want %q", ic.MediaType, "url")
	}
	if string(ic.Data) != "https://example.com/image.png" {
		t.Errorf("Data = %q, want %q", ic.Data, "https://example.com/image.png")
	}
}

func TestAudioContent(t *testing.T) {
	data := []byte("fake-audio-data")
	ac := Audio(data, "audio/mp3")

	if string(ac.Data) != "fake-audio-data" {
		t.Errorf("Data = %v, want %v", ac.Data, data)
	}
	if ac.MediaType != "audio/mp3" {
		t.Errorf("MediaType = %q, want %q", ac.MediaType, "audio/mp3")
	}

	var _ Content = ac
}

func TestToolCallContent(t *testing.T) {
	input := json.RawMessage(`{"location": "Tokyo"}`)
	tc := NewToolCallContent("call-123", "get_weather", input)

	if tc.ID != "call-123" {
		t.Errorf("ID = %q, want %q", tc.ID, "call-123")
	}
	if tc.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", tc.Name, "get_weather")
	}
	if string(tc.Input) != `{"location": "Tokyo"}` {
		t.Errorf("Input = %s, want %s", tc.Input, input)
	}

	var _ Content = tc
}

func TestToolCallContentFromJSON(t *testing.T) {
	input := map[string]string{"location": "Tokyo"}
	tc := NewToolCallContentFromJSON("call-123", "get_weather", input)

	if tc.ID != "call-123" {
		t.Errorf("ID = %q, want %q", tc.ID, "call-123")
	}
	if tc.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", tc.Name, "get_weather")
	}

	var parsed map[string]string
	if err := json.Unmarshal(tc.Input, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal input: %v", err)
	}
	if parsed["location"] != "Tokyo" {
		t.Errorf("location = %q, want %q", parsed["location"], "Tokyo")
	}
}

func TestToolResultContent(t *testing.T) {
	result := json.RawMessage(`{"temperature": 22.5}`)
	tr := NewToolResultContent("call-123", "get_weather", result, false)

	if tr.ID != "call-123" {
		t.Errorf("ID = %q, want %q", tr.ID, "call-123")
	}
	if tr.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", tr.Name, "get_weather")
	}
	if string(tr.Result) != `{"temperature": 22.5}` {
		t.Errorf("Result = %s, want %s", tr.Result, result)
	}
	if tr.IsError {
		t.Errorf("IsError = %v, want false", tr.IsError)
	}

	var _ Content = tr
}

func TestToolResultContentFromJSON(t *testing.T) {
	result := map[string]float64{"temperature": 22.5}
	tr := NewToolResultContentFromJSON("call-123", "get_weather", result, true)

	if tr.IsError != true {
		t.Errorf("IsError = %v, want true", tr.IsError)
	}

	var parsed map[string]float64
	if err := json.Unmarshal(tr.Result, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if parsed["temperature"] != 22.5 {
		t.Errorf("temperature = %v, want 22.5", parsed["temperature"])
	}
}

func TestReasoningContent(t *testing.T) {
	rc := ReasoningContent{Text: "Let me think..."}

	if rc.Text != "Let me think..." {
		t.Errorf("Text = %q, want %q", rc.Text, "Let me think...")
	}

	var _ Content = rc
}

func TestMessageConstructors(t *testing.T) {
	t.Run("SystemMessage", func(t *testing.T) {
		msg := SystemMessage("You are a helpful assistant.")
		if msg.Role != RoleSystem {
			t.Errorf("Role = %q, want %q", msg.Role, RoleSystem)
		}
		if len(msg.Content) != 1 {
			t.Fatalf("Content length = %d, want 1", len(msg.Content))
		}
		if tc, ok := msg.Content[0].(TextContent); !ok {
			t.Error("Content[0] should be TextContent")
		} else if tc.Text != "You are a helpful assistant." {
			t.Errorf("Text = %q, want %q", tc.Text, "You are a helpful assistant.")
		}
	})

	t.Run("UserMessage", func(t *testing.T) {
		msg := UserMessage("Hello!")
		if msg.Role != RoleUser {
			t.Errorf("Role = %q, want %q", msg.Role, RoleUser)
		}
		if len(msg.Content) != 1 {
			t.Fatalf("Content length = %d, want 1", len(msg.Content))
		}
		if tc, ok := msg.Content[0].(TextContent); !ok {
			t.Error("Content[0] should be TextContent")
		} else if tc.Text != "Hello!" {
			t.Errorf("Text = %q, want %q", tc.Text, "Hello!")
		}
	})

	t.Run("UserMessageWithContent", func(t *testing.T) {
		msg := UserMessageWithContent(Text("Hello"), Image([]byte("data"), "image/png"))
		if msg.Role != RoleUser {
			t.Errorf("Role = %q, want %q", msg.Role, RoleUser)
		}
		if len(msg.Content) != 2 {
			t.Fatalf("Content length = %d, want 2", len(msg.Content))
		}
	})

	t.Run("AssistantMessage", func(t *testing.T) {
		msg := AssistantMessage("Hi there!")
		if msg.Role != RoleAssistant {
			t.Errorf("Role = %q, want %q", msg.Role, RoleAssistant)
		}
	})

	t.Run("AssistantMessageWithToolCalls", func(t *testing.T) {
		tc := NewToolCallContent("call-1", "test", json.RawMessage("{}"))
		msg := AssistantMessageWithToolCalls(tc)
		if msg.Role != RoleAssistant {
			t.Errorf("Role = %q, want %q", msg.Role, RoleAssistant)
		}
		if len(msg.Content) != 1 {
			t.Fatalf("Content length = %d, want 1", len(msg.Content))
		}
	})

	t.Run("ToolResultMessage", func(t *testing.T) {
		tr := NewToolResultContent("call-1", "test", json.RawMessage("{}"), false)
		msg := ToolResultMessage(tr)
		if msg.Role != RoleTool {
			t.Errorf("Role = %q, want %q", msg.Role, RoleTool)
		}
	})
}

func TestUsageAdd(t *testing.T) {
	u1 := Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
	u2 := Usage{PromptTokens: 20, CompletionTokens: 10, TotalTokens: 30}

	result := u1.Add(u2)

	if result.PromptTokens != 30 {
		t.Errorf("PromptTokens = %d, want 30", result.PromptTokens)
	}
	if result.CompletionTokens != 15 {
		t.Errorf("CompletionTokens = %d, want 15", result.CompletionTokens)
	}
	if result.TotalTokens != 45 {
		t.Errorf("TotalTokens = %d, want 45", result.TotalTokens)
	}
}

func TestFinishReason(t *testing.T) {
	reasons := []struct {
		reason   FinishReason
		expected string
	}{
		{FinishReasonStop, "stop"},
		{FinishReasonLength, "length"},
		{FinishReasonToolCalls, "tool_calls"},
		{FinishReasonContentFilter, "content_filter"},
		{FinishReasonError, "error"},
	}

	for _, tt := range reasons {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.reason) != tt.expected {
				t.Errorf("FinishReason = %q, want %q", tt.reason, tt.expected)
			}
		})
	}
}

func TestToolChoiceConstructors(t *testing.T) {
	t.Run("ToolChoiceAuto", func(t *testing.T) {
		tc := ToolChoiceAuto()
		if tc.Type != "auto" {
			t.Errorf("Type = %q, want %q", tc.Type, "auto")
		}
	})

	t.Run("ToolChoiceNone", func(t *testing.T) {
		tc := ToolChoiceNone()
		if tc.Type != "none" {
			t.Errorf("Type = %q, want %q", tc.Type, "none")
		}
	})

	t.Run("ToolChoiceRequired", func(t *testing.T) {
		tc := ToolChoiceRequired()
		if tc.Type != "required" {
			t.Errorf("Type = %q, want %q", tc.Type, "required")
		}
	})

	t.Run("ToolChoiceNamed", func(t *testing.T) {
		tc := ToolChoiceNamed("get_weather")
		if tc.Type != "tool" {
			t.Errorf("Type = %q, want %q", tc.Type, "tool")
		}
		if tc.ToolName != "get_weather" {
			t.Errorf("ToolName = %q, want %q", tc.ToolName, "get_weather")
		}
	})
}

func TestResponseFormatConstructors(t *testing.T) {
	t.Run("ResponseFormatText", func(t *testing.T) {
		rf := ResponseFormatText()
		if rf.Type != "text" {
			t.Errorf("Type = %q, want %q", rf.Type, "text")
		}
	})

	t.Run("ResponseFormatJSON", func(t *testing.T) {
		rf := ResponseFormatJSON()
		if rf.Type != "json_object" {
			t.Errorf("Type = %q, want %q", rf.Type, "json_object")
		}
	})

	t.Run("ResponseFormatJSONSchema", func(t *testing.T) {
		schema := json.RawMessage(`{"type": "object"}`)
		rf := ResponseFormatJSONSchema(schema)
		if rf.Type != "json_schema" {
			t.Errorf("Type = %q, want %q", rf.Type, "json_schema")
		}
		if string(rf.JSONSchema) != `{"type": "object"}` {
			t.Errorf("JSONSchema = %s, want %s", rf.JSONSchema, schema)
		}
	})
}

func TestToolDefinition(t *testing.T) {
	schema := json.RawMessage(`{"type": "object", "properties": {"location": {"type": "string"}}}`)
	td := ToolDefinition{
		Name:        "get_weather",
		Description: "Get the weather",
		InputSchema: schema,
	}

	if td.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", td.Name, "get_weather")
	}
	if td.Description != "Get the weather" {
		t.Errorf("Description = %q, want %q", td.Description, "Get the weather")
	}
}

func TestToolCall(t *testing.T) {
	tc := ToolCall{
		ID:    "call-123",
		Name:  "get_weather",
		Input: json.RawMessage(`{"location": "Tokyo"}`),
	}

	if tc.ID != "call-123" {
		t.Errorf("ID = %q, want %q", tc.ID, "call-123")
	}
	if tc.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", tc.Name, "get_weather")
	}
}

func TestToolResult(t *testing.T) {
	tr := ToolResult{
		ID:      "call-123",
		Name:    "get_weather",
		Result:  json.RawMessage(`{"temperature": 22.5}`),
		IsError: false,
	}

	if tr.ID != "call-123" {
		t.Errorf("ID = %q, want %q", tr.ID, "call-123")
	}
	if tr.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", tr.Name, "get_weather")
	}
	if tr.IsError {
		t.Error("IsError should be false")
	}
}

func TestConfig(t *testing.T) {
	seed := 42
	rf := ResponseFormatJSON()
	cfg := Config{
		System:         "You are helpful",
		MaxTokens:      100,
		Temperature:    0.7,
		TopP:           0.9,
		TopK:           40,
		StopSequences:  []string{"stop"},
		Tools:          []ToolDefinition{{Name: "test"}},
		ToolChoice:     ToolChoiceAuto(),
		Seed:           &seed,
		ResponseFormat: &rf,
	}

	if cfg.System != "You are helpful" {
		t.Errorf("System = %q, want %q", cfg.System, "You are helpful")
	}
	if cfg.MaxTokens != 100 {
		t.Errorf("MaxTokens = %d, want 100", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", cfg.Temperature)
	}
	if *cfg.Seed != 42 {
		t.Errorf("Seed = %d, want 42", *cfg.Seed)
	}
}

func TestGenerateRequest(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Hello",
		Messages: []Message{
			UserMessage("Hi"),
		},
		System: "You are helpful",
		Config: Config{MaxTokens: 100},
	}

	if req.Prompt != "Hello" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Hello")
	}
	if len(req.Messages) != 1 {
		t.Errorf("Messages length = %d, want 1", len(req.Messages))
	}
}

func TestGenerateResult(t *testing.T) {
	result := GenerateResult{
		Text:    "Hello!",
		Content: []Content{Text("Hello!")},
		ToolCalls: []ToolCall{
			{ID: "call-1", Name: "test"},
		},
		FinishReason: FinishReasonStop,
		Usage:        Usage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15},
	}

	if result.Text != "Hello!" {
		t.Errorf("Text = %q, want %q", result.Text, "Hello!")
	}
	if result.FinishReason != FinishReasonStop {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, FinishReasonStop)
	}
	if len(result.ToolCalls) != 1 {
		t.Errorf("ToolCalls length = %d, want 1", len(result.ToolCalls))
	}
}

func TestStreamPartTypes(t *testing.T) {
	t.Run("StreamTextPart", func(t *testing.T) {
		p := StreamTextPart{Delta: "Hello"}
		var _ StreamPart = p
		if p.Delta != "Hello" {
			t.Errorf("Delta = %q, want %q", p.Delta, "Hello")
		}
	})

	t.Run("StreamToolCallPart", func(t *testing.T) {
		p := StreamToolCallPart{ToolCall: ToolCall{ID: "call-1"}}
		var _ StreamPart = p
		if p.ToolCall.ID != "call-1" {
			t.Errorf("ToolCall.ID = %q, want %q", p.ToolCall.ID, "call-1")
		}
	})

	t.Run("StreamFinishPart", func(t *testing.T) {
		p := StreamFinishPart{FinishReason: FinishReasonStop, Usage: Usage{TotalTokens: 10}}
		var _ StreamPart = p
		if p.FinishReason != FinishReasonStop {
			t.Errorf("FinishReason = %q, want %q", p.FinishReason, FinishReasonStop)
		}
	})

	t.Run("StreamErrorPart", func(t *testing.T) {
		err := &Error{Code: CodeInvalidRequest}
		p := StreamErrorPart{Error: err}
		var _ StreamPart = p
		if p.Error != err {
			t.Errorf("Error = %v, want %v", p.Error, err)
		}
	})
}

func TestEmbedRequest(t *testing.T) {
	req := EmbedRequest{
		Texts: []string{"Hello", "World"},
	}

	if len(req.Texts) != 2 {
		t.Errorf("Texts length = %d, want 2", len(req.Texts))
	}
}

func TestEmbedResult(t *testing.T) {
	result := EmbedResult{
		Embeddings: [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Usage:      Usage{PromptTokens: 10, CompletionTokens: 0, TotalTokens: 10},
	}

	if len(result.Embeddings) != 2 {
		t.Errorf("Embeddings length = %d, want 2", len(result.Embeddings))
	}
	if result.Usage.TotalTokens != 10 {
		t.Errorf("TotalTokens = %d, want 10", result.Usage.TotalTokens)
	}
}

func TestModelInfo_Methods(t *testing.T) {
	info := ModelInfo{
		Provider:     "openai",
		ModelID:      "gpt-4",
		Capabilities: []Capability{CapStreaming, CapTools, CapVision},
		MaxTokens:    4096,
		ContextSize:  8192,
	}

	if !info.HasCapability(CapStreaming) {
		t.Error("Should have CapStreaming")
	}
	if !info.HasCapability(CapTools) {
		t.Error("Should have CapTools")
	}
	if !info.HasCapability(CapVision) {
		t.Error("Should have CapVision")
	}
	if info.HasCapability(CapAudio) {
		t.Error("Should not have CapAudio")
	}
}

func TestGroundingMetadata(t *testing.T) {
	t.Run("basic grounding", func(t *testing.T) {
		gm := GroundingMetadata{
			Sources: []GroundingSource{
				{Index: 0, URI: "https://example.com/doc1", Title: "Document 1"},
				{Index: 1, URI: "https://example.com/doc2", Title: "Document 2"},
			},
			Citations: []Citation{
				{Text: "Hello", StartIndex: 0, EndIndex: 5, SourceIndices: []int{0}},
				{Text: "World", StartIndex: 6, EndIndex: 11, SourceIndices: []int{1}},
			},
		}

		if len(gm.Sources) != 2 {
			t.Errorf("Sources length = %d, want 2", len(gm.Sources))
		}
		if gm.Sources[0].URI != "https://example.com/doc1" {
			t.Errorf("Source URI = %q, want %q", gm.Sources[0].URI, "https://example.com/doc1")
		}
		if len(gm.Citations) != 2 {
			t.Errorf("Citations length = %d, want 2", len(gm.Citations))
		}
	})

	t.Run("JSON roundtrip", func(t *testing.T) {
		original := GroundingMetadata{
			Sources: []GroundingSource{
				{Index: 0, URI: "https://example.com/source", Title: "Source"},
			},
			Citations: []Citation{
				{Text: "citation text", StartIndex: 10, EndIndex: 25, SourceIndices: []int{0, 1}},
			},
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var restored GroundingMetadata
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(restored.Sources) != len(original.Sources) {
			t.Errorf("Sources length mismatch: got %d, want %d", len(restored.Sources), len(original.Sources))
		}
		if restored.Sources[0].URI != original.Sources[0].URI {
			t.Errorf("Source URI = %q, want %q", restored.Sources[0].URI, original.Sources[0].URI)
		}
		if len(restored.Citations) != len(original.Citations) {
			t.Errorf("Citations length mismatch: got %d, want %d", len(restored.Citations), len(original.Citations))
		}
	})

	t.Run("nil grounding in GenerateResult", func(t *testing.T) {
		result := GenerateResult{
			Text:         "Hello",
			FinishReason: FinishReasonStop,
		}

		if result.Grounding != nil {
			t.Error("Grounding should be nil by default")
		}
	})

	t.Run("grounding in GenerateResult", func(t *testing.T) {
		result := GenerateResult{
			Text:         "Hello",
			FinishReason: FinishReasonStop,
			Grounding: &GroundingMetadata{
				Sources: []GroundingSource{
					{Index: 0, URI: "https://example.com"},
				},
			},
		}

		if result.Grounding == nil {
			t.Error("Grounding should not be nil")
		}
		if len(result.Grounding.Sources) != 1 {
			t.Errorf("Sources length = %d, want 1", len(result.Grounding.Sources))
		}
	})
}

func TestCodeExecution(t *testing.T) {
	t.Run("basic code execution", func(t *testing.T) {
		ce := CodeExecution{
			Language: "PYTHON",
			Code:     "print('hello')",
			Outcome:  "OUTCOME_OK",
			Output:   "hello\n",
		}

		if ce.Language != "PYTHON" {
			t.Errorf("Language = %q, want %q", ce.Language, "PYTHON")
		}
		if ce.Code != "print('hello')" {
			t.Errorf("Code = %q, want %q", ce.Code, "print('hello')")
		}
	})

	t.Run("JSON roundtrip", func(t *testing.T) {
		original := CodeExecution{
			Language: "PYTHON",
			Code:     "x = 1 + 2\nprint(x)",
			Outcome:  "OUTCOME_OK",
			Output:   "3\n",
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var restored CodeExecution
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if restored.Language != original.Language {
			t.Errorf("Language = %q, want %q", restored.Language, original.Language)
		}
		if restored.Code != original.Code {
			t.Errorf("Code = %q, want %q", restored.Code, original.Code)
		}
		if restored.Outcome != original.Outcome {
			t.Errorf("Outcome = %q, want %q", restored.Outcome, original.Outcome)
		}
	})

	t.Run("code executions in GenerateResult", func(t *testing.T) {
		result := GenerateResult{
			Text:         "Here is the result:",
			FinishReason: FinishReasonStop,
			CodeExecutions: []CodeExecution{
				{Language: "PYTHON", Code: "print(1+1)", Outcome: "OUTCOME_OK", Output: "2\n"},
				{Language: "PYTHON", Code: "print(2+2)", Outcome: "OUTCOME_OK", Output: "4\n"},
			},
		}

		if len(result.CodeExecutions) != 2 {
			t.Errorf("CodeExecutions length = %d, want 2", len(result.CodeExecutions))
		}
		if result.CodeExecutions[0].Language != "PYTHON" {
			t.Errorf("Language = %q, want %q", result.CodeExecutions[0].Language, "PYTHON")
		}
	})

	t.Run("nil code executions by default", func(t *testing.T) {
		result := GenerateResult{
			Text:         "Hello",
			FinishReason: FinishReasonStop,
		}

		if result.CodeExecutions != nil && len(result.CodeExecutions) != 0 {
			t.Error("CodeExecutions should be nil/empty by default")
		}
	})
}
