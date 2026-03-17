package middleware

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracing returns a middleware that adds OpenTelemetry tracing to requests.
// The tracer provider is used to create a tracer for the SDK.
//
// Example:
//
//	tp := otel.GetTracerProvider()
//	middleware.Tracing(tp)
func Tracing(tp trace.TracerProvider) Middleware {
	tracer := tp.Tracer("github.com/seifalmotaz/lamar-ai-sdk")

	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			ctx, span := tracer.Start(ctx, "lamar.generate",
				trace.WithAttributes(
					attribute.String("provider", req.Provider()),
					attribute.String("model", req.ModelID()),
				),
			)
			defer span.End()

			resp, err := next.Handle(ctx, req)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}

			span.SetAttributes(
				attribute.Int("prompt_tokens", resp.Usage().PromptTokens),
				attribute.Int("completion_tokens", resp.Usage().CompletionTokens),
				attribute.Int("total_tokens", resp.Usage().TotalTokens),
				attribute.String("finish_reason", string(resp.FinishReason())),
			)

			span.SetStatus(codes.Ok, "success")
			return resp, nil
		})
	}
}

// TracingWithConfig returns a tracing middleware with custom configuration.
type TracingConfig struct {
	// TracerName is the name used to create the tracer.
	TracerName string

	// SpanName is the name of the span created for each request.
	SpanName string

	// Attributes are additional attributes to add to the span.
	Attributes []attribute.KeyValue
}

// TracingWithConfig returns a middleware that adds OpenTelemetry tracing with custom configuration.
func TracingWithConfig(tp trace.TracerProvider, cfg TracingConfig) Middleware {
	tracerName := cfg.TracerName
	if tracerName == "" {
		tracerName = "github.com/seifalmotaz/lamar-ai-sdk"
	}
	spanName := cfg.SpanName
	if spanName == "" {
		spanName = "lamar.generate"
	}

	tracer := tp.Tracer(tracerName)

	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			attrs := []attribute.KeyValue{
				attribute.String("provider", req.Provider()),
				attribute.String("model", req.ModelID()),
			}
			attrs = append(attrs, cfg.Attributes...)

			ctx, span := tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
			defer span.End()

			resp, err := next.Handle(ctx, req)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}

			span.SetAttributes(
				attribute.Int("prompt_tokens", resp.Usage().PromptTokens),
				attribute.Int("completion_tokens", resp.Usage().CompletionTokens),
				attribute.Int("total_tokens", resp.Usage().TotalTokens),
				attribute.String("finish_reason", string(resp.FinishReason())),
			)

			span.SetStatus(codes.Ok, "success")
			return resp, nil
		})
	}
}

// TracingForEmbed returns a middleware that adds OpenTelemetry tracing for embedding requests.
func TracingForEmbed(tp trace.TracerProvider) Middleware {
	tracer := tp.Tracer("github.com/seifalmotaz/lamar-ai-sdk")

	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			ctx, span := tracer.Start(ctx, "lamar.embed",
				trace.WithAttributes(
					attribute.String("provider", req.Provider()),
					attribute.String("model", req.ModelID()),
					attribute.Int("input_count", req.InputCount()),
				),
			)
			defer span.End()

			resp, err := next.Handle(ctx, req)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}

			span.SetAttributes(
				attribute.Int("total_tokens", resp.Usage().TotalTokens),
			)

			span.SetStatus(codes.Ok, "success")
			return resp, nil
		})
	}
}
