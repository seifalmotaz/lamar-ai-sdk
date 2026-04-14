package ollama

import "github.com/seifalmotaz/lamar-ai-sdk/provider"

func (p *Provider) Llama32() provider.LanguageModel {
	return p.StreamingModel("llama3.2")
}

func (p *Provider) Llama3() provider.LanguageModel {
	return p.StreamingModel("llama3")
}

func (p *Provider) Llama31() provider.LanguageModel {
	return p.StreamingModel("llama3.1")
}

func (p *Provider) Llama31_8B() provider.LanguageModel {
	return p.StreamingModel("llama3.1:8b")
}

func (p *Provider) Llama31_70B() provider.LanguageModel {
	return p.StreamingModel("llama3.1:70b")
}

func (p *Provider) Llama33() provider.LanguageModel {
	return p.StreamingModel("llama3.3")
}

func (p *Provider) Llama33_70B() provider.LanguageModel {
	return p.StreamingModel("llama3.3:70b")
}

func (p *Provider) Qwen25() provider.LanguageModel {
	return p.StreamingModel("qwen2.5")
}

func (p *Provider) Qwen25_7B() provider.LanguageModel {
	return p.StreamingModel("qwen2.5:7b")
}

func (p *Provider) Qwen25_14B() provider.LanguageModel {
	return p.StreamingModel("qwen2.5:14b")
}

func (p *Provider) Qwen3() provider.LanguageModel {
	return p.StreamingModel("qwen3")
}

func (p *Provider) DeepSeekR1() provider.LanguageModel {
	return p.StreamingModel("deepseek-r1")
}

func (p *Provider) DeepSeekV3() provider.LanguageModel {
	return p.StreamingModel("deepseek-v3")
}

func (p *Provider) Phi3() provider.LanguageModel {
	return p.StreamingModel("phi3")
}

func (p *Provider) Phi3Mini() provider.LanguageModel {
	return p.StreamingModel("phi3:mini")
}

func (p *Provider) Phi35() provider.LanguageModel {
	return p.StreamingModel("phi3.5")
}

func (p *Provider) Mistral() provider.LanguageModel {
	return p.StreamingModel("mistral")
}

func (p *Provider) Mistral7B() provider.LanguageModel {
	return p.StreamingModel("mistral:7b")
}

func (p *Provider) Mixtral() provider.LanguageModel {
	return p.StreamingModel("mixtral")
}

func (p *Provider) Codellama() provider.LanguageModel {
	return p.StreamingModel("codellama")
}

func (p *Provider) Llava() provider.LanguageModel {
	return p.StreamingModel("llava")
}

func (p *Provider) Gemma2() provider.LanguageModel {
	return p.StreamingModel("gemma2")
}

func (p *Provider) Gemma2_9B() provider.LanguageModel {
	return p.StreamingModel("gemma2:9b")
}

func (p *Provider) CommandR() provider.LanguageModel {
	return p.StreamingModel("command-r")
}

func (p *Provider) NomicEmbedText() provider.EmbeddingModel {
	return p.Embedding("nomic-embed-text")
}

func (p *Provider) MxbaiEmbedLarge() provider.EmbeddingModel {
	return p.Embedding("mxbai-embed-large")
}

func (p *Provider) AllMinilm() provider.EmbeddingModel {
	return p.Embedding("all-minilm")
}

func Llama32() provider.LanguageModel {
	return NewProvider().Llama32()
}

func Llama3() provider.LanguageModel {
	return NewProvider().Llama3()
}

func Llama31() provider.LanguageModel {
	return NewProvider().Llama31()
}

func Llama33() provider.LanguageModel {
	return NewProvider().Llama33()
}

func Qwen25() provider.LanguageModel {
	return NewProvider().Qwen25()
}

func Qwen3() provider.LanguageModel {
	return NewProvider().Qwen3()
}

func DeepSeekR1() provider.LanguageModel {
	return NewProvider().DeepSeekR1()
}

func DeepSeekV3() provider.LanguageModel {
	return NewProvider().DeepSeekV3()
}

func Phi3() provider.LanguageModel {
	return NewProvider().Phi3()
}

func Phi3Mini() provider.LanguageModel {
	return NewProvider().Phi3Mini()
}

func Mistral() provider.LanguageModel {
	return NewProvider().Mistral()
}

func Mixtral() provider.LanguageModel {
	return NewProvider().Mixtral()
}

func Llava() provider.LanguageModel {
	return NewProvider().Llava()
}

func Gemma2() provider.LanguageModel {
	return NewProvider().Gemma2()
}

func CommandR() provider.LanguageModel {
	return NewProvider().CommandR()
}

func NomicEmbedText() provider.EmbeddingModel {
	return NewProvider().NomicEmbedText()
}

func MxbaiEmbedLarge() provider.EmbeddingModel {
	return NewProvider().MxbaiEmbedLarge()
}

func AllMinilm() provider.EmbeddingModel {
	return NewProvider().AllMinilm()
}
