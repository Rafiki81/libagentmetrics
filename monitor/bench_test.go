package monitor

import (
	"testing"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

var benchCursorResult cursorDBParseResult
var benchTokenString string

func BenchmarkParseCursorDBLines(b *testing.B) {
	b.ReportAllocs()
	lines := []string{
		`{"usageData":{"inputTokens":100,"outputTokens":50},"modelConfig":{"modelName":"gpt-4.1"},"conversationMap":{"a":{},"b":{}}}`,
		`{"usageData":{"inputTokens":20,"outputTokens":10},"modelConfig":{"modelName":"default,default,default,default"},"conversationMap":{"c":{}}}`,
		`invalid-json`,
		``,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCursorResult = parseCursorDBLines(lines)
	}
}

func BenchmarkFormatTokenCount(b *testing.B) {
	b.ReportAllocs()
	counts := []int64{0, 1, 999, 1_500, 500_000, 2_500_000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchTokenString = FormatTokenCount(counts[i%len(counts)])
	}
}

func BenchmarkAlertMonitorCheckNoAlert(b *testing.B) {
	b.ReportAllocs()
	th := DefaultThresholds()
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "bench-agent", Name: "Bench Agent"},
		CPU:    10,
		Memory: 100,
		Tokens: agent.TokenMetrics{TotalTokens: 100, EstCost: 0.001},
		Session: agent.SessionMetrics{
			LastActiveAt: time.Now(),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		am.Check(inst)
	}
}

func BenchmarkAlertMonitorCheckFleet(b *testing.B) {
	b.ReportAllocs()
	th := DefaultThresholds()
	th.DailyBudgetUSD = 10
	th.MonthlyBudgetUSD = 100
	am := NewAlertMonitor(th)

	agents := make([]agent.Instance, 0, 10)
	for i := 0; i < 10; i++ {
		agents = append(agents, agent.Instance{
			Info: agent.Info{ID: "agent", Name: "Agent"},
			Tokens: agent.TokenMetrics{
				TotalTokens: 10_000,
				EstCost:     0.75,
			},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		am.CheckFleet(agents)
	}
}
