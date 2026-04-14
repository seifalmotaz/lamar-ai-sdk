package ollama

type ChatConfig struct {
	Temperature      *float64
	TopP             *float64
	TopK             *int
	Seed             *int
	MaxTokens        int
	StopSequences    []string
	Mirostat         *int
	MirostatEta      *float64
	MirostatTau      *float64
	NumCtx           *int
	NumPredict       *int
	NumKeep          *int
	RepeatLastN      *int
	RepeatPenalty    *float64
	PresencePenalty  *float64
	FrequencyPenalty *float64
	TFSZ             *float64
	TypicalP         *float64
	MinP             *float64
	Think            *bool
	KeepAlive        string
}

func ChatTemperature(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.Temperature = &v
	}
}

func ChatTopP(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.TopP = &v
	}
}

func ChatTopK(v int) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.TopK = &v
	}
}

func ChatSeed(v int) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.Seed = &v
	}
}

func ChatMaxTokens(v int) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.MaxTokens = v
	}
}

func ChatStopSequences(seqs ...string) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.StopSequences = append(c.StopSequences, seqs...)
	}
}

func ChatMirostat(v int) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.Mirostat = &v
	}
}

func ChatMirostatEta(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.MirostatEta = &v
	}
}

func ChatMirostatTau(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.MirostatTau = &v
	}
}

func ChatNumCtx(v int) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.NumCtx = &v
	}
}

func ChatNumPredict(v int) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.NumPredict = &v
	}
}

func ChatNumKeep(v int) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.NumKeep = &v
	}
}

func ChatRepeatLastN(v int) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.RepeatLastN = &v
	}
}

func ChatRepeatPenalty(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.RepeatPenalty = &v
	}
}

func ChatPresencePenalty(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.PresencePenalty = &v
	}
}

func ChatFrequencyPenalty(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.FrequencyPenalty = &v
	}
}

func ChatTFSZ(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.TFSZ = &v
	}
}

func ChatTypicalP(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.TypicalP = &v
	}
}

func ChatMinP(v float64) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.MinP = &v
	}
}

func ChatThink(v bool) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.Think = &v
	}
}

func ChatKeepAlive(v string) func(*ChatConfig) {
	return func(c *ChatConfig) {
		c.KeepAlive = v
	}
}

func newChatConfig(opts ...func(*ChatConfig)) *ChatConfig {
	config := &ChatConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

func (c *ChatConfig) applyToRequest(req *ChatRequest) {
	if c == nil {
		return
	}

	options := &ChatOptions{}
	hasOptions := false

	if c.Temperature != nil {
		options.Temperature = c.Temperature
		hasOptions = true
	}
	if c.TopP != nil {
		options.TopP = c.TopP
		hasOptions = true
	}
	if c.TopK != nil {
		options.TopK = c.TopK
		hasOptions = true
	}
	if c.Seed != nil {
		options.Seed = c.Seed
		hasOptions = true
	}
	if c.Mirostat != nil {
		options.Mirostat = c.Mirostat
		hasOptions = true
	}
	if c.MirostatEta != nil {
		options.MirostatEta = c.MirostatEta
		hasOptions = true
	}
	if c.MirostatTau != nil {
		options.MirostatTau = c.MirostatTau
		hasOptions = true
	}
	if c.NumCtx != nil {
		options.NumCtx = c.NumCtx
		hasOptions = true
	}
	if c.NumPredict != nil {
		options.NumPredict = c.NumPredict
		hasOptions = true
	}
	if c.NumKeep != nil {
		options.NumKeep = c.NumKeep
		hasOptions = true
	}
	if c.RepeatLastN != nil {
		options.RepeatLastN = c.RepeatLastN
		hasOptions = true
	}
	if c.RepeatPenalty != nil {
		options.RepeatPenalty = c.RepeatPenalty
		hasOptions = true
	}
	if c.PresencePenalty != nil {
		options.PresencePenalty = c.PresencePenalty
		hasOptions = true
	}
	if c.FrequencyPenalty != nil {
		options.FrequencyPenalty = c.FrequencyPenalty
		hasOptions = true
	}
	if c.TFSZ != nil {
		options.TFSZ = c.TFSZ
		hasOptions = true
	}
	if c.TypicalP != nil {
		options.TypicalP = c.TypicalP
		hasOptions = true
	}
	if c.MinP != nil {
		options.MinP = c.MinP
		hasOptions = true
	}
	if len(c.StopSequences) > 0 {
		options.Stop = c.StopSequences
		hasOptions = true
	}

	if hasOptions {
		req.Options = options
	}

	if c.KeepAlive != "" {
		req.KeepAlive = c.KeepAlive
	}
}

type EmbeddingConfig struct {
	Truncate  *bool
	KeepAlive string
}

func EmbeddingTruncate(v bool) func(*EmbeddingConfig) {
	return func(c *EmbeddingConfig) {
		c.Truncate = &v
	}
}

func EmbeddingKeepAlive(v string) func(*EmbeddingConfig) {
	return func(c *EmbeddingConfig) {
		c.KeepAlive = v
	}
}

func newEmbeddingConfig(opts ...func(*EmbeddingConfig)) *EmbeddingConfig {
	config := &EmbeddingConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}
