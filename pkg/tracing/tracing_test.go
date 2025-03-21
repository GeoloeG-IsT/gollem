package tracing_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/tracing"
)

// TestConsoleTracer tests the console tracer
func TestConsoleTracer(t *testing.T) {
	// Create a console tracer
	tracer := tracing.NewConsoleTracer()

	// Start a span
	ctx := context.Background()
	ctx, span := tracer.StartSpan(ctx, "test_span")

	// Check the span
	if span.Name != "test_span" {
		t.Fatalf("Span name is incorrect: %s", span.Name)
	}

	// Add an event
	tracer.AddEvent(ctx, "test_event", map[string]interface{}{
		"key": "value",
	})

	// Set an attribute
	tracer.SetAttribute(ctx, "test_attribute", "test_value")

	// End the span
	tracer.EndSpan(ctx, tracing.SpanStatusOK, nil)

	// Test error case
	ctx, span = tracer.StartSpan(ctx, "error_span")
	tracer.EndSpan(ctx, tracing.SpanStatusError, fmt.Errorf("test error"))

	// Test nested spans
	ctx, parentSpan := tracer.StartSpan(ctx, "parent_span")
	ctx, childSpan := tracer.StartSpan(ctx, "child_span")

	// Check the child span
	if childSpan.ParentID != parentSpan.ID {
		t.Fatalf("Child span parent ID is incorrect: %s", childSpan.ParentID)
	}
	if childSpan.TraceID != parentSpan.TraceID {
		t.Fatalf("Child span trace ID is incorrect: %s", childSpan.TraceID)
	}

	// End the spans
	tracer.EndSpan(ctx, tracing.SpanStatusOK, nil)
	tracer.EndSpan(context.WithValue(context.Background(), struct{}{}, parentSpan), tracing.SpanStatusOK, nil)

	// Flush the tracer
	err := tracer.Flush(ctx)
	if err != nil {
		t.Fatalf("Failed to flush tracer: %v", err)
	}
}

// TestFileTracer tests the file tracer
func TestFileTracer(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	tracePath := filepath.Join(tempDir, "trace.log")

	// Create a file tracer
	tracer, err := tracing.NewFileTracer(tracePath)
	if err != nil {
		t.Fatalf("Failed to create file tracer: %v", err)
	}

	// Start a span
	ctx := context.Background()
	ctx, span := tracer.StartSpan(ctx, "test_span")

	// Check the span
	if span.Name != "test_span" {
		t.Fatalf("Span name is incorrect: %s", span.Name)
	}

	// Add an event
	tracer.AddEvent(ctx, "test_event", map[string]interface{}{
		"key": "value",
	})

	// Set an attribute
	tracer.SetAttribute(ctx, "test_attribute", "test_value")

	// End the span
	tracer.EndSpan(ctx, tracing.SpanStatusOK, nil)

	// Flush the tracer
	err = tracer.Flush(ctx)
	if err != nil {
		t.Fatalf("Failed to flush tracer: %v", err)
	}

	// Close the tracer
	err = tracer.Close()
	if err != nil {
		t.Fatalf("Failed to close tracer: %v", err)
	}

	// Check that the file exists
	_, err = os.Stat(tracePath)
	if err != nil {
		t.Fatalf("Trace file does not exist: %v", err)
	}

	// Read the file
	content, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to read trace file: %v", err)
	}

	// Check the content
	if len(content) == 0 {
		t.Fatal("Trace file is empty")
	}

	if !strings.Contains(string(content), "test_span") {
		t.Fatal("Trace file does not contain span name")
	}

	if !strings.Contains(string(content), "test_event") {
		t.Fatal("Trace file does not contain event name")
	}

	if !strings.Contains(string(content), "test_attribute") {
		t.Fatal("Trace file does not contain attribute name")
	}
}

// TestLLMTracer tests the LLM tracer middleware
func TestLLMTracer(t *testing.T) {
	// Create a mock provider
	provider := &MockProvider{
		name: "mock_provider",
	}

	// Create a console tracer
	tracer := tracing.NewConsoleTracer()

	// Create an LLM tracer
	llmTracer := tracing.NewLLMTracer(provider, tracer)

	// Check the name
	if llmTracer.Name() != "mock_provider_traced" {
		t.Fatalf("LLM tracer name is incorrect: %s", llmTracer.Name())
	}

	// Generate a response
	ctx := context.Background()
	prompt := core.NewPrompt("Test prompt")
	response, err := llmTracer.Generate(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to generate response: %v", err)
	}

	// Check the response
	if response.Text != "Mock response for: Test prompt" {
		t.Fatalf("Response text is incorrect: %s", response.Text)
	}

	// Generate a streaming response
	stream, err := llmTracer.GenerateStream(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to generate stream: %v", err)
	}

	// Read the stream
	var streamText string
	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read stream: %v", err)
		}
		streamText += chunk.Text
	}

	// Check the stream text
	if streamText != "Mock response for: Test prompt" {
		t.Fatalf("Stream text is incorrect: %s", streamText)
	}

	// Close the stream
	err = stream.Close()
	if err != nil {
		t.Fatalf("Failed to close stream: %v", err)
	}
}

// TestTracerFactory tests the tracer factory
func TestTracerFactory(t *testing.T) {
	// Create a factory
	factory := tracing.NewTracerFactory()

	// Create a console tracer
	consoleConfig := map[string]interface{}{
		"type": "console",
	}
	consoleTracer, err := factory.CreateTracer(consoleConfig)
	if err != nil {
		t.Fatalf("Failed to create console tracer: %v", err)
	}
	if consoleTracer == nil {
		t.Fatal("Console tracer is nil")
	}

	// Create a file tracer
	tempDir := t.TempDir()
	tracePath := filepath.Join(tempDir, "trace.log")
	fileConfig := map[string]interface{}{
		"type": "file",
		"path": tracePath,
	}
	fileTracer, err := factory.CreateTracer(fileConfig)
	if err != nil {
		t.Fatalf("Failed to create file tracer: %v", err)
	}
	if fileTracer == nil {
		t.Fatal("File tracer is nil")
	}

	// Test error cases
	_, err = factory.CreateTracer(map[string]interface{}{})
	if err == nil {
		t.Fatal("No error when creating tracer with no type")
	}

	_, err = factory.CreateTracer(map[string]interface{}{
		"type": "unknown",
	})
	if err == nil {
		t.Fatal("No error when creating tracer with unknown type")
	}

	_, err = factory.CreateTracer(map[string]interface{}{
		"type": "file",
	})
	if err == nil {
		t.Fatal("No error when creating file tracer with no path")
	}

	_, err = factory.CreateTracer(map[string]interface{}{
		"type":     "remote",
		"endpoint": "http://example.com",
	})
	if err != nil {
		t.Fatalf("Failed to create remote tracer: %v", err)
	}

	_, err = factory.CreateTracer(map[string]interface{}{
		"type": "remote",
	})
	if err == nil {
		t.Fatal("No error when creating remote tracer with no endpoint")
	}

	_, err = factory.CreateTracer(map[string]interface{}{
		"type":       "phoenix",
		"endpoint":   "http://example.com",
		"project_id": "test",
	})
	if err != nil {
		t.Fatalf("Failed to create phoenix tracer: %v", err)
	}

	_, err = factory.CreateTracer(map[string]interface{}{
		"type":     "phoenix",
		"endpoint": "http://example.com",
	})
	if err == nil {
		t.Fatal("No error when creating phoenix tracer with no project ID")
	}
}

// MockProvider is a mock implementation of the LLMProvider interface
type MockProvider struct {
	name string
}

// Name returns the name of the provider
func (p *MockProvider) Name() string {
	return p.name
}

// Generate generates a response for the given prompt
func (p *MockProvider) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	return &core.Response{
		Text: "Mock response for: " + prompt.Text,
		TokensUsed: &core.TokenUsage{
			Prompt:     10,
			Completion: 10,
			Total:      20,
		},
		FinishReason: "stop",
		ModelInfo: &core.ModelInfo{
			Name:     "mock-model",
			Provider: p.name,
		},
		ProviderInfo: &core.ProviderInfo{
			Name:    p.name,
			Version: "1.0.0",
		},
	}, nil
}

// GenerateStream generates a streaming response for the given prompt
func (p *MockProvider) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
	return &MockResponseStream{
		chunks: []string{
			"Mock ",
			"response ",
			"for: ",
			prompt.Text,
		},
	}, nil
}

// MockResponseStream is a mock implementation of the ResponseStream interface
type MockResponseStream struct {
	chunks []string
	index  int
}

// Next returns the next chunk of the response
func (s *MockResponseStream) Next() (*core.ResponseChunk, error) {
	if s.index >= len(s.chunks) {
		return nil, io.EOF
	}

	chunk := &core.ResponseChunk{
		Text:    s.chunks[s.index],
		IsFinal: s.index == len(s.chunks)-1,
	}

	if chunk.IsFinal {
		chunk.FinishReason = "stop"
	}

	s.index++
	return chunk, nil
}

// Close closes the stream
func (s *MockResponseStream) Close() error {
	return nil
}
