package session

import (
	"time"
)

// GeminiSessionAnalytics holds metrics for a Gemini session
type GeminiSessionAnalytics struct {
	// Token usage
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`

	// Current context size (last turn's input + cache read tokens)
	CurrentContextTokens int `json:"current_context_tokens"`

	// Session metrics
	TotalTurns int           `json:"total_turns"`
	Duration   time.Duration `json:"duration"`
	StartTime  time.Time     `json:"start_time"`
	LastActive time.Time     `json:"last_active"`

	// Cost estimation
	EstimatedCost float64 `json:"estimated_cost"`

	// Model detected from session file messages
	Model string `json:"model,omitempty"`

	// In-memory cache: last file modification time (skip re-parse if unchanged)
	LastFileModTime time.Time `json:"-"`
}

// TotalTokens returns the sum of input and output tokens
func (a *GeminiSessionAnalytics) TotalTokens() int {
	return a.InputTokens + a.OutputTokens
}

// GeminiModelPricing holds pricing per million tokens
type GeminiModelPricing struct {
	Input  float64
	Output float64
}

// geminiPricing contains pricing per million tokens for each model (as of Jan 2025)
var geminiPricing = map[string]GeminiModelPricing{
	"gemini-1.5-flash": {Input: 0.075, Output: 0.30},
	"gemini-1.5-pro":   {Input: 3.50, Output: 10.50},
	"gemini-2.0-flash": {Input: 0.10, Output: 0.40},
	"gemini-2.5-flash": {Input: 0.15, Output: 0.60},
	"gemini-2.5-pro":   {Input: 1.25, Output: 10.00},
	// Fallback
	"default": {Input: 0.15, Output: 0.60},
}

// CalculateCost estimates session cost based on token usage and model pricing
func (a *GeminiSessionAnalytics) CalculateCost(model string) float64 {
	pricing, ok := geminiPricing[model]
	if !ok {
		pricing = geminiPricing["default"]
	}

	inputM := float64(a.InputTokens) / 1_000_000
	outputM := float64(a.OutputTokens) / 1_000_000

	return inputM*pricing.Input + outputM*pricing.Output
}
