package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/user/gollem/pkg/core"
)

// RemoteTracer is a tracer that sends traces to a remote endpoint
type RemoteTracer struct {
	endpoint string
	apiKey   string
	buffer   []*Span
	bufSize  int
	client   *http.Client
	mu       sync.Mutex
}

// NewRemoteTracer creates a new remote tracer
func NewRemoteTracer(endpoint, apiKey string, bufSize int) *RemoteTracer {
	if bufSize <= 0 {
		bufSize = 100
	}

	return &RemoteTracer{
		endpoint: endpoint,
		apiKey:   apiKey,
		buffer:   make([]*Span, 0, bufSize),
		bufSize:  bufSize,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// StartSpan starts a new span
func (t *RemoteTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
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

	// Store the span in the context
	return context.WithValue(ctx, spanKey{}, span), span
}

// EndSpan ends a span
func (t *RemoteTracer) EndSpan(ctx context.Context, status SpanStatus, err error) {
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

	// Add the span to the buffer
	t.buffer = append(t.buffer, span)

	// Flush if the buffer is full
	if len(t.buffer) >= t.bufSize {
		go t.flushInternal(context.Background())
	}
}

// AddEvent adds an event to the current span
func (t *RemoteTracer) AddEvent(ctx context.Context, name string, attrs map[string]interface{}) {
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
}

// SetAttribute sets an attribute on the current span
func (t *RemoteTracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the span from the context
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Set the attribute
	span.Attributes[key] = value
}

// Flush flushes any pending spans
func (t *RemoteTracer) Flush(ctx context.Context) error {
	return t.flushInternal(ctx)
}

// flushInternal flushes the buffer to the remote endpoint
func (t *RemoteTracer) flushInternal(ctx context.Context) error {
	t.mu.Lock()
	if len(t.buffer) == 0 {
		t.mu.Unlock()
		return nil
	}

	// Copy the buffer
	spans := make([]*Span, len(t.buffer))
	copy(spans, t.buffer)
	t.buffer = t.buffer[:0]
	t.mu.Unlock()

	// Convert spans to JSON
	data, err := json.Marshal(spans)
	if err != nil {
		return fmt.Errorf("failed to marshal spans: %w", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		t.endpoint,
		bytes.NewBuffer(data),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.apiKey))
	}

	// Send the request
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// PhoenixTracer is a tracer compatible with Arize Phoenix
type PhoenixTracer struct {
	endpoint string
	apiKey   string
	projectID string
	buffer   []*PhoenixSpan
	bufSize  int
	client   *http.Client
	mu       sync.Mutex
}

// PhoenixSpan represents a span in the Phoenix format
type PhoenixSpan struct {
	ID         string                 `json:"id"`
	TraceID    string                 `json:"trace_id"`
	ParentID   string                 `json:"parent_id,omitempty"`
	Name       string                 `json:"name"`
	StartTime  int64                  `json:"start_time"`
	EndTime    int64                  `json:"end_time"`
	Status     string                 `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	Events     []PhoenixEvent         `json:"events,omitempty"`
	ProjectID  string                 `json:"project_id"`
}

// PhoenixEvent represents an event in the Phoenix format
type PhoenixEvent struct {
	Name       string                 `json:"name"`
	Time       int64                  `json:"time"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// NewPhoenixTracer creates a new Phoenix tracer
func NewPhoenixTracer(endpoint, apiKey, projectID string, bufSize int) *PhoenixTracer {
	if bufSize <= 0 {
		bufSize = 100
	}

	return &PhoenixTracer{
		endpoint:  endpoint,
		apiKey:    apiKey,
		projectID: projectID,
		buffer:    make([]*PhoenixSpan, 0, bufSize),
		bufSize:   bufSize,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// StartSpan starts a new span
func (t *PhoenixTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
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

	// Store the span in the context
	return context.WithValue(ctx, spanKey{}, span), span
}

// EndSpan ends a span
func (t *PhoenixTracer) EndSpan(ctx context.Context, status SpanStatus, err error) {
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

	// Convert to Phoenix format
	phoenixSpan := &PhoenixSpan{
		ID:         span.ID,
		TraceID:    span.TraceID,
		ParentID:   span.ParentID,
		Name:       span.Name,
		StartTime:  span.StartTime.UnixNano() / 1000000, // Convert to milliseconds
		EndTime:    span.EndTime.UnixNano() / 1000000,   // Convert to milliseconds
		Status:     string(span.Status),
		Attributes: span.Attributes,
		ProjectID:  t.projectID,
	}

	// Convert events
	if len(span.Events) > 0 {
		phoenixSpan.Events = make([]PhoenixEvent, len(span.Events))
		for i, event := range span.Events {
			phoenixSpan.Events[i] = PhoenixEvent{
				Name:       event.Name,
				Time:       event.Time.UnixNano() / 1000000, // Convert to milliseconds
				Attributes: event.Attributes,
			}
		}
	}

	// Add the span to the buffer
	t.buffer = append(t.buffer, phoenixSpan)

	// Flush if the buffer is full
	if len(t.buffer) >= t.bufSize {
		go t.flushInternal(context.Background())
	}
}

// AddEvent adds an event to the current span
func (t *PhoenixTracer) AddEvent(ctx context.Context, name string, attrs map[string]interface{}) {
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
}

// SetAttribute sets an attribute on the current span
func (t *PhoenixTracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the span from the context
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Set the attribute
	span.Attributes[key] = value
}

// Flush flushes any pending spans
func (t *PhoenixTracer) Flush(ctx context.Context) error {
	return t.flushInternal(ctx)
}

// flushInternal flushes the buffer to the Phoenix endpoint
func (t *PhoenixTracer) flushInternal(ctx context.Context) error {
	t.mu.Lock()
	if len(t.buffer) == 0 {
		t.mu.Unlock()
		return nil
	}

	// Copy the buffer
	spans := make([]*PhoenixSpan, len(t.buffer))
	copy(spans, t.buffer)
	t.buffer = t.buffer[:0]
	t.mu.Unlock()

	// Convert spans to JSON
	data, err := json.Marshal(spans)
	if err != nil {
		return fmt.Errorf("failed to marshal spans: %w", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		t.endpoint,
		bytes.NewBuffer(data),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.apiKey))
	}

	// Send the request
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TracerFactory creates tracers based on configuration
type TracerFactory struct{}

// NewTracerFactory creates a new tracer factory
func NewTracerFactory() *TracerFactory {
	return &TracerFactory{}
}

// CreateTracer creates a tracer based on the configuration
func (f *TracerFactory) CreateTracer(config map[string]interface{}) (Tracer, error) {
	// Get the tracer type
	tracerType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("tracer type not specified")
	}

	// Create the tracer based on the type
	switch tracerType {
	case "console":
		return NewConsoleTracer(), nil
	case "file":
		path, ok := config["path"].(string)
		if !ok {
			return nil, fmt.Errorf("file path not specified")
		}
		return NewFileTracer(path)
	case "remote":
		endpoint, ok := config["endpoint"].(string)
		if !ok {
			return nil, fmt.Errorf("remote endpoint not specified")
		}
		apiKey, _ := config["api_key"].(string)
		bufSize := 100
		if size, ok := config["buffer_size"].(float64); ok {
			bufSize = int(size)
		}
		return NewRemoteTracer(endpoint, apiKey, bufSize), nil
	case "phoenix":
		endpoint, ok := config["endpoint"].(string)
		if !ok {
			return nil, fmt.Errorf("phoenix endpoint not specified")
		}
		apiKey, _ := config["api_key"].(string)
		projectID, ok := config["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("phoenix project ID not specified")
		}
		bufSize := 100
		if size, ok := config["buffer_size"].(float64); ok {
			bufSize = int(size)
		}
		return NewPhoenixTracer(endpoint, apiKey, projectID, bufSize), nil
	default:
		return nil, fmt.Errorf("unknown tracer type: %s", tracerType)
	}
}
