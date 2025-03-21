package structured

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/validation"
)

// OutputParser parses structured output from LLM responses
type OutputParser struct {
	schema validation.JSONSchema
}

// NewOutputParser creates a new output parser with the given schema
func NewOutputParser(schema validation.JSONSchema) *OutputParser {
	return &OutputParser{
		schema: schema,
	}
}

// ParseResponse parses structured output from a response
func (p *OutputParser) ParseResponse(response *core.Response) (interface{}, error) {
	// If the response already has structured output, return it
	if response.StructuredOutput != nil {
		return response.StructuredOutput, nil
	}
	
	// Try to extract JSON from the response text
	jsonStr, err := extractJSON(response.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON: %w", err)
	}
	
	// Parse the JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// Validate the data against the schema
	validator := validation.NewValidator(p.schema)
	errors := validator.Validate(data)
	if len(errors) > 0 {
		return nil, fmt.Errorf("validation errors: %v", errors)
	}
	
	return data, nil
}

// extractJSON extracts JSON from text
func extractJSON(text string) (string, error) {
	// Look for JSON between triple backticks
	if start := strings.Index(text, "```json"); start != -1 {
		start += 7 // Skip "```json"
		if end := strings.Index(text[start:], "```"); end != -1 {
			return strings.TrimSpace(text[start : start+end]), nil
		}
	}
	
	// Look for JSON between single backticks
	if start := strings.Index(text, "`{"); start != -1 {
		start += 1 // Skip "`"
		if end := strings.Index(text[start:], "`"); end != -1 {
			return strings.TrimSpace(text[start : start+end]), nil
		}
	}
	
	// Look for JSON between curly braces
	if start := strings.Index(text, "{"); start != -1 {
		// Count opening and closing braces to find the matching closing brace
		count := 1
		for i := start + 1; i < len(text); i++ {
			if text[i] == '{' {
				count++
			} else if text[i] == '}' {
				count--
				if count == 0 {
					return strings.TrimSpace(text[start : i+1]), nil
				}
			}
		}
	}
	
	return "", fmt.Errorf("no JSON found in text")
}

// StructuredPromptBuilder builds prompts for structured output
type StructuredPromptBuilder struct {
	schema validation.JSONSchema
}

// NewStructuredPromptBuilder creates a new structured prompt builder
func NewStructuredPromptBuilder(schema validation.JSONSchema) *StructuredPromptBuilder {
	return &StructuredPromptBuilder{
		schema: schema,
	}
}

// BuildPrompt builds a prompt for structured output
func (b *StructuredPromptBuilder) BuildPrompt(prompt *core.Prompt) *core.Prompt {
	result := *prompt
	
	// Add the schema to the prompt
	result.Schema = b.schema
	
	// Add instructions for structured output
	schemaJSON, _ := json.MarshalIndent(b.schema, "", "  ")
	
	// Add instructions to the system message
	if result.SystemMessage != "" {
		result.SystemMessage += fmt.Sprintf("\n\nYour response must be valid JSON that conforms to the following schema:\n```json\n%s\n```\n\nProvide only the JSON in your response, without any additional text.", string(schemaJSON))
	} else {
		result.SystemMessage = fmt.Sprintf("Your response must be valid JSON that conforms to the following schema:\n```json\n%s\n```\n\nProvide only the JSON in your response, without any additional text.", string(schemaJSON))
	}
	
	return &result
}

// SchemaGenerator generates JSON schemas from Go structs
type SchemaGenerator struct{}

// NewSchemaGenerator creates a new schema generator
func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{}
}

// GenerateSchema generates a JSON schema from a Go struct
func (g *SchemaGenerator) GenerateSchema(v interface{}) (validation.JSONSchema, error) {
	return validation.GenerateSchema(v)
}

// StructuredOutputHandler handles structured output from LLMs
type StructuredOutputHandler struct {
	parser   *OutputParser
	builder  *StructuredPromptBuilder
	provider core.LLMProvider
}

// NewStructuredOutputHandler creates a new structured output handler
func NewStructuredOutputHandler(schema validation.JSONSchema, provider core.LLMProvider) *StructuredOutputHandler {
	return &StructuredOutputHandler{
		parser:   NewOutputParser(schema),
		builder:  NewStructuredPromptBuilder(schema),
		provider: provider,
	}
}

// Generate generates structured output for a prompt
func (h *StructuredOutputHandler) Generate(ctx context.Context, prompt *core.Prompt) (interface{}, error) {
	// Build a structured prompt
	structuredPrompt := h.builder.BuildPrompt(prompt)
	
	// Generate a response
	response, err := h.provider.Generate(ctx, structuredPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}
	
	// Parse the response
	result, err := h.parser.ParseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return result, nil
}
