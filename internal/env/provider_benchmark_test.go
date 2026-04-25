package env

import (
	"context"
	"testing"
)

func BenchmarkExecuteProvider_Get(b *testing.B) {
	p := NewPlugin()
	ctx := context.Background()

	b.Setenv("BENCH_TEST_VAR", "benchmark-value")

	inputs := map[string]any{
		"operation": "get",
		"name":      "BENCH_TEST_VAR",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = p.ExecuteProvider(ctx, "env", inputs)
	}
}

func BenchmarkExecuteProvider_List(b *testing.B) {
	p := NewPlugin()
	ctx := context.Background()

	inputs := map[string]any{
		"operation": "list",
		"prefix":    "BENCH_",
	}

	b.Setenv("BENCH_A", "1")
	b.Setenv("BENCH_B", "2")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = p.ExecuteProvider(ctx, "env", inputs)
	}
}
