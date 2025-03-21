package optimization

import (
	"strings"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// PromptOptimizer optimizes prompts for better results and efficiency
type PromptOptimizer struct {
	strategies []OptimizationStrategy
}

// OptimizationStrategy defines a strategy for optimizing prompts
type OptimizationStrategy interface {
	// Name returns the name of the strategy
	Name() string
	
	// Optimize optimizes a prompt
	Optimize(prompt *core.Prompt) (*core.Prompt, error)
}

// NewPromptOptimizer creates a new prompt optimizer with the given strategies
func NewPromptOptimizer(strategies ...OptimizationStrategy) *PromptOptimizer {
	return &PromptOptimizer{
		strategies: strategies,
	}
}

// AddStrategy adds a strategy to the optimizer
func (o *PromptOptimizer) AddStrategy(strategy OptimizationStrategy) {
	o.strategies = append(o.strategies, strategy)
}

// Optimize applies all strategies to optimize a prompt
func (o *PromptOptimizer) Optimize(prompt *core.Prompt) (*core.Prompt, error) {
	result := prompt
	var err error
	
	for _, strategy := range o.strategies {
		result, err = strategy.Optimize(result)
		if err != nil {
			return nil, err
		}
	}
	
	return result, nil
}

// TemplateStrategy replaces template variables in prompts
type TemplateStrategy struct {
	variables map[string]string
}

// NewTemplateStrategy creates a new template strategy
func NewTemplateStrategy(variables map[string]string) *TemplateStrategy {
	return &TemplateStrategy{
		variables: variables,
	}
}

// Name returns the name of the strategy
func (s *TemplateStrategy) Name() string {
	return "template"
}

// Optimize replaces template variables in the prompt
func (s *TemplateStrategy) Optimize(prompt *core.Prompt) (*core.Prompt, error) {
	result := *prompt
	
	// Replace variables in the text
	text := result.Text
	for key, value := range s.variables {
		text = strings.ReplaceAll(text, "{{"+key+"}}", value)
	}
	result.Text = text
	
	// Replace variables in the system message
	if result.SystemMessage != "" {
		systemMessage := result.SystemMessage
		for key, value := range s.variables {
			systemMessage = strings.ReplaceAll(systemMessage, "{{"+key+"}}", value)
		}
		result.SystemMessage = systemMessage
	}
	
	return &result, nil
}

// TokenLimitStrategy ensures prompts don't exceed token limits
type TokenLimitStrategy struct {
	maxTokens int
	estimator TokenEstimator
}

// TokenEstimator estimates the number of tokens in a text
type TokenEstimator interface {
	// EstimateTokens estimates the number of tokens in a text
	EstimateTokens(text string) int
}

// NewTokenLimitStrategy creates a new token limit strategy
func NewTokenLimitStrategy(maxTokens int, estimator TokenEstimator) *TokenLimitStrategy {
	return &TokenLimitStrategy{
		maxTokens: maxTokens,
		estimator: estimator,
	}
}

// Name returns the name of the strategy
func (s *TokenLimitStrategy) Name() string {
	return "token_limit"
}

// Optimize ensures the prompt doesn't exceed the token limit
func (s *TokenLimitStrategy) Optimize(prompt *core.Prompt) (*core.Prompt, error) {
	result := *prompt
	
	// Estimate tokens in the prompt
	totalTokens := s.estimator.EstimateTokens(result.Text)
	if result.SystemMessage != "" {
		totalTokens += s.estimator.EstimateTokens(result.SystemMessage)
	}
	
	// If we're under the limit, return the prompt as is
	if totalTokens <= s.maxTokens {
		return &result, nil
	}
	
	// If we're over the limit, truncate the prompt text
	// This is a simple strategy; more sophisticated strategies could be implemented
	excessTokens := totalTokens - s.maxTokens
	textTokens := s.estimator.EstimateTokens(result.Text)
	
	if excessTokens < textTokens {
		// Estimate how many characters to remove
		// This is a rough approximation; a better approach would use a proper tokenizer
		charsPerToken := len(result.Text) / textTokens
		charsToRemove := excessTokens * charsPerToken
		
		if charsToRemove < len(result.Text) {
			result.Text = result.Text[:len(result.Text)-charsToRemove]
		} else {
			// If we need to remove more than the text, truncate the system message too
			result.Text = ""
			systemTokens := s.estimator.EstimateTokens(result.SystemMessage)
			systemCharsPerToken := len(result.SystemMessage) / systemTokens
			systemCharsToRemove := (excessTokens - textTokens) * systemCharsPerToken
			
			if systemCharsToRemove < len(result.SystemMessage) {
				result.SystemMessage = result.SystemMessage[:len(result.SystemMessage)-systemCharsToRemove]
			} else {
				result.SystemMessage = ""
			}
		}
	}
	
	return &result, nil
}

// SimpleTokenEstimator is a simple implementation of TokenEstimator
type SimpleTokenEstimator struct{}

// EstimateTokens estimates the number of tokens in a text
// This is a very simple approximation; a real implementation would use a proper tokenizer
func (e *SimpleTokenEstimator) EstimateTokens(text string) int {
	// A very rough approximation: 1 token ≈ 4 characters
	return len(text) / 4
}

// ChainOfThoughtStrategy adds chain-of-thought prompting
type ChainOfThoughtStrategy struct{}

// NewChainOfThoughtStrategy creates a new chain-of-thought strategy
func NewChainOfThoughtStrategy() *ChainOfThoughtStrategy {
	return &ChainOfThoughtStrategy{}
}

// Name returns the name of the strategy
func (s *ChainOfThoughtStrategy) Name() string {
	return "chain_of_thought"
}

// Optimize adds chain-of-thought instructions to the prompt
func (s *ChainOfThoughtStrategy) Optimize(prompt *core.Prompt) (*core.Prompt, error) {
	result := *prompt
	
	// Add chain-of-thought instructions to the system message
	if result.SystemMessage != "" {
		result.SystemMessage += "\n\nPlease think step by step to solve this problem. Break down your reasoning into clear steps."
	} else {
		result.SystemMessage = "Please think step by step to solve this problem. Break down your reasoning into clear steps."
	}
	
	return &result, nil
}

// FewShotStrategy adds few-shot examples to the prompt
type FewShotStrategy struct {
	examples []FewShotExample
}

// FewShotExample represents a few-shot example
type FewShotExample struct {
	Input  string
	Output string
}

// NewFewShotStrategy creates a new few-shot strategy
func NewFewShotStrategy(examples []FewShotExample) *FewShotStrategy {
	return &FewShotStrategy{
		examples: examples,
	}
}

// Name returns the name of the strategy
func (s *FewShotStrategy) Name() string {
	return "few_shot"
}

// Optimize adds few-shot examples to the prompt
func (s *FewShotStrategy) Optimize(prompt *core.Prompt) (*core.Prompt, error) {
	result := *prompt
	
	// Build the few-shot examples
	var examplesText strings.Builder
	examplesText.WriteString("Here are some examples:\n\n")
	
	for i, example := range s.examples {
		examplesText.WriteString(fmt.Sprintf("Example %d:\nInput: %s\nOutput: %s\n\n", i+1, example.Input, example.Output))
	}
	
	examplesText.WriteString("Now, please solve the following:\n\n")
	
	// Add the examples before the prompt text
	result.Text = examplesText.String() + result.Text
	
	return &result, nil
}
