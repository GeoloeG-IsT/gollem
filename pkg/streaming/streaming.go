package streaming

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// StreamHandler defines the interface for handling streaming responses
type StreamHandler interface {
	// HandleChunk processes a response chunk
	HandleChunk(chunk *core.ResponseChunk) error
	
	// Complete is called when the stream is complete
	Complete(response *core.Response) error
}

// StreamProcessor processes streaming responses
type StreamProcessor struct {
	handler StreamHandler
	buffer  string
	mu      sync.Mutex
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(handler StreamHandler) *StreamProcessor {
	return &StreamProcessor{
		handler: handler,
		buffer:  "",
	}
}

// Process processes a streaming response
func (p *StreamProcessor) Process(ctx context.Context, stream core.ResponseStream) (*core.Response, error) {
	p.mu.Lock()
	p.buffer = ""
	p.mu.Unlock()
	
	var finalResponse *core.Response
	
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			chunk, err := stream.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// Stream is complete
					if finalResponse != nil {
						return finalResponse, p.handler.Complete(finalResponse)
					}
					return &core.Response{
						Text: p.buffer,
					}, nil
				}
				return nil, err
			}
			
			p.mu.Lock()
			p.buffer += chunk.Text
			p.mu.Unlock()
			
			if err := p.handler.HandleChunk(chunk); err != nil {
				return nil, err
			}
			
			if chunk.IsFinal {
				finalResponse = &core.Response{
					Text:         p.buffer,
					FinishReason: chunk.FinishReason,
					Metadata:     chunk.Metadata,
				}
			}
		}
	}
}

// DefaultStreamHandler is a simple implementation of StreamHandler
type DefaultStreamHandler struct {
	OnChunk    func(chunk *core.ResponseChunk) error
	OnComplete func(response *core.Response) error
}

// HandleChunk processes a response chunk
func (h *DefaultStreamHandler) HandleChunk(chunk *core.ResponseChunk) error {
	if h.OnChunk != nil {
		return h.OnChunk(chunk)
	}
	return nil
}

// Complete is called when the stream is complete
func (h *DefaultStreamHandler) Complete(response *core.Response) error {
	if h.OnComplete != nil {
		return h.OnComplete(response)
	}
	return nil
}

// TextStreamHandler collects text from a stream
type TextStreamHandler struct {
	Text      string
	OnNewText func(text string)
	mu        sync.Mutex
}

// HandleChunk processes a response chunk
func (h *TextStreamHandler) HandleChunk(chunk *core.ResponseChunk) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.Text += chunk.Text
	
	if h.OnNewText != nil {
		h.OnNewText(chunk.Text)
	}
	
	return nil
}

// Complete is called when the stream is complete
func (h *TextStreamHandler) Complete(response *core.Response) error {
	return nil
}

// JSONStreamHandler collects JSON from a stream
type JSONStreamHandler struct {
	Text       string
	JSONResult interface{}
	Schema     interface{}
	OnComplete func(result interface{})
	mu         sync.Mutex
}

// HandleChunk processes a response chunk
func (h *JSONStreamHandler) HandleChunk(chunk *core.ResponseChunk) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.Text += chunk.Text
	
	return nil
}

// Complete is called when the stream is complete
func (h *JSONStreamHandler) Complete(response *core.Response) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// In a real implementation, this would parse the JSON and validate it against the schema
	// For simplicity, we're just setting the result to the response's structured output
	h.JSONResult = response.StructuredOutput
	
	if h.OnComplete != nil {
		h.OnComplete(h.JSONResult)
	}
	
	return nil
}
