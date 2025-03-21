package mistral

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

// Provider implements the core.LLMProvider interface for Mistral AI
type Provider struct {
	config Config
	client *http.Client
}

// Config contains the configuration for the Mistral provider
type Config struct {
	// APIKey is the Mistral API key
	APIKey string `json:"api_key"`
	
	// Model is the model to use (e.g., "mistral-medium", "mistral-large")
	Model string `json:"model"`
	
	// Endpoint is the API endpoint (optional, defaults to Mistral's API)
	Endpoint string `json:"endpoint,omitempty"`
	
	// Timeout is the request timeout in seconds
	Timeout int `json:"timeout,omitempty"`
}

// NewProvider creates a new Mistral provider
func NewProvider(config Config) (*Provider, error) {
	if config.APIKey == "" {
		return nil, errors.New("API key is required")
	}
	
	if config.Model == "" {
		config.Model = "mistral-medium"
	}
	
	if config.Endpoint == "" {
		config.Endpoint = "https://api.mistral.ai/v1"
	}
	
	if config.Timeout == 0 {
		config.Timeout = 30
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
	return "mistral"
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
		fmt.Sprintf("%s/chat/completions", p.config.Endpoint),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	
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
	var mistralResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&mistralResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Convert to core.Response
	response := &core.Response{
		Text: mistralResp.Choices[0].Message.Content,
		TokensUsed: &core.TokenUsage{
			Prompt:     mistralResp.Usage.PromptTokens,
			Completion: mistralResp.Usage.CompletionTokens,
			Total:      mistralResp.Usage.TotalTokens,
		},
		FinishReason: mistralResp.Choices[0].FinishReason,
		ModelInfo: &core.ModelInfo{
			Name:      p.config.Model,
			Provider:  "mistral",
			Timestamp: time.Now().Format(time.RFC3339),
		},
		ProviderInfo: &core.ProviderInfo{
			Name:    "mistral",
			Version: "1.0.0",
		},
		Metadata: map[string]interface{}{
			"id":      mistralResp.ID,
			"created": mistralResp.Created,
			"model":   mistralResp.Model,
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
		fmt.Sprintf("%s/chat/completions", p.config.Endpoint),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	
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
	return &mistralStream{
		reader: resp.Body,
	}, nil
}

// prepareRequestBody prepares the request body for the Mistral API
func (p *Provider) prepareRequestBody(prompt *core.Prompt) ([]byte, error) {
	messages := []chatMessage{
		{
			Role:    "user",
			Content: prompt.Text,
		},
	}
	
	if prompt.SystemMessage != "" {
		messages = append([]chatMessage{
			{
				Role:    "system",
				Content: prompt.SystemMessage,
			},
		}, messages...)
	}
	
	reqBody := chatCompletionRequest{
		Model:            p.config.Model,
		Messages:         messages,
		Temperature:      prompt.Temperature,
		MaxTokens:        prompt.MaxTokens,
		TopP:             prompt.TopP,
		Stop:             prompt.StopSequences,
	}
	
	// Add schema if provided
	if prompt.Schema != nil {
		// In a real implementation, this would set the response_format to JSON
		// and include the schema
		// For simplicity, we're not implementing this
	}
	
	// Add additional parameters
	for k, v := range prompt.AdditionalParams {
		// In a real implementation, this would add the parameters to the request
		// For simplicity, we're not implementing this
	}
	
	return json.Marshal(reqBody)
}

// mistralStream implements core.ResponseStream for Mistral
type mistralStream struct {
	reader io.ReadCloser
	buffer []byte
}

// Next returns the next chunk of the response
func (s *mistralStream) Next() (*core.ResponseChunk, error) {
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
	var streamResp chatCompletionStreamResponse
	if err := json.Unmarshal(line[6:], &streamResp); err != nil {
		return nil, fmt.Errorf("failed to parse stream response: %w", err)
	}
	
	// Create a response chunk
	chunk := &core.ResponseChunk{
		Text:    streamResp.Choices[0].Delta.Content,
		IsFinal: false,
		Metadata: map[string]interface{}{
			"id":      streamResp.ID,
			"created": streamResp.Created,
			"model":   streamResp.Model,
		},
	}
	
	// Check if this is the final chunk
	if streamResp.Choices[0].FinishReason != "" {
		chunk.IsFinal = true
		chunk.FinishReason = streamResp.Choices[0].FinishReason
	}
	
	return chunk, nil
}

// Close closes the stream
func (s *mistralStream) Close() error {
	return s.reader.Close()
}

// readLine reads a line from the stream
func (s *mistralStream) readLine() ([]byte, error) {
	var line []byte
	var isPrefix bool
	
	for {
		// If we have data in the buffer, try to find a newline
		if len(s.buffer) > 0 {
			if i := bytes.IndexByte(s.buffer, '\n'); i >= 0 {
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
		
		// Append to the buffer
		s.buffer = append(s.buffer, buf[:n]...)
		
		// If the buffer is too large, return an error
		if len(s.buffer) > 1024*1024 {
			return nil, errors.New("buffer overflow")
		}
	}
}

// chatMessage represents a message in a chat completion request
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionRequest represents a request to the chat completions API
type chatCompletionRequest struct {
	Model            string        `json:"model"`
	Messages         []chatMessage `json:"messages"`
	Temperature      float64       `json:"temperature,omitempty"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	TopP             float64       `json:"top_p,omitempty"`
	Stop             []string      `json:"stop,omitempty"`
}

// chatCompletionResponse represents a response from the chat completions API
type chatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int `json:"index"`
		Message      struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// chatCompletionStreamResponse represents a streaming response from the chat completions API
type chatCompletionStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int `json:"index"`
		Delta        struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
