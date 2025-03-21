# Tracing Documentation

This document provides detailed information about the tracing functionality in Gollem.

## Overview

Gollem includes comprehensive tracing capabilities that allow you to monitor and debug LLM interactions. The tracing system is compatible with Arize Phoenix and provides multiple tracer implementations:

1. **Console Tracer**: Logs traces to the console
2. **File Tracer**: Logs traces to a file
3. **Remote Tracer**: Sends traces to a remote endpoint
4. **Phoenix Tracer**: Sends traces to Arize Phoenix

## Basic Usage

Here's a simple example of using tracing:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/user/gollem/pkg/core"
	"github.com/user/gollem/pkg/providers/openai"
	"github.com/user/gollem/pkg/tracing"
)

func main() {
	// Create a provider
	provider, err := openai.NewProvider(openai.Config{
		APIKey: "your-api-key",
		Model:  "gpt-4",
	})
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create a tracer
	tracer := tracing.NewConsoleTracer()

	// Wrap the provider with tracing
	tracedProvider := tracing.NewLLMTracer(provider, tracer)

	// Create a prompt
	prompt := core.NewPrompt("What is the capital of France?")

	// Generate a response
	ctx := context.Background()
	response, err := tracedProvider.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Print the response
	fmt.Println(response.Text)

	// Flush the tracer
	tracer.Flush(ctx)
}
```

## Components

### Span

The `Span` type represents a trace span:

```go
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
```

### SpanEvent

The `SpanEvent` type represents an event that occurred during a span:

```go
type SpanEvent struct {
	// Name is the name of the event
	Name string

	// Time is when the event occurred
	Time time.Time

	// Attributes contains additional information about the event
	Attributes map[string]interface{}
}
```

### Tracer

The `Tracer` interface defines methods for creating and managing spans:

```go
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
```

## Tracer Implementations

### Console Tracer

The console tracer logs traces to the console:

```go
// Create a console tracer
tracer := tracing.NewConsoleTracer()
```

### File Tracer

The file tracer logs traces to a file:

```go
// Create a file tracer
tracer, err := tracing.NewFileTracer("/path/to/trace.log")
if err != nil {
	log.Fatalf("Failed to create file tracer: %v", err)
}

// Close the tracer when done
defer tracer.Close()
```

### Remote Tracer

The remote tracer sends traces to a remote endpoint:

```go
// Create a remote tracer
tracer := tracing.NewRemoteTracer(
	"https://api.example.com/traces",
	"your-api-key",
	100, // Buffer size
)
```

### Phoenix Tracer

The Phoenix tracer sends traces to Arize Phoenix:

```go
// Create a Phoenix tracer
tracer := tracing.NewPhoenixTracer(
	"https://phoenix.arize.com/api/traces",
	"your-api-key",
	"your-project-id",
	100, // Buffer size
)
```

## LLM Tracing

The `LLMTracer` middleware adds tracing to an LLM provider:

```go
// Create a provider
provider, err := openai.NewProvider(openai.Config{
	APIKey: "your-api-key",
	Model:  "gpt-4",
})
if err != nil {
	log.Fatalf("Failed to create provider: %v", err)
}

// Create a tracer
tracer := tracing.NewConsoleTracer()

// Wrap the provider with tracing
tracedProvider := tracing.NewLLMTracer(provider, tracer)
```

## Tracer Factory

The `TracerFactory` creates tracers based on configuration:

```go
// Create a factory
factory := tracing.NewTracerFactory()

// Create a tracer from configuration
tracer, err := factory.CreateTracer(map[string]interface{}{
	"type": "console",
})
if err != nil {
	log.Fatalf("Failed to create tracer: %v", err)
}
```

## Configuration

Tracing can be configured via the configuration system:

```json
{
  "tracing": {
    "enabled": true,
    "type": "console",
    "file_path": "/path/to/trace.log",
    "remote_endpoint": "https://api.example.com/traces",
    "api_key": "your-api-key",
    "project_id": "your-project-id",
    "buffer_size": 100
  }
}
```

Environment variables can be used to override configuration values:

- `GOLLEM_TRACING_ENABLED`: Enable or disable tracing
- `GOLLEM_TRACING_TYPE`: Tracer type (console, file, remote, phoenix)
- `GOLLEM_TRACING_FILE_PATH`: File path for file tracer
- `GOLLEM_TRACING_REMOTE_ENDPOINT`: Endpoint for remote tracer
- `GOLLEM_TRACING_API_KEY`: API key for remote or Phoenix tracer
- `GOLLEM_TRACING_PROJECT_ID`: Project ID for Phoenix tracer
- `GOLLEM_TRACING_BUFFER_SIZE`: Buffer size for remote or Phoenix tracer

## Advanced Usage

### Custom Spans

You can create custom spans to trace specific operations:

```go
// Start a span
ctx, span := tracer.StartSpan(ctx, "my_operation")

// Perform the operation
result, err := performOperation()

// Add attributes
tracer.SetAttribute(ctx, "result", result)

// Add events
tracer.AddEvent(ctx, "operation_step", map[string]interface{}{
	"step": "processing",
})

// End the span
if err != nil {
	tracer.EndSpan(ctx, tracing.SpanStatusError, err)
} else {
	tracer.EndSpan(ctx, tracing.SpanStatusOK, nil)
}
```

### Nested Spans

You can create nested spans to trace hierarchical operations:

```go
// Start a parent span
ctx, parentSpan := tracer.StartSpan(ctx, "parent_operation")

// Start a child span
ctx, childSpan := tracer.StartSpan(ctx, "child_operation")

// Perform the child operation
childResult, err := performChildOperation()

// End the child span
if err != nil {
	tracer.EndSpan(ctx, tracing.SpanStatusError, err)
} else {
	tracer.EndSpan(ctx, tracing.SpanStatusOK, nil)
}

// Perform the parent operation
parentResult, err := performParentOperation()

// End the parent span
if err != nil {
	tracer.EndSpan(ctx, tracing.SpanStatusError, err)
} else {
	tracer.EndSpan(ctx, tracing.SpanStatusOK, nil)
}
```

### Custom Tracers

You can implement custom tracers by implementing the `Tracer` interface:

```go
type MyTracer struct {
	// Custom fields
}

func (t *MyTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	// Implement start span logic
}

func (t *MyTracer) EndSpan(ctx context.Context, status SpanStatus, err error) {
	// Implement end span logic
}

func (t *MyTracer) AddEvent(ctx context.Context, name string, attrs map[string]interface{}) {
	// Implement add event logic
}

func (t *MyTracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	// Implement set attribute logic
}

func (t *MyTracer) Flush(ctx context.Context) error {
	// Implement flush logic
}
```

## Integration with Arize Phoenix

Gollem's tracing system is compatible with [Arize Phoenix](https://github.com/Arize-ai/phoenix), an open-source tool for LLM observability:

1. Install and run Phoenix:
   ```bash
   pip install arize-phoenix
   phoenix start
   ```

2. Configure Gollem to use the Phoenix tracer:
   ```json
   {
     "tracing": {
       "enabled": true,
       "type": "phoenix",
       "remote_endpoint": "http://localhost:6006/api/traces",
       "project_id": "my_project"
     }
   }
   ```

3. Run your application and view traces in the Phoenix UI at `http://localhost:6006`.

## Best Practices

1. **Span Naming**: Use descriptive names for spans to make them easier to identify.
2. **Attributes**: Add relevant attributes to spans to provide context.
3. **Events**: Use events to mark important points in a span's lifecycle.
4. **Error Handling**: Always end spans with the appropriate status and error.
5. **Flushing**: Always flush tracers before your application exits.
6. **Buffer Size**: Choose an appropriate buffer size for remote tracers based on your application's needs.
7. **Sampling**: For high-volume applications, consider implementing sampling to reduce the number of traces.

## Examples

See the [examples directory](../examples/tracing) for complete tracing examples.
