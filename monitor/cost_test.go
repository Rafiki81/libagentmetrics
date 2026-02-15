package monitor

import (
	"testing"
)

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		input   int64
		output  int64
		wantMin float64
		wantMax float64
	}{
		{
			name:    "small usage",
			model:   "gpt-4o-mini",
			input:   1000,
			output:  500,
			wantMin: 0,
			wantMax: 0.001,
		},
		{
			name:    "gpt-4o 1M tokens",
			model:   "gpt-4o",
			input:   500000,
			output:  500000,
			wantMin: 5.0,
			wantMax: 7.0,
		},
		{
			name:    "claude-sonnet-4 usage",
			model:   "claude-sonnet-4",
			input:   100000,
			output:  50000,
			wantMin: 0.9,
			wantMax: 1.1,
		},
		{
			name:    "zero tokens",
			model:   "gpt-4o",
			input:   0,
			output:  0,
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "unknown model uses default",
			model:   "totally-unknown-model",
			input:   1000000,
			output:  1000000,
			wantMin: 3.5,
			wantMax: 4.5, // 1.00 + 3.00
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimateCost(tt.model, tt.input, tt.output)
			if cost < tt.wantMin || cost > tt.wantMax {
				t.Errorf("EstimateCost(%q, %d, %d) = %f, want between %f and %f",
					tt.model, tt.input, tt.output, cost, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestFindPricing_ExactMatch(t *testing.T) {
	tests := []struct {
		model      string
		wantInput  float64
		wantOutput float64
	}{
		{"gpt-4o", 2.50, 10.00},
		{"gpt-4o-mini", 0.15, 0.60},
		{"gpt-4", 30.00, 60.00},
		{"claude-opus-4", 15.00, 75.00},
		{"claude-sonnet-4", 3.00, 15.00},
		{"claude-3-haiku", 0.25, 1.25},
		{"gemini-2.0-flash", 0.10, 0.40},
		{"gemini-1.5-pro", 1.25, 5.00},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			p := FindPricing(tt.model)
			if p.InputPer1M != tt.wantInput {
				t.Errorf("InputPer1M = %f, want %f", p.InputPer1M, tt.wantInput)
			}
			if p.OutputPer1M != tt.wantOutput {
				t.Errorf("OutputPer1M = %f, want %f", p.OutputPer1M, tt.wantOutput)
			}
		})
	}
}

func TestFindPricing_EmptyModel(t *testing.T) {
	p := FindPricing("")
	defaultP := ModelPrices["default"]
	if p.InputPer1M != defaultP.InputPer1M {
		t.Errorf("empty model InputPer1M = %f, want %f", p.InputPer1M, defaultP.InputPer1M)
	}
}

func TestFindPricing_FuzzyMatch(t *testing.T) {
	// Models that should match via substring matching
	tests := []struct {
		model       string
		expectInput float64
	}{
		{"claude-opus-4-latest", 15.00}, // Contains "claude" and "opus"
		{"gpt-4o-2024-01-01", 2.50},     // Contains "gpt-4o"
		{"some-gpt-4-variant", 30.00},   // Contains "gpt-4"
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			p := FindPricing(tt.model)
			if p.InputPer1M != tt.expectInput {
				t.Errorf("FindPricing(%q).InputPer1M = %f, want %f", tt.model, p.InputPer1M, tt.expectInput)
			}
		})
	}
}

func TestFindPricing_FamilyFallback(t *testing.T) {
	// Test family-level fallback for claude
	p := FindPricing("claude-future-model")
	if p.InputPer1M != 3.00 { // Falls back to claude-sonnet-4
		t.Errorf("claude fallback InputPer1M = %f, want 3.00", p.InputPer1M)
	}

	// Gemini fallback
	p = FindPricing("gemini-99-turbo")
	if p.InputPer1M != 0.10 { // Falls back to gemini-2.0-flash
		t.Errorf("gemini fallback InputPer1M = %f, want 0.10", p.InputPer1M)
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		cost float64
		want string
	}{
		{0, "—"},
		{-1, "—"},
		{0.001, "<$0.01"},
		{0.005, "<$0.01"},
		{0.01, "$0.01"},
		{0.50, "$0.50"},
		{1.234, "$1.23"},
		{99.99, "$99.99"},
		{100.0, "$100.00"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatCost(tt.cost)
			if got != tt.want {
				t.Errorf("FormatCost(%f) = %q, want %q", tt.cost, got, tt.want)
			}
		})
	}
}

func TestModelPrices_HasDefault(t *testing.T) {
	if _, ok := ModelPrices["default"]; !ok {
		t.Error("ModelPrices is missing 'default' entry")
	}
}

func TestModelPrices_AllPositive(t *testing.T) {
	for model, p := range ModelPrices {
		if p.InputPer1M < 0 {
			t.Errorf("model %q has negative InputPer1M: %f", model, p.InputPer1M)
		}
		if p.OutputPer1M < 0 {
			t.Errorf("model %q has negative OutputPer1M: %f", model, p.OutputPer1M)
		}
	}
}
