package optimization_test

import (
	"strings"
	"testing"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/optimization"
)

// TestTemplateStrategy tests the template strategy for prompt optimization
func TestTemplateStrategy(t *testing.T) {
	// Create a template strategy
	variables := map[string]string{
		"name":    "John",
		"company": "Acme Corp",
		"role":    "developer",
	}
	strategy := optimization.NewTemplateStrategy(variables)

	// Check the name
	if strategy.Name() != "template" {
		t.Fatalf("Strategy name is incorrect: %s", strategy.Name())
	}

	// Create a prompt with template variables
	prompt := core.NewPrompt("Hello {{name}}, welcome to {{company}}! You are a {{role}}.")

	// Optimize the prompt
	optimizedPrompt, err := strategy.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt: %v", err)
	}

	// Check the optimized prompt
	expectedText := "Hello John, welcome to Acme Corp! You are a developer."
	if optimizedPrompt.Text != expectedText {
		t.Fatalf("Optimized prompt text is incorrect: %s", optimizedPrompt.Text)
	}

	// Test with system message
	prompt.SystemMessage = "You are helping {{name}} from {{company}}."
	optimizedPrompt, err = strategy.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt with system message: %v", err)
	}

	// Check the optimized system message
	expectedSystemMessage := "You are helping John from Acme Corp."
	if optimizedPrompt.SystemMessage != expectedSystemMessage {
		t.Fatalf("Optimized system message is incorrect: %s", optimizedPrompt.SystemMessage)
	}
}

// TestTokenLimitStrategy tests the token limit strategy for prompt optimization
func TestTokenLimitStrategy(t *testing.T) {
	// Create a token estimator
	estimator := &MockTokenEstimator{}

	// Create a token limit strategy with a very small token limit
	strategy := optimization.NewTokenLimitStrategy(5, estimator)

	// Check the name
	if strategy.Name() != "token_limit" {
		t.Fatalf("Strategy name is incorrect: %s", strategy.Name())
	}

	// Create a prompt that exceeds the token limit
	prompt := core.NewPrompt("This is a very long prompt that exceeds the token limit.")

	// Optimize the prompt
	optimizedPrompt, err := strategy.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt: %v", err)
	}

	// Check the optimized prompt
	if len(optimizedPrompt.Text) >= len(prompt.Text) {
		t.Fatalf("Optimized prompt text is not shorter: %s", optimizedPrompt.Text)
	}

	// Test with system message
	prompt = core.NewPrompt("Short prompt")
	prompt.SystemMessage = "This is a very long system message that exceeds the token limit when combined with the prompt."

	// Optimize the prompt
	optimizedPrompt, err = strategy.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt with system message: %v", err)
	}

	// Check the optimized prompt
	if len(optimizedPrompt.Text)+len(optimizedPrompt.SystemMessage) >= len(prompt.Text)+len(prompt.SystemMessage) {
		t.Fatalf("Optimized prompt is not shorter: text=%s, system=%s", optimizedPrompt.Text, optimizedPrompt.SystemMessage)
	}
}

// MockTokenEstimator is a mock implementation of TokenEstimator for testing
type MockTokenEstimator struct{}

// EstimateTokens returns a token count that will ensure the test passes
func (e *MockTokenEstimator) EstimateTokens(text string) int {
	// Return a high token count to ensure truncation happens
	return len(text) / 2
}

// TestChainOfThoughtStrategy tests the chain of thought strategy for prompt optimization
func TestChainOfThoughtStrategy(t *testing.T) {
	// Create a chain of thought strategy
	strategy := optimization.NewChainOfThoughtStrategy()

	// Check the name
	if strategy.Name() != "chain_of_thought" {
		t.Fatalf("Strategy name is incorrect: %s", strategy.Name())
	}

	// Create a prompt
	prompt := core.NewPrompt("Solve this math problem: 5 + 7 * 2")

	// Optimize the prompt
	optimizedPrompt, err := strategy.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt: %v", err)
	}

	// Check the optimized prompt
	if optimizedPrompt.SystemMessage == "" {
		t.Fatal("Optimized prompt has no system message")
	}

	if !strings.Contains(optimizedPrompt.SystemMessage, "think step by step") {
		t.Fatalf("Optimized system message does not contain chain of thought instructions: %s", optimizedPrompt.SystemMessage)
	}

	// Test with existing system message
	prompt.SystemMessage = "You are a math tutor."
	optimizedPrompt, err = strategy.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt with system message: %v", err)
	}

	// Check the optimized system message
	if !strings.Contains(optimizedPrompt.SystemMessage, "You are a math tutor.") {
		t.Fatalf("Optimized system message does not contain original message: %s", optimizedPrompt.SystemMessage)
	}

	if !strings.Contains(optimizedPrompt.SystemMessage, "think step by step") {
		t.Fatalf("Optimized system message does not contain chain of thought instructions: %s", optimizedPrompt.SystemMessage)
	}
}

// TestFewShotStrategy tests the few-shot strategy for prompt optimization
func TestFewShotStrategy(t *testing.T) {
	// Create examples
	examples := []optimization.FewShotExample{
		{
			Input:  "What is 2 + 2?",
			Output: "4",
		},
		{
			Input:  "What is 3 * 3?",
			Output: "9",
		},
	}

	// Create a few-shot strategy
	strategy := optimization.NewFewShotStrategy(examples)

	// Check the name
	if strategy.Name() != "few_shot" {
		t.Fatalf("Strategy name is incorrect: %s", strategy.Name())
	}

	// Create a prompt
	prompt := core.NewPrompt("What is 5 + 5?")

	// Optimize the prompt
	optimizedPrompt, err := strategy.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt: %v", err)
	}

	// Check the optimized prompt
	if !strings.Contains(optimizedPrompt.Text, "Example 1") {
		t.Fatalf("Optimized prompt does not contain examples: %s", optimizedPrompt.Text)
	}

	if !strings.Contains(optimizedPrompt.Text, "What is 2 + 2?") {
		t.Fatalf("Optimized prompt does not contain first example input: %s", optimizedPrompt.Text)
	}

	if !strings.Contains(optimizedPrompt.Text, "4") {
		t.Fatalf("Optimized prompt does not contain first example output: %s", optimizedPrompt.Text)
	}

	if !strings.Contains(optimizedPrompt.Text, "What is 5 + 5?") {
		t.Fatalf("Optimized prompt does not contain original prompt: %s", optimizedPrompt.Text)
	}
}

// TestPromptOptimizer tests the prompt optimizer with multiple strategies
func TestPromptOptimizer(t *testing.T) {
	// Create strategies
	templateStrategy := optimization.NewTemplateStrategy(map[string]string{
		"name": "John",
	})
	cotStrategy := optimization.NewChainOfThoughtStrategy()

	// Create an optimizer with the strategies
	optimizer := optimization.NewPromptOptimizer(templateStrategy, cotStrategy)

	// Create a prompt
	prompt := core.NewPrompt("Hello {{name}}, solve this problem: 5 + 7 * 2")

	// Optimize the prompt
	optimizedPrompt, err := optimizer.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt: %v", err)
	}

	// Check the optimized prompt
	if !strings.Contains(optimizedPrompt.Text, "Hello John") {
		t.Fatalf("Optimized prompt does not contain template replacement: %s", optimizedPrompt.Text)
	}

	if !strings.Contains(optimizedPrompt.SystemMessage, "think step by step") {
		t.Fatalf("Optimized system message does not contain chain of thought instructions: %s", optimizedPrompt.SystemMessage)
	}

	// Add another strategy
	estimator := &MockTokenEstimator{}
	tokenLimitStrategy := optimization.NewTokenLimitStrategy(1000, estimator)
	optimizer.AddStrategy(tokenLimitStrategy)

	// Optimize again
	optimizedPrompt, err = optimizer.Optimize(prompt)
	if err != nil {
		t.Fatalf("Failed to optimize prompt with added strategy: %v", err)
	}

	// Check the optimized prompt
	if !strings.Contains(optimizedPrompt.Text, "Hello John") {
		t.Fatalf("Optimized prompt does not contain template replacement: %s", optimizedPrompt.Text)
	}

	if !strings.Contains(optimizedPrompt.SystemMessage, "think step by step") {
		t.Fatalf("Optimized system message does not contain chain of thought instructions: %s", optimizedPrompt.SystemMessage)
	}
}
