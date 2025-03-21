package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// Span represents a trace span
type Span struct {
	// ID is a unique identifier for the span
	ID string

	// TraceID is the ID of the trace this span belongs to
	TraceID string

	// ParentID is the ID of the parent span
	ParentID string

	// Name is the name of the span
	Name string

	// StartTime is when the span started
	StartTime time.Time

	// EndTime is when the span ended
	EndTime time.Time

	// Status is the status of the span
	Status SpanStatus

	// Attributes contains additional information about the span
	Attributes map[string]interface{}

	// Events are events that occurred during the span
	Events []SpanEvent

	// Children are child spans
	Children []*Span
}

// SpanStatus represents the status of a span
type SpanStatus string

const (
	// SpanStatusOK indicates the span completed successfully
	SpanStatusOK SpanStatus = "ok"

	// SpanStatusError indicates the span completed with an error
	SpanStatusError SpanStatus = "error"

	// SpanStatusCanceled indicates the span was canceled
	SpanStatusCanceled SpanStatus = "canceled"
)

// SpanEvent represents an event that occurred during a span
type SpanEvent struct {
	// Name is the name of the event
	Name string

	// Time is when the event occurred
	Time time.Time

	// Attributes contains additional information about the event
	Attributes map[string]interface{}
}

// Tracer creates and manages spans
type Tracer interface {
	// StartSpan starts a new span
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span)

	// EndSpan ends a span
	EndSpan(ctx context.Context, status SpanStatus, err error)

	// AddEvent adds an event to the current span
	AddEvent(ctx context.Context, name string, attrs map[string]interface{})

	// SetAttribute sets an attribute on the current span
	SetAttribute(ctx context.Context, key string, value interface{})

	// Flush flushes any pending spans
	Flush(ctx context.Context) error
}

// SpanOption configures a span
type SpanOption func(*Span)

// WithAttributes sets attributes on a span
func WithAttributes(attrs map[string]interface{}) SpanOption {
	return func(s *Span) {
		for k, v := range attrs {
			s.Attributes[k] = v
		}
	}
}

// WithParent sets the parent span
func WithParent(parent *Span) SpanOption {
	return func(s *Span) {
		if parent != nil {
			s.ParentID = parent.ID
			s.TraceID = parent.TraceID
			parent.Children = append(parent.Children, s)
		}
	}
}

// spanKey is the context key for spans
type spanKey struct{}

// ConsoleTracer is a tracer that logs to the console
type ConsoleTracer struct {
	mu sync.Mutex
}

// NewConsoleTracer creates a new console tracer
func NewConsoleTracer() *ConsoleTracer {
	return &ConsoleTracer{}
}

// StartSpan starts a new span
func (t *ConsoleTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create a new span
	span := &Span{
		ID:         fmt.Sprintf("span-%d", time.Now().UnixNano()),
		TraceID:    fmt.Sprintf("trace-%d", time.Now().UnixNano()),
		Name:       name,
		StartTime:  time.Now(),
		Attributes: make(map[string]interface{}),
		Events:     make([]SpanEvent, 0),
		Children:   make([]*Span, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(span)
	}

	// Get the parent span from the context
	if parent, ok := ctx.Value(spanKey{}).(*Span); ok && span.ParentID == "" {
		span.ParentID = parent.ID
		span.TraceID = parent.TraceID
		parent.Children = append(parent.Children, span)
	}

	// Log the span start
	fmt.Printf("[TRACE] %s: Started span %s (trace: %s, parent: %s)\n",
		span.StartTime.Format(time.RFC3339),
		span.Name,
		span.TraceID,
		span.ParentID)

	// Store the span in the context
	return context.WithValue(ctx, spanKey{}, span), span
}

// EndSpan ends a span
func (t *ConsoleTracer) EndSpan(ctx context.Context, status SpanStatus, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the span from the context
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Set the end time and status
	span.EndTime = time.Now()
	span.Status = status

	// Add error information
	if err != nil {
		span.Attributes["error"] = err.Error()
	}

	// Calculate the duration
	duration := span.EndTime.Sub(span.StartTime)

	// Log the span end
	fmt.Printf("[TRACE] %s: Ended span %s (trace: %s, duration: %s, status: %s)\n",
		span.EndTime.Format(time.RFC3339),
		span.Name,
		span.TraceID,
		duration,
		span.Status)

	// Log attributes
	if len(span.Attributes) > 0 {
		fmt.Printf("[TRACE] Attributes: %v\n", span.Attributes)
	}

	// Log events
	for _, event := range span.Events {
		fmt.Printf("[TRACE] Event: %s at %s\n",
			event.Name,
			event.Time.Format(time.RFC3339))
		if len(event.Attributes) > 0 {
			fmt.Printf("[TRACE] Event attributes: %v\n", event.Attributes)
		}
	}
}

// AddEvent adds an event to the current span
func (t *ConsoleTracer) AddEvent(ctx context.Context, name string, attrs map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the span from the context
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Create the event
	event := SpanEvent{
		Name:       name,
		Time:       time.Now(),
		Attributes: attrs,
	}

	// Add the event to the span
	span.Events = append(span.Events, event)

	// Log the event
	fmt.Printf("[TRACE] %s: Event %s on span %s (trace: %s)\n",
		event.Time.Format(time.RFC3339),
		event.Name,
		span.Name,
		span.TraceID)
	if len(event.Attributes) > 0 {
		fmt.Printf("[TRACE] Event attributes: %v\n", event.Attributes)
	}
}

// SetAttribute sets an attribute on the current span
func (t *ConsoleTracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the span from the context
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Set the attribute
	span.Attributes[key] = value

	// Log the attribute
	fmt.Printf("[TRACE] %s: Set attribute %s=%v on span %s (trace: %s)\n",
		time.Now().Format(time.RFC3339),
		key,
		value,
		span.Name,
		span.TraceID)
}

// Flush flushes any pending spans
func (t *ConsoleTracer) Flush(ctx context.Context) error {
	// Nothing to do for console tracer
	return nil
}

// FileTracer is a tracer that logs to a file
type FileTracer struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileTracer creates a new file tracer
func NewFileTracer(path string) (*FileTracer, error) {
	// Open the file
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open trace file: %w", err)
	}

	return &FileTracer{
		file: file,
	}, nil
}

// StartSpan starts a new span
func (t *FileTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create a new span
	span := &Span{
		ID:         fmt.Sprintf("span-%d", time.Now().UnixNano()),
		TraceID:    fmt.Sprintf("trace-%d", time.Now().UnixNano()),
		Name:       name,
		StartTime:  time.Now(),
		Attributes: make(map[string]interface{}),
		Events:     make([]SpanEvent, 0),
		Children:   make([]*Span, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(span)
	}

	// Get the parent span from the context
	if parent, ok := ctx.Value(spanKey{}).(*Span); ok && span.ParentID == "" {
		span.ParentID = parent.ID
		span.TraceID = parent.TraceID
		parent.Children = append(parent.Children, span)
	}

	// Log the span start
	t.file.WriteString(fmt.Sprintf("[TRACE] %s: Started span %s (trace: %s, parent: %s)\n",
		span.StartTime.Format(time.RFC3339),
		span.Name,
		span.TraceID,
		span.ParentID))

	// Store the span in the context
	return context.WithValue(ctx, spanKey{}, span), span
}

// EndSpan ends a span
func (t *FileTracer) EndSpan(ctx context.Context, status SpanStatus, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the span from the context
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Set the end time and status
	span.EndTime = time.Now()
	span.Status = status

	// Add error information
	if err != nil {
		span.Attributes["error"] = err.Error()
	}

	// Calculate the duration
	duration := span.EndTime.Sub(span.StartTime)

	// Log the span end
	t.file.WriteString(fmt.Sprintf("[TRACE] %s: Ended span %s (trace: %s, duration: %s, status: %s)\n",
		span.EndTime.Format(time.RFC3339),
		span.Name,
		span.TraceID,
		duration,
		span.Status))

	// Log attributes
	if len(span.Attributes) > 0 {
		t.file.WriteString(fmt.Sprintf("[TRACE] Attributes: %v\n", span.Attributes))
	}

	// Log events
	for _, event := range span.Events {
		t.file.WriteString(fmt.Sprintf("[TRACE] Event: %s at %s\n",
			event.Name,
			event.Time.Format(time.RFC3339)))
		if len(event.Attributes) > 0 {
			t.file.WriteString(fmt.Sprintf("[TRACE] Event attributes: %v\n", event.Attributes))
		}
	}
}

// AddEvent adds an event to the current span
func (t *FileTracer) AddEvent(ctx context.Context, name string, attrs map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the span from the context
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Create the event
	event := SpanEvent{
		Name:       name,
		Time:       time.Now(),
		Attributes: attrs,
	}

	// Add the event to the span
	span.Events = append(span.Events, event)

	// Log the event
	t.file.WriteString(fmt.Sprintf("[TRACE] %s: Event %s on span %s (trace: %s)\n",
		event.Time.Format(time.RFC3339),
		event.Name,
		span.Name,
		span.TraceID))
	if len(event.Attributes) > 0 {
		t.file.WriteString(fmt.Sprintf("[TRACE] Event attributes: %v\n", event.Attributes))
	}
}

// SetAttribute sets an attribute on the current span
func (t *FileTracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the span from the context
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Set the attribute
	span.Attributes[key] = value

	// Log the attribute
	t.file.WriteString(fmt.Sprintf("[TRACE] %s: Set attribute %s=%v on span %s (trace: %s)\n",
		time.Now().Format(time.RFC3339),
		key,
		value,
		span.Name,
		span.TraceID))
}

// Flush flushes any pending spans
func (t *FileTracer) Flush(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.file.Sync()
}

// Close closes the file tracer
func (t *FileTracer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.file.Close()
}

// LLMTracer is a middleware that adds tracing to an LLM provider
type LLMTracer struct {
	provider core.LLMProvider
	tracer   Tracer
}

// NewLLMTracer creates a new LLM tracer
func NewLLMTracer(provider core.LLMProvider, tracer Tracer) *LLMTracer {
	return &LLMTracer{
		provider: provider,
		tracer:   tracer,
	}
}

// Name returns the name of the provider
func (t *LLMTracer) Name() string {
	return t.provider.Name() + "_traced"
}

// Generate generates a response for the given prompt
func (t *LLMTracer) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	// Start a span
	ctx, span := t.tracer.StartSpan(ctx, "llm.generate", WithAttributes(map[string]interface{}{
		"provider": t.provider.Name(),
		"prompt":   prompt.Text,
	}))

	// Generate a response
	response, err := t.provider.Generate(ctx, prompt)

	// Add response information
	if response != nil {
		t.tracer.SetAttribute(ctx, "response.text", response.Text)
		if response.TokensUsed != nil {
			t.tracer.SetAttribute(ctx, "response.tokens.prompt", response.TokensUsed.Prompt)
			t.tracer.SetAttribute(ctx, "response.tokens.completion", response.TokensUsed.Completion)
			t.tracer.SetAttribute(ctx, "response.tokens.total", response.TokensUsed.Total)
		}
		t.tracer.SetAttribute(ctx, "response.finish_reason", response.FinishReason)
	}

	// End the span
	if err != nil {
		t.tracer.EndSpan(ctx, SpanStatusError, err)
	} else {
		t.tracer.EndSpan(ctx, SpanStatusOK, nil)
	}

	return response, err
}

// GenerateStream generates a streaming response for the given prompt
func (t *LLMTracer) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
	// Start a span
	ctx, span := t.tracer.StartSpan(ctx, "llm.generate_stream", WithAttributes(map[string]interface{}{
		"provider": t.provider.Name(),
		"prompt":   prompt.Text,
	}))

	// Generate a streaming response
	stream, err := t.provider.GenerateStream(ctx, prompt)
	if err != nil {
		t.tracer.EndSpan(ctx, SpanStatusError, err)
		return nil, err
	}

	// Wrap the stream with tracing
	return &tracedResponseStream{
		stream: stream,
		tracer: t.tracer,
		ctx:    ctx,
	}, nil
}

// tracedResponseStream wraps a ResponseStream with tracing
type tracedResponseStream struct {
	stream core.ResponseStream
	tracer Tracer
	ctx    context.Context
	chunks int
}

// Next returns the next chunk of the response
func (s *tracedResponseStream) Next() (*core.ResponseChunk, error) {
	// Get the next chunk
	chunk, err := s.stream.Next()

	// Add chunk information
	if chunk != nil {
		s.chunks++
		s.tracer.AddEvent(s.ctx, "stream.chunk", map[string]interface{}{
			"chunk":    s.chunks,
			"text":     chunk.Text,
			"is_final": chunk.IsFinal,
		})
	}

	// End the span if the stream is done
	if err != nil {
		if err == io.EOF {
			s.tracer.SetAttribute(s.ctx, "stream.chunks", s.chunks)
			s.tracer.EndSpan(s.ctx, SpanStatusOK, nil)
		} else {
			s.tracer.EndSpan(s.ctx, SpanStatusError, err)
		}
	}

	return chunk, err
}

// Close closes the stream
func (s *tracedResponseStream) Close() error {
	return s.stream.Close()
}
