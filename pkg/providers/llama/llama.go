package llama

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

// Provider implements the core.LLMProvider interface for Llama models
type Provider struct {
	config Config
	client *http.Client
}

// Config contains the configuration for the Llama provider
type Config struct {
	// Endpoint is the API endpoint for the Llama model server
	Endpoint string `json:"endpoint"`
	
	// Model is the model to use
	Model string `json:"model,omitempty"`
	
	// APIKey is the API key (if required by the server)
	APIKey string `json:"api_key,omitempty"`
	
	// Timeout is the request timeout in seconds
	Timeout int `json:"timeout,omitempty"`
}

// NewProvider creates a new Llama provider
func NewProvider(config Config) (*Provider, error) {
	if config.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	
	if config.Timeout == 0 {
		config.Timeout = 60 // Longer timeout for local models
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
	return "llama"
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
		fmt.Sprintf("%s/v1/completions", p.config.Endpoint),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	}
	
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
	var llamaResp completionResponse
	if err := json.NewDecoder(resp.Body).Decode(&llamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Convert to core.Response
	response := &core.Response{
		Text: llamaResp.Choices[0].Text,
		TokensUsed: &core.TokenUsage{
			Prompt:     llamaResp.Usage.PromptTokens,
			Completion: llamaResp.Usage.CompletionTokens,
			Total:      llamaResp.Usage.TotalTokens,
		},
		FinishReason: llamaResp.Choices[0].FinishReason,
		ModelInfo: &core.ModelInfo{
			Name:      p.config.Model,
			Provider:  "llama",
			Timestamp: time.Now().Format(time.RFC3339),
		},
		ProviderInfo: &core.ProviderInfo{
			Name:    "llama",
			Version: "1.0.0",
		},
		Metadata: map[string]interface{}{
			"id":      llamaResp.ID,
			"created": llamaResp.Created,
			"model":   llamaResp.Model,
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
		fmt.Sprintf("%s/v1/completions", p.config.Endpoint),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	}
	
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
	return &llamaStream{
		reader: resp.Body,
	}, nil
}

// prepareRequestBody prepares the request body for the Llama API
func (p *Provider) prepareRequestBody(prompt *core.Prompt) ([]byte, error) {
	// Create the request body
	reqBody := completionRequest{
		Model:            p.config.Model,
		Prompt:           formatPrompt(prompt),
		Temperature:      prompt.Temperature,
		MaxTokens:        prompt.MaxTokens,
		TopP:             prompt.TopP,
		FrequencyPenalty: prompt.FrequencyPenalty,
		PresencePenalty:  prompt.PresencePenalty,
		Stop:             prompt.StopSequences,
	}
	
	// Add additional parameters
	for k, v := range prompt.AdditionalParams {
		// In a real implementation, this would add the parameters to the request
		// For simplicity, we're not implementing this
	}
	
	return json.Marshal(reqBody)
}

// formatPrompt formats the prompt for the Llama API
func formatPrompt(prompt *core.Prompt) string {
	// If there's a system message, include it
	if prompt.SystemMessage != "" {
		return fmt.Sprintf("<s>[INST] <<SYS>>\n%s\n<</SYS>>\n\n%s [/INST]", 
			prompt.SystemMessage, prompt.Text)
	}
	
	return fmt.Sprintf("<s>[INST] %s [/INST]", prompt.Text)
}

// llamaStream implements core.ResponseStream for Llama
type llamaStream struct {
	reader io.ReadCloser
	buffer []byte
}

// Next returns the next chunk of the response
func (s *llamaStream) Next() (*core.ResponseChunk, error) {
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
	var streamResp completionStreamResponse
	if err := json.Unmarshal(line[6:], &streamResp); err != nil {
		return nil, fmt.Errorf("failed to parse stream response: %w", err)
	}
	
	// Create a response chunk
	chunk := &core.ResponseChunk{
		Text:    streamResp.Choices[0].Text,
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
func (s *llamaStream) Close() error {
	return s.reader.Close()
}

// readLine reads a line from the stream
func (s *llamaStream) readLine() ([]byte, error) {
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

// completionRequest represents a request to the completions API
type completionRequest struct {
	Model            string   `json:"model,omitempty"`
	Prompt           string   `json:"prompt"`
	Temperature      float64  `json:"temperature,omitempty"`
	MaxTokens        int      `json:"max_tokens,omitempty"`
	TopP             float64  `json:"top_p,omitempty"`
	FrequencyPenalty float64  `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64  `json:"presence_penalty,omitempty"`
	Stop             []string `json:"stop,omitempty"`
}

// completionResponse represents a response from the completions API
type completionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// completionStreamResponse represents a streaming response from the completions API
type completionStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
