package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// Provider implements the core.LLMProvider interface for Anthropic
type Provider struct {
	config Config
	client *http.Client
}

// Config contains the configuration for the Anthropic provider
type Config struct {
	// APIKey is the Anthropic API key
	APIKey string `json:"api_key"`
	
	// Model is the model to use (e.g., "claude-3-opus", "claude-3-sonnet")
	Model string `json:"model"`
	
	// Endpoint is the API endpoint (optional, defaults to Anthropic's API)
	Endpoint string `json:"endpoint,omitempty"`
	
	// Timeout is the request timeout in seconds
	Timeout int `json:"timeout,omitempty"`
	
	// Version is the Anthropic API version
	Version string `json:"version,omitempty"`
}

// NewProvider creates a new Anthropic provider
func NewProvider(config Config) (*Provider, error) {
	if config.APIKey == "" {
		return nil, errors.New("API key is required")
	}
	
	if config.Model == "" {
		config.Model = "claude-3-opus"
	}
	
	if config.Endpoint == "" {
		config.Endpoint = "https://api.anthropic.com"
	}
	
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	
	if config.Version == "" {
		config.Version = "2023-06-01"
	}
	
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	
	return &Provider{
		config: config,
		client: client,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "anthropic"
}

// Generate generates a response for the given prompt
func (p *Provider) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	// Prepare the request
	reqBody, err := p.prepareRequestBody(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request body: %w", err)
	}
	
	// Create the request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/v1/messages", p.config.Endpoint),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.config.APIKey)
	req.Header.Set("Anthropic-Version", p.config.Version)
	
	// Send the request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check for errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse the response
	var anthropicResp messageResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Convert to core.Response
	response := &core.Response{
		Text: anthropicResp.Content[0].Text,
		TokensUsed: &core.TokenUsage{
			Prompt:     anthropicResp.Usage.InputTokens,
			Completion: anthropicResp.Usage.OutputTokens,
			Total:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
		FinishReason: anthropicResp.StopReason,
		ModelInfo: &core.ModelInfo{
			Name:     p.config.Model,
			Provider: "anthropic",
			Version:  "1.0.0",
		},
		ProviderInfo: &core.ProviderInfo{
			Name:    "anthropic",
			Version: "1.0.0",
		},
	}
	
	// Handle structured output if a schema was provided
	if prompt.Schema != nil {
		// In a real implementation, this would parse the JSON and validate it against the schema
		// For simplicity, we're just setting it to nil
		response.StructuredOutput = nil
	}
	
	return response, nil
}

// GenerateStream generates a streaming response for the given prompt
func (p *Provider) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
	// Prepare the request
	reqBody, err := p.prepareRequestBody(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request body: %w", err)
	}
	
	// Set streaming to true
	var reqMap map[string]interface{}
	if err := json.Unmarshal(reqBody, &reqMap); err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}
	reqMap["stream"] = true
	reqBody, err = json.Marshal(reqMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	
	// Create the request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/v1/messages", p.config.Endpoint),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.config.APIKey)
	req.Header.Set("Anthropic-Version", p.config.Version)
	
	// Send the request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Check for errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Create a stream
	return &anthropicStream{
		reader: resp.Body,
	}, nil
}

// prepareRequestBody prepares the request body for the Anthropic API
func (p *Provider) prepareRequestBody(prompt *core.Prompt) ([]byte, error) {
	// Create the messages
	messages := []message{
		{
			Role:    "user",
			Content: prompt.Text,
		},
	}
	
	// Create the request body
	reqBody := messageRequest{
		Model:       p.config.Model,
		Messages:    messages,
		MaxTokens:   prompt.MaxTokens,
		Temperature: prompt.Temperature,
		TopP:        prompt.TopP,
		StopSequences: prompt.StopSequences,
	}
	
	// Add system message if provided
	if prompt.SystemMessage != "" {
		reqBody.System = prompt.SystemMessage
	}
	
	// Add schema if provided
	if prompt.Schema != nil {
		// In a real implementation, this would set the response_format to JSON
		// and include the schema
		// For simplicity, we're not implementing this
	}
	
	// Add additional parameters
	// For simplicity, we're not implementing this
	
	return json.Marshal(reqBody)
}

// anthropicStream implements core.ResponseStream for Anthropic
type anthropicStream struct {
	reader io.ReadCloser
	buffer []byte
}

// Next returns the next chunk of the response
func (s *anthropicStream) Next() (*core.ResponseChunk, error) {
	// Read the next line
	line, err := s.readLine()
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("failed to read line: %w", err)
	}
	
	// Skip empty lines
	if len(line) == 0 {
		return s.Next()
	}
	
	// Skip lines that don't start with "data: "
	if len(line) < 6 || string(line[:6]) != "data: " {
		return s.Next()
	}
	
	// Handle the "data: [DONE]" message
	if string(line) == "data: [DONE]" {
		return nil, io.EOF
	}
	
	// Parse the JSON
	var streamResp messageStreamResponse
	if err := json.Unmarshal(line[6:], &streamResp); err != nil {
		return nil, fmt.Errorf("failed to parse stream response: %w", err)
	}
	
	// Create a response chunk
	chunk := &core.ResponseChunk{
		Text:    streamResp.Delta.Text,
		IsFinal: false,
	}
	
	// Check if this is the final chunk
	if streamResp.Type == "message_stop" {
		chunk.IsFinal = true
		chunk.FinishReason = streamResp.StopReason
	}
	
	return chunk, nil
}

// Close closes the stream
func (s *anthropicStream) Close() error {
	return s.reader.Close()
}

// readLine reads a line from the stream
func (s *anthropicStream) readLine() ([]byte, error) {
	var line []byte
	
	for {
		// If we have data in the buffer, try to find a newline
		if len(s.buffer) > 0 {
			i := bytes.IndexByte(s.buffer, '\n')
			if i >= 0 {
				line = s.buffer[:i]
				s.buffer = s.buffer[i+1:]
				return line, nil
			}
		}
		
		// Read more data
		buf := make([]byte, 1024)
		n, err := s.reader.Read(buf)
		if err != nil {
			if err == io.EOF && len(s.buffer) > 0 {
				// Return the remaining data
				line = s.buffer
				s.buffer = nil
				return line, nil
			}
			return nil, err
		}
		
		// Append to buffer
		s.buffer = append(s.buffer, buf[:n]...)
	}
}

// message represents a message in a message request
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messageRequest represents a request to the messages API
type messageRequest struct {
	Model         string    `json:"model"`
	Messages      []message `json:"messages"`
	System        string    `json:"system,omitempty"`
	MaxTokens     int       `json:"max_tokens,omitempty"`
	Temperature   float64   `json:"temperature,omitempty"`
	TopP          float64   `json:"top_p,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
}

// messageResponse represents a response from the messages API
type messageResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Role      string `json:"role"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// messageStreamResponse represents a streaming response from the messages API
type messageStreamResponse struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason,omitempty"`
	Delta      struct {
		Type    string `json:"type,omitempty"`
		Text    string `json:"text,omitempty"`
	} `json:"delta"`
}
