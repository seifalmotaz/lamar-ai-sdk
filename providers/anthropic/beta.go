package anthropic

func (m *ChatModel) buildBetaHeaders() []string {
	var headers []string

	if m.config == nil {
		return headers
	}

	if m.config.Thinking != nil && m.config.Thinking.Type != "disabled" {
		headers = append(headers, "max-tokens-3-5-sonnet-2024-07-15")
	}

	if len(m.config.MCPServers) > 0 {
		headers = append(headers, "mcp-client-2025-04-04")
	}

	if m.config.Container != nil {
		headers = append(headers, "code-execution-2025-08-25")
		headers = append(headers, "skills-2025-10-02")
		headers = append(headers, "files-api-2025-04-14")
	}

	if m.config.StructuredOutputMode == "outputFormat" {
		headers = append(headers, "structured-outputs-2025-11-13")
	}

	if m.config.Speed == "fast" {
		headers = append(headers, "fast-mode-2026-02-01")
	}

	if m.config.Effort != "" {
		headers = append(headers, "effort-2025-11-24")
	}

	if m.config.ToolStreaming != nil && *m.config.ToolStreaming {
		headers = append(headers, "fine-grained-tool-streaming-2025-05-14")
	}

	return headers
}

func (m *ChatModel) hasBetaHeaders() bool {
	return len(m.buildBetaHeaders()) > 0
}

func mergeBetaHeaders(headers []string) string {
	if len(headers) == 0 {
		return ""
	}
	result := headers[0]
	for i := 1; i < len(headers); i++ {
		result += "," + headers[i]
	}
	return result
}
