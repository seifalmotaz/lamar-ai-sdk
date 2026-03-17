package contract

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type TestCase struct {
	Name         string
	Capabilities []TestCapability
	Run          func(t *testing.T, model provider.Model)
}

type CustomAssertions struct {
	SkipUsage      bool
	ValidateError  func(t *testing.T, err error)
	ValidateResult func(t *testing.T, result interface{})
}

type TestSuiteOptions struct {
	Name             string
	Models           []ModelWithCapabilities
	Timeout          time.Duration
	CustomAssertions CustomAssertions
}

type FeatureTestSuite struct {
	Options      TestSuiteOptions
	Generator    func(t *testing.T, model provider.Model)
	Streamer     func(t *testing.T, model provider.Model)
	Embedder     func(t *testing.T, model provider.Model)
	ImageGen     func(t *testing.T, model provider.Model)
	Transcriber  func(t *testing.T, model provider.Model)
	Synthesizer  func(t *testing.T, model provider.Model)
	ObjectGen    func(t *testing.T, model provider.Model)
	StreamObject func(t *testing.T, model provider.Model)
	Tools        func(t *testing.T, model provider.Model)
}

func NewFeatureTestSuite(opts TestSuiteOptions) *FeatureTestSuite {
	return &FeatureTestSuite{
		Options: opts,
	}
}

func (s *FeatureTestSuite) Run(t *testing.T) {
	for _, m := range s.Options.Models {
		s.runModelTests(t, m)
	}
}

func (s *FeatureTestSuite) runModelTests(t *testing.T, m ModelWithCapabilities) {
	modelID := m.Model.ModelID()

	if s.Generator != nil {
		t.Run(fmt.Sprintf("%s/Generator", modelID), func(t *testing.T) {
			RequireCapability(t, m, CapTextGeneration)
			s.Generator(t, m.Model)
		})
	}

	if s.Streamer != nil {
		t.Run(fmt.Sprintf("%s/Streamer", modelID), func(t *testing.T) {
			RequireCapability(t, m, CapTextGeneration)
			if !HasTestCapability(m, CapTextGeneration) {
				t.Skipf("model %s lacks capability: %s", modelID, CapTextGeneration)
			}
			s.Streamer(t, m.Model)
		})
	}

	if s.Embedder != nil {
		t.Run(fmt.Sprintf("%s/Embedding", modelID), func(t *testing.T) {
			RequireCapability(t, m, CapEmbedding)
			s.Embedder(t, m.Model)
		})
	}

	if s.ImageGen != nil {
		t.Run(fmt.Sprintf("%s/ImageGeneration", modelID), func(t *testing.T) {
			RequireCapability(t, m, CapImageGeneration)
			s.ImageGen(t, m.Model)
		})
	}

	if s.Transcriber != nil {
		t.Run(fmt.Sprintf("%s/Transcription", modelID), func(t *testing.T) {
			RequireCapability(t, m, CapTranscription)
			s.Transcriber(t, m.Model)
		})
	}

	if s.Synthesizer != nil {
		t.Run(fmt.Sprintf("%s/Speech", modelID), func(t *testing.T) {
			RequireCapability(t, m, CapSpeech)
			s.Synthesizer(t, m.Model)
		})
	}

	if s.ObjectGen != nil {
		t.Run(fmt.Sprintf("%s/ObjectGeneration", modelID), func(t *testing.T) {
			RequireCapability(t, m, CapObjectGeneration)
			s.ObjectGen(t, m.Model)
		})
	}

	if s.StreamObject != nil {
		t.Run(fmt.Sprintf("%s/StreamObject", modelID), func(t *testing.T) {
			RequireAllCapabilities(t, m, []TestCapability{CapObjectGeneration})
			s.StreamObject(t, m.Model)
		})
	}

	if s.Tools != nil {
		t.Run(fmt.Sprintf("%s/ToolCalls", modelID), func(t *testing.T) {
			RequireCapability(t, m, CapToolCalls)
			s.Tools(t, m.Model)
		})
	}
}

func RunWithModels(t *testing.T, models []ModelWithCapabilities, tests []TestCase) {
	for _, m := range models {
		for _, tc := range tests {
			name := fmt.Sprintf("%s/%s", m.Model.ModelID(), tc.Name)
			t.Run(name, func(t *testing.T) {
				if !HasAllTestCapabilities(m, tc.Capabilities) {
					t.Skipf("model %s lacks capabilities: %v", m.Model.ModelID(), tc.Capabilities)
				}
				tc.Run(t, m.Model)
			})
		}
	}
}

type GeneratorTestFunc func(t *testing.T, ctx context.Context, model provider.Generator)

func RunGeneratorTests(t *testing.T, models []ModelWithCapabilities, tests map[string]GeneratorTestFunc) {
	for _, m := range models {
		generator, ok := m.Model.(provider.Generator)
		if !ok {
			continue
		}

		for name, testFn := range tests {
			t.Run(fmt.Sprintf("%s/%s", m.Model.ModelID(), name), func(t *testing.T) {
				ctx := context.Background()
				if timeout := getTestTimeout(t); timeout > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, timeout)
					defer cancel()
				}
				testFn(t, ctx, generator)
			})
		}
	}
}

type StreamerTestFunc func(t *testing.T, ctx context.Context, model provider.Streamer)

func RunStreamerTests(t *testing.T, models []ModelWithCapabilities, tests map[string]StreamerTestFunc) {
	for _, m := range models {
		streamer, ok := m.Model.(provider.Streamer)
		if !ok {
			continue
		}

		for name, testFn := range tests {
			t.Run(fmt.Sprintf("%s/%s", m.Model.ModelID(), name), func(t *testing.T) {
				ctx := context.Background()
				if timeout := getTestTimeout(t); timeout > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, timeout)
					defer cancel()
				}
				testFn(t, ctx, streamer)
			})
		}
	}
}

type EmbedderTestFunc func(t *testing.T, ctx context.Context, model provider.EmbeddingModel)

func RunEmbedderTests(t *testing.T, models []ModelWithCapabilities, tests map[string]EmbedderTestFunc) {
	for _, m := range models {
		embedder, ok := m.Model.(provider.EmbeddingModel)
		if !ok {
			continue
		}

		for name, testFn := range tests {
			t.Run(fmt.Sprintf("%s/%s", m.Model.ModelID(), name), func(t *testing.T) {
				ctx := context.Background()
				if timeout := getTestTimeout(t); timeout > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, timeout)
					defer cancel()
				}
				testFn(t, ctx, embedder)
			})
		}
	}
}

func getTestTimeout(t *testing.T) time.Duration {
	t.Helper()
	if deadline, ok := t.Deadline(); ok {
		return time.Until(deadline)
	}
	return 0
}
