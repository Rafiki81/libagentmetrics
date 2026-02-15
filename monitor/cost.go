package monitor

import "fmt"

// ModelPricing holds pricing per 1M tokens for a model.
type ModelPricing struct {
	InputPer1M  float64
	OutputPer1M float64
}

// ModelPrices maps model name patterns to pricing (USD per 1M tokens).
var ModelPrices = map[string]ModelPricing{
	// OpenAI
	"gpt-4o":        {InputPer1M: 2.50, OutputPer1M: 10.00},
	"gpt-4o-mini":   {InputPer1M: 0.15, OutputPer1M: 0.60},
	"gpt-4-turbo":   {InputPer1M: 10.00, OutputPer1M: 30.00},
	"gpt-4":         {InputPer1M: 30.00, OutputPer1M: 60.00},
	"gpt-3.5-turbo": {InputPer1M: 0.50, OutputPer1M: 1.50},
	"o1":            {InputPer1M: 15.00, OutputPer1M: 60.00},
	"o1-mini":       {InputPer1M: 3.00, OutputPer1M: 12.00},
	"o1-pro":        {InputPer1M: 150.00, OutputPer1M: 600.00},
	"o3":            {InputPer1M: 10.00, OutputPer1M: 40.00},
	"o3-mini":       {InputPer1M: 1.10, OutputPer1M: 4.40},
	"codex":         {InputPer1M: 3.00, OutputPer1M: 12.00},

	// Anthropic
	"claude-opus-4":     {InputPer1M: 15.00, OutputPer1M: 75.00},
	"claude-sonnet-4":   {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-3.5-sonnet": {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-3-opus":     {InputPer1M: 15.00, OutputPer1M: 75.00},
	"claude-3-sonnet":   {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-3-haiku":    {InputPer1M: 0.25, OutputPer1M: 1.25},
	"claude-3.5-haiku":  {InputPer1M: 0.80, OutputPer1M: 4.00},

	// Google
	"gemini-2.0-flash": {InputPer1M: 0.10, OutputPer1M: 0.40},
	"gemini-1.5-pro":   {InputPer1M: 1.25, OutputPer1M: 5.00},
	"gemini-1.5-flash": {InputPer1M: 0.075, OutputPer1M: 0.30},

	// Fallback
	"default": {InputPer1M: 1.00, OutputPer1M: 3.00},
}

// EstimateCost calculates estimated cost based on model and token counts.
func EstimateCost(model string, inputTokens, outputTokens int64) float64 {
	pricing := FindPricing(model)
	inputCost := float64(inputTokens) / 1_000_000.0 * pricing.InputPer1M
	outputCost := float64(outputTokens) / 1_000_000.0 * pricing.OutputPer1M
	return inputCost + outputCost
}

// FindPricing returns the best matching pricing for a model name.
// It tries, in order: exact match, substring match, model-family fallback
// (claude, gpt-4, gemini), and finally the "default" entry.
func FindPricing(model string) ModelPricing {
	if model == "" {
		return ModelPrices["default"]
	}

	if p, ok := ModelPrices[model]; ok {
		return p
	}

	bestMatch := ""
	for key := range ModelPrices {
		if key == "default" {
			continue
		}
		if containsSubstr(model, key) || containsSubstr(key, model) {
			if len(key) > len(bestMatch) {
				bestMatch = key
			}
		}
	}

	if bestMatch != "" {
		return ModelPrices[bestMatch]
	}

	if containsSubstr(model, "claude") {
		if containsSubstr(model, "opus") {
			return ModelPrices["claude-opus-4"]
		}
		if containsSubstr(model, "haiku") {
			return ModelPrices["claude-3-haiku"]
		}
		return ModelPrices["claude-sonnet-4"]
	}
	if containsSubstr(model, "gpt-4") {
		if containsSubstr(model, "mini") {
			return ModelPrices["gpt-4o-mini"]
		}
		return ModelPrices["gpt-4o"]
	}
	if containsSubstr(model, "gemini") {
		return ModelPrices["gemini-2.0-flash"]
	}

	return ModelPrices["default"]
}

func containsSubstr(s, substr string) bool {
	if len(substr) == 0 {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			a, b := s[i+j], substr[j]
			if a != b && a != b+32 && a != b-32 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// FormatCost formats a USD cost value for display.
// Returns "—" for zero/negative, "<$0.01" for sub-cent amounts,
// or "$X.XX" otherwise.
func FormatCost(cost float64) string {
	if cost <= 0 {
		return "—"
	}
	if cost < 0.01 {
		return "<$0.01"
	}
	return fmt.Sprintf("$%.2f", cost)
}
