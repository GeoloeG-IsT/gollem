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

	"github.com/user/gollem/pkg/core"
)

// Provider implements the core.LLMProvider interface for Google's Gemini models
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
	
	// ProjectID is the Google Cloud project ID (optional)
	ProjectID string `json:"project_id,omitempty"`
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
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", 
		p.config.Endpoint, 
		p.config.Model, 
		p.config.APIKey)
	
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
	
	// Extract the text from the response
	text := ""
	if len(googleResp.Candidates) > 0 && len(googleResp.Candidates[0].Content.Parts) > 0 {
		if textValue, ok := googleResp.Candidates[0].Content.Parts[0]["text"]; ok {
			text = textValue.(string)
		}
	}
	
	// Convert to core.Response
	response := &core.Response{
		Text: text,
		TokensUsed: &core.TokenUsage{
			Prompt:     googleResp.UsageMetadata.PromptTokenCount,
			Completion: googleResp.UsageMetadata.CandidatesTokenCount,
			Total:      googleResp.UsageMetadata.TotalTokenCount,
		},
		FinishReason: googleResp.Candidates[0].FinishReason,
		ModelInfo: &core.ModelInfo{
			Name:      p.config.Model,
			Provider:  "google",
			Timestamp: time.Now().Format(time.RFC3339),
		},
		ProviderInfo: &core.ProviderInfo{
			Name:    "google",
			Version: "1.0.0",
		},
		Metadata: map[string]interface{}{
			"model": googleResp.Candidates[0].ModelInfo.ModelName,
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
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s", 
		p.config.Endpoint, 
		p.config.Model, 
		p.config.APIKey)
	
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
	content := map[string]interface{}{
		"role": "user",
		"parts": []map[string]interface{}{
			{
				"text": prompt.Text,
			},
		},
	}
	
	// Create the request body
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{content},
		"generationConfig": map[string]interface{}{
			"temperature":      prompt.Temperature,
			"maxOutputTokens": prompt.MaxTokens,
			"topP":             prompt.TopP,
		},
	}
	
	// Add system message if provided
	if prompt.SystemMessage != "" {
		reqBody["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{
					"text": prompt.SystemMessage,
				},
			},
		}
	}
	
	// Add stop sequences if provided
	if len(prompt.StopSequences) > 0 {
		reqBody["generationConfig"].(map[string]interface{})["stopSequences"] = prompt.StopSequences
	}
	
	// Add schema if provided
	if prompt.Schema != nil {
		// In a real implementation, this would set the response format to JSON
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
	var streamResp generateContentStreamResponse
	if err := json.Unmarshal(line, &streamResp); err != nil {
		return nil, fmt.Errorf("failed to parse stream response: %w", err)
	}
	
	// Extract the text from the response
	text := ""
	if len(streamResp.Candidates) > 0 && len(streamResp.Candidates[0].Content.Parts) > 0 {
		if textValue, ok := streamResp.Candidates[0].Content.Parts[0]["text"]; ok {
			text = textValue.(string)
		}
	}
	
	// Create a response chunk
	chunk := &core.ResponseChunk{
		Text:    text,
		IsFinal: false,
		Metadata: map[string]interface{}{
			"model": streamResp.Candidates[0].ModelInfo.ModelName,
		},
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

// generateContentResponse represents a response from the generateContent API
type generateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []map[string]interface{} `json:"parts"`
			Role  string                   `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
		ModelInfo    struct {
			ModelName string `json:"modelName"`
		} `json:"modelInfo"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// generateContentStreamResponse represents a streaming response from the generateContent API
type generateContentStreamResponse struct {
	Candidates []struct {
		Content struct {
			Parts []map[string]interface{} `json:"parts"`
			Role  string                   `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
		ModelInfo    struct {
			ModelName string `json:"modelName"`
		} `json:"modelInfo"`
	} `json:"candidates"`
}
