package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// TracerFactory manages multiple tracers
type TracerFactory struct {
	tracers map[string]Tracer
	mu      sync.Mutex
}

// NewTracerFactory creates a new tracer factory
func NewTracerFactory() *TracerFactory {
	return &TracerFactory{
		tracers: make(map[string]Tracer),
	}
}

// CreateTracer creates a new tracer based on configuration
func (f *TracerFactory) CreateTracer(config map[string]interface{}) (Tracer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	tracerType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tracer type in configuration")
	}

	switch tracerType {
	case "console":
		return NewConsoleTracer(), nil
	case "file":
		path, ok := config["path"].(string)
		if !ok {
			return nil, fmt.Errorf("missing path for file tracer")
		}
		return NewFileTracer(path)
	case "remote":
		endpoint, ok := config["endpoint"].(string)
		if !ok {
			return nil, fmt.Errorf("missing endpoint for remote tracer")
		}
		apiKey, _ := config["api_key"].(string) // Optional
		timeout, _ := config["timeout"].(int)   // Optional, default to 0
		return NewRemoteTracer(endpoint, apiKey, timeout)
	case "phoenix":
		endpoint, ok := config["endpoint"].(string)
		if !ok {
			return nil, fmt.Errorf("missing endpoint for phoenix tracer")
		}
		projectID, ok := config["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing project ID for phoenix tracer")
		}
		apiKey, _ := config["api_key"].(string) // Optional
		bufSize, _ := config["buf_size"].(int)  // Optional, default to 100
		tracer := NewPhoenixTracer(endpoint, apiKey, projectID, bufSize)
		return tracer, nil
	default:
		return nil, fmt.Errorf("unknown tracer type: %s", tracerType)
	}
}

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
type SpanStatus int

const (
	// SpanStatusOK indicates the span completed successfully
	SpanStatusOK SpanStatus = iota

	// SpanStatusError indicates the span completed with an error
	SpanStatusError

	// SpanStatusCanceled indicates the span was canceled
	SpanStatusCanceled
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
	EndSpan(ctx context.Context, status SpanStatus)

	// AddEvent adds an event to the current span
	AddEvent(ctx context.Context, name string, attributes map[string]interface{})

	// SetAttribute sets an attribute on the current span
	SetAttribute(ctx context.Context, key string, value interface{})

	// Flush flushes any pending spans
	Flush() error
}

// SpanOption configures a span
type SpanOption func(*Span)

// WithAttributes sets attributes on a span
func WithAttributes(attributes map[string]interface{}) SpanOption {
	return func(s *Span) {
		if s.Attributes == nil {
			s.Attributes = make(map[string]interface{})
		}
		for k, v := range attributes {
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
		}
	}
}

// spanKey is the context key for spans
type spanKey struct{}

// ConsoleTracer is a simple tracer that logs to the console
type ConsoleTracer struct {
	mu    sync.Mutex
	spans map[string]*Span
}

// NewConsoleTracer creates a new console tracer
func NewConsoleTracer() *ConsoleTracer {
	return &ConsoleTracer{
		spans: make(map[string]*Span),
	}
}

// StartSpan starts a new span
func (t *ConsoleTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create a new span
	span := &Span{
		ID:         generateID(),
		TraceID:    generateID(),
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

	// Store the span
	t.spans[span.ID] = span

	// Add to parent if exists
	if span.ParentID != "" {
		if parent, ok := t.spans[span.ParentID]; ok {
			parent.Children = append(parent.Children, span)
		}
	}

	// Store in context
	ctx = context.WithValue(ctx, spanKey{}, span)

	fmt.Printf("Started span: %s (%s)\n", span.Name, span.ID)
	return ctx, span
}

// EndSpan ends a span
func (t *ConsoleTracer) EndSpan(ctx context.Context, status SpanStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		fmt.Println("No span in context")
		return
	}

	// Update the span
	span.EndTime = time.Now()
	span.Status = status

	fmt.Printf("Ended span: %s (%s) - %v\n", span.Name, span.ID, span.EndTime.Sub(span.StartTime))
}

// AddEvent adds an event to the current span
func (t *ConsoleTracer) AddEvent(ctx context.Context, name string, attributes map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		fmt.Println("No span in context")
		return
	}

	// Add the event
	event := SpanEvent{
		Name:       name,
		Time:       time.Now(),
		Attributes: attributes,
	}
	span.Events = append(span.Events, event)

	fmt.Printf("Added event to span: %s - %s\n", span.Name, name)
}

// SetAttribute sets an attribute on the current span
func (t *ConsoleTracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		fmt.Println("No span in context")
		return
	}

	// Set the attribute
	span.Attributes[key] = value

	fmt.Printf("Set attribute on span: %s - %s=%v\n", span.Name, key, value)
}

// Flush flushes any pending spans
func (t *ConsoleTracer) Flush() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Print all spans
	for _, span := range t.spans {
		if span.ParentID == "" {
			t.printSpan(span, 0)
		}
	}

	// Clear spans
	t.spans = make(map[string]*Span)

	return nil
}

// printSpan prints a span and its children
func (t *ConsoleTracer) printSpan(span *Span, indent int) {
	// Print the span
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "  "
	}

	fmt.Printf("%sSpan: %s (%s)\n", indentStr, span.Name, span.ID)
	fmt.Printf("%s  Duration: %v\n", indentStr, span.EndTime.Sub(span.StartTime))
	fmt.Printf("%s  Status: %v\n", indentStr, span.Status)

	// Print attributes
	if len(span.Attributes) > 0 {
		fmt.Printf("%s  Attributes:\n", indentStr)
		for k, v := range span.Attributes {
			fmt.Printf("%s    %s: %v\n", indentStr, k, v)
		}
	}

	// Print events
	if len(span.Events) > 0 {
		fmt.Printf("%s  Events:\n", indentStr)
		for _, event := range span.Events {
			fmt.Printf("%s    %s (%v)\n", indentStr, event.Name, event.Time.Sub(span.StartTime))
			if len(event.Attributes) > 0 {
				for k, v := range event.Attributes {
					fmt.Printf("%s      %s: %v\n", indentStr, k, v)
				}
			}
		}
	}

	// Print children
	for _, child := range span.Children {
		t.printSpan(child, indent+1)
	}
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// FileTracer is a tracer that writes to a file
type FileTracer struct {
	mu       sync.Mutex
	spans    map[string]*Span
	file     *os.File
	filename string
}

// NewFileTracer creates a new file tracer
func NewFileTracer(filename string) (*FileTracer, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace file: %w", err)
	}

	return &FileTracer{
		spans:    make(map[string]*Span),
		file:     file,
		filename: filename,
	}, nil
}

// StartSpan starts a new span
func (t *FileTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create a new span
	span := &Span{
		ID:         generateID(),
		TraceID:    generateID(),
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

	// Store the span
	t.spans[span.ID] = span

	// Add to parent if exists
	if span.ParentID != "" {
		if parent, ok := t.spans[span.ParentID]; ok {
			parent.Children = append(parent.Children, span)
		}
	}

	// Store in context
	ctx = context.WithValue(ctx, spanKey{}, span)

	// Write to file
	t.writeEvent("start_span", map[string]interface{}{
		"span_id":    span.ID,
		"trace_id":   span.TraceID,
		"parent_id":  span.ParentID,
		"name":       span.Name,
		"start_time": span.StartTime,
		"attributes": span.Attributes,
	})

	return ctx, span
}

// EndSpan ends a span
func (t *FileTracer) EndSpan(ctx context.Context, status SpanStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Update the span
	span.EndTime = time.Now()
	span.Status = status

	// Write to file
	t.writeEvent("end_span", map[string]interface{}{
		"span_id":  span.ID,
		"end_time": span.EndTime,
		"status":   status,
		"duration": span.EndTime.Sub(span.StartTime).Milliseconds(),
	})
}

// AddEvent adds an event to the current span
func (t *FileTracer) AddEvent(ctx context.Context, name string, attributes map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Add the event
	event := SpanEvent{
		Name:       name,
		Time:       time.Now(),
		Attributes: attributes,
	}
	span.Events = append(span.Events, event)

	// Write to file
	t.writeEvent("add_event", map[string]interface{}{
		"span_id":    span.ID,
		"event_name": name,
		"event_time": event.Time,
		"attributes": attributes,
	})
}

// SetAttribute sets an attribute on the current span
func (t *FileTracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Set the attribute
	span.Attributes[key] = value

	// Write to file
	t.writeEvent("set_attribute", map[string]interface{}{
		"span_id": span.ID,
		"key":     key,
		"value":   value,
	})
}

// Flush flushes any pending spans
func (t *FileTracer) Flush() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Write all spans
	for _, span := range t.spans {
		if span.ParentID == "" {
			t.writeSpan(span)
		}
	}

	// Clear spans
	t.spans = make(map[string]*Span)

	return t.file.Sync()
}

// Close closes the tracer
func (t *FileTracer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.file.Close()
}

// writeEvent writes an event to the file
func (t *FileTracer) writeEvent(eventType string, data map[string]interface{}) {
	event := map[string]interface{}{
		"type":      eventType,
		"timestamp": time.Now(),
		"data":      data,
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal trace event: %v\n", err)
		return
	}

	_, err = t.file.Write(append(bytes, '\n'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write trace event: %v\n", err)
	}
}

// writeSpan writes a span and its children to the file
func (t *FileTracer) writeSpan(span *Span) {
	// Write the span
	t.writeEvent("span", map[string]interface{}{
		"span_id":    span.ID,
		"trace_id":   span.TraceID,
		"parent_id":  span.ParentID,
		"name":       span.Name,
		"start_time": span.StartTime,
		"end_time":   span.EndTime,
		"status":     span.Status,
		"attributes": span.Attributes,
		"events":     span.Events,
	})

	// Write children
	for _, child := range span.Children {
		t.writeSpan(child)
	}
}

// StreamTracer is a tracer that streams events to a writer
type StreamTracer struct {
	mu     sync.Mutex
	spans  map[string]*Span
	writer io.Writer
}

// NewStreamTracer creates a new stream tracer
func NewStreamTracer(writer io.Writer) *StreamTracer {
	return &StreamTracer{
		spans:  make(map[string]*Span),
		writer: writer,
	}
}

// StartSpan starts a new span
func (t *StreamTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create a new span
	span := &Span{
		ID:         generateID(),
		TraceID:    generateID(),
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

	// Store the span
	t.spans[span.ID] = span

	// Add to parent if exists
	if span.ParentID != "" {
		if parent, ok := t.spans[span.ParentID]; ok {
			parent.Children = append(parent.Children, span)
		}
	}

	// Store in context
	ctx = context.WithValue(ctx, spanKey{}, span)

	// Write to stream
	t.writeEvent("start_span", map[string]interface{}{
		"span_id":    span.ID,
		"trace_id":   span.TraceID,
		"parent_id":  span.ParentID,
		"name":       span.Name,
		"start_time": span.StartTime,
		"attributes": span.Attributes,
	})

	return ctx, span
}

// EndSpan ends a span
func (t *StreamTracer) EndSpan(ctx context.Context, status SpanStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Update the span
	span.EndTime = time.Now()
	span.Status = status

	// Write to stream
	t.writeEvent("end_span", map[string]interface{}{
		"span_id":  span.ID,
		"end_time": span.EndTime,
		"status":   status,
		"duration": span.EndTime.Sub(span.StartTime).Milliseconds(),
	})
}

// AddEvent adds an event to the current span
func (t *StreamTracer) AddEvent(ctx context.Context, name string, attributes map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Add the event
	event := SpanEvent{
		Name:       name,
		Time:       time.Now(),
		Attributes: attributes,
	}
	span.Events = append(span.Events, event)

	// Write to stream
	t.writeEvent("add_event", map[string]interface{}{
		"span_id":    span.ID,
		"event_name": name,
		"event_time": event.Time,
		"attributes": attributes,
	})
}

// SetAttribute sets an attribute on the current span
func (t *StreamTracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the current span
	span, ok := ctx.Value(spanKey{}).(*Span)
	if !ok {
		return
	}

	// Set the attribute
	span.Attributes[key] = value

	// Write to stream
	t.writeEvent("set_attribute", map[string]interface{}{
		"span_id": span.ID,
		"key":     key,
		"value":   value,
	})
}

// Flush flushes any pending spans
func (t *StreamTracer) Flush() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear spans
	t.spans = make(map[string]*Span)

	return nil
}

// writeEvent writes an event to the stream
func (t *StreamTracer) writeEvent(eventType string, data map[string]interface{}) {
	event := map[string]interface{}{
		"type":      eventType,
		"timestamp": time.Now(),
		"data":      data,
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal trace event: %v\n", err)
		return
	}

	_, err = t.writer.Write(append(bytes, '\n'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write trace event: %v\n", err)
	}
}
