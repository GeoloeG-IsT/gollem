package tracing

import (
	"context"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// LLMTracer is a middleware that traces LLM interactions
type LLMTracer struct {
	tracerName string
	provider   core.LLMProvider
	tracer     Tracer
}

// Name returns the name of the LLMTracer
func (t *LLMTracer) Name() string {
	return t.tracerName
}

// NewLLMTracer creates a new LLM tracer
func NewLLMTracer(name string, provider core.LLMProvider, tracer Tracer) *LLMTracer {
	return &LLMTracer{
		tracerName: name,
		provider:   provider,
		tracer:     tracer,
	}
}

// Generate generates a response for a prompt with tracing
func (t *LLMTracer) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	// Start a span for the generation
	ctx, _ = t.tracer.StartSpan(ctx, "llm_generate")

	// Add attributes to the span
	t.tracer.SetAttribute(ctx, "prompt", prompt.Text)

	// Generate a response
	response, err := t.provider.Generate(ctx, prompt)

	// End the span
	if err != nil {
		t.tracer.EndSpan(ctx, SpanStatusError)
		return nil, err
	}

	// Add response attributes to the span
	t.tracer.SetAttribute(ctx, "response_length", len(response.Text))
	t.tracer.SetAttribute(ctx, "tokens_used", response.TokensUsed.Total)

	// End the span
	t.tracer.EndSpan(ctx, SpanStatusOK)

	return response, nil
}

// GenerateStream generates a streaming response for a prompt with tracing
func (t *LLMTracer) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
	// Start a span for the generation
	ctx, span := t.tracer.StartSpan(ctx, "llm_generate_stream")

	// Add attributes to the span
	t.tracer.SetAttribute(ctx, "prompt", prompt.Text)

	// Generate a streaming response
	stream, err := t.provider.GenerateStream(ctx, prompt)
	if err != nil {
		t.tracer.EndSpan(ctx, SpanStatusError)
		return nil, err
	}

	// Return a traced stream
	return &tracedResponseStream{
		stream: stream,
		tracer: t.tracer,
		ctx:    ctx,
		span:   span,
	}, nil
}

// tracedResponseStream is a response stream with tracing
type tracedResponseStream struct {
	stream core.ResponseStream
	tracer Tracer
	ctx    context.Context
	span   *Span
	closed bool
}

// Next returns the next chunk from the stream
func (s *tracedResponseStream) Next() (*core.ResponseChunk, error) {
	chunk, err := s.stream.Next()
	if err != nil {
		if err.Error() == "EOF" {
			s.tracer.SetAttribute(s.ctx, "stream_complete", true)
		} else {
			s.tracer.SetAttribute(s.ctx, "stream_error", err.Error())
		}
		return chunk, err
	}

	s.tracer.AddEvent(s.ctx, "stream_chunk", map[string]interface{}{
		"chunk_length": len(chunk.Text),
		"is_final":     chunk.IsFinal,
	})

	if chunk.IsFinal {
		s.tracer.SetAttribute(s.ctx, "finish_reason", chunk.FinishReason)
	}

	return chunk, nil
}

// Close closes the stream
func (s *tracedResponseStream) Close() error {
	if !s.closed {
		s.tracer.EndSpan(s.ctx, SpanStatusOK)
		s.closed = true
	}
	return s.stream.Close()
}
