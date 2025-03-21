package google

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

// Provider implements the core.LLMProvider interface for Google AI
type Provider struct {
	config Config
	client *http.Client
}

// Config contains the configuration for the Google provider
type Config struct {
	// APIKey is the Google API key
	APIKey string `json:"api_key"`
	
	// Model is the model to use (e.g., "gemini-pro", "gemini-ultra")
	Model string `json:"model"`
	
	// Endpoint is the API endpoint (optional, defaults to Google's API)
	Endpoint string `json:"endpoint,omitempty"`
	
	// Timeout is the request timeout in seconds
	Timeout int `json:"timeout,omitempty"`
	
	// Project is the Google Cloud project ID (optional)
	Project string `json:"project,omitempty"`
	
	// Location is the Google Cloud location (optional)
	Location string `json:"location,omitempty"`
}

// NewProvider creates a new Google provider
func NewProvider(config Config) (*Provider, error) {
	if config.APIKey == "" {
		return nil, errors.New("API key is required")
	}
	
	if config.Model == "" {
		config.Model = "gemini-pro"
	}
	
	if config.Endpoint == "" {
		config.Endpoint = "https://generativelanguage.googleapis.com/v1beta"
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
	return "google"
}

// Generate generates a response for the given prompt
func (p *Provider) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	// Prepare the request
	reqBody, err := p.prepareRequestBody(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request body: %w", err)
	}
	
	// Create the request URL
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.config.Endpoint, p.config.Model, p.config.APIKey)
	
	// Create the request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
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
	var googleResp generateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Convert to core.Response
	response := &core.Response{
		Text: googleResp.Candidates[0].Content.Parts[0].Text,
		TokensUsed: &core.TokenUsage{
			Prompt:     googleResp.UsageMetadata.PromptTokenCount,
			Completion: googleResp.UsageMetadata.CandidatesTokenCount,
			Total:      googleResp.UsageMetadata.TotalTokenCount,
		},
		FinishReason: googleResp.Candidates[0].FinishReason,
		ModelInfo: &core.ModelInfo{
			Name:     p.config.Model,
			Provider: "google",
			Version:  "1.0.0",
		},
		ProviderInfo: &core.ProviderInfo{
			Name:    "google",
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
	
	// Create the request URL
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s", p.config.Endpoint, p.config.Model, p.config.APIKey)
	
	// Create the request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
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
	return &googleStream{
		reader: resp.Body,
	}, nil
}

// prepareRequestBody prepares the request body for the Google API
func (p *Provider) prepareRequestBody(prompt *core.Prompt) ([]byte, error) {
	// Create the content
	contentObj := contentType{
		Parts: []part{
			{
				Text: prompt.Text,
			},
		},
	}
	
	// Create the request body
	reqBody := generateContentRequest{
		Contents: []contentType{contentObj},
		GenerationConfig: generationConfig{
			Temperature:     prompt.Temperature,
			MaxOutputTokens: prompt.MaxTokens,
			TopP:            prompt.TopP,
			StopSequences:   prompt.StopSequences,
		},
	}
	
	// Add system message if provided
	if prompt.SystemMessage != "" {
		reqBody.SystemInstruction = &contentType{
			Parts: []part{
				{
					Text: prompt.SystemMessage,
				},
			},
		}
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

// googleStream implements core.ResponseStream for Google
type googleStream struct {
	reader io.ReadCloser
	buffer []byte
}

// Next returns the next chunk of the response
func (s *googleStream) Next() (*core.ResponseChunk, error) {
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
	
	// Parse the JSON
	var streamResp generateContentResponse
	if err := json.Unmarshal(line, &streamResp); err != nil {
		return nil, fmt.Errorf("failed to parse stream response: %w", err)
	}
	
	// Create a response chunk
	chunk := &core.ResponseChunk{
		Text:    streamResp.Candidates[0].Content.Parts[0].Text,
		IsFinal: false,
	}
	
	// Check if this is the final chunk
	if streamResp.Candidates[0].FinishReason != "" {
		chunk.IsFinal = true
		chunk.FinishReason = streamResp.Candidates[0].FinishReason
	}
	
	return chunk, nil
}

// Close closes the stream
func (s *googleStream) Close() error {
	return s.reader.Close()
}

// readLine reads a line from the stream
func (s *googleStream) readLine() ([]byte, error) {
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

// contentType represents the content in a request or response
type contentType struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

// part represents a part of content
type part struct {
	Text string `json:"text"`
}

// generationConfig represents the generation configuration
type generationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// generateContentRequest represents a request to the generateContent API
type generateContentRequest struct {
	Contents          []contentType      `json:"contents"`
	SystemInstruction *contentType       `json:"systemInstruction,omitempty"`
	GenerationConfig  generationConfig   `json:"generationConfig,omitempty"`
}

// generateContentResponse represents a response from the generateContent API
type generateContentResponse struct {
	Candidates []struct {
		Content      contentType `json:"content"`
		FinishReason string      `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount      int `json:"promptTokenCount"`
		CandidatesTokenCount  int `json:"candidatesTokenCount"`
		TotalTokenCount       int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}
