package parser

import (
	"context"
	"testing"
	"time"
)

// TestOptimizedFeatures tests core optimized functionality
func TestOptimizedFeatures(t *testing.T) {
	config := Config{
		Delimiters:           `\s+`,
		ChildBranchThreshold: 3,
	}

	logs := []string{
		"User alice@example.com logged in from 192.168.1.100",
		"User bob@example.com logged in from 192.168.1.101",
		"System backup process started at 2024-01-15 10:30:00",
		"System backup process completed at 2024-01-15 10:35:00",
		"ERROR: Database connection failed after 30 seconds",
		"ERROR: Network timeout occurred during sync",
		"INFO: Application startup sequence completed successfully",
		"INFO: All services are running normally",
	}

	// Test adaptive processor
	processor := NewAdaptiveProcessor(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := processor.ProcessAdaptive(ctx, logs)
	if err != nil {
		t.Fatalf("Adaptive processing failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected results from adaptive processor")
	}

	// Verify correctness
	totalCount := 0
	for _, result := range results {
		totalCount += result.Count
		if result.Template == "" {
			t.Error("Empty template found")
		}
		if result.Count <= 0 {
			t.Error("Invalid count found")
		}
	}

	if totalCount != len(logs) {
		t.Errorf("Expected %d logs processed, got %d", len(logs), totalCount)
	}

	t.Logf("Optimized processing: %d templates for %d logs", len(results), len(logs))
}

// TestSIMDCapabilities tests SIMD capability detection
func TestSIMDCapabilities(t *testing.T) {
	caps := DetectSIMDCapabilities()
	t.Logf("SIMD Capabilities: Platform=%s, AVX2=%v, SSE42=%v, NEON=%v",
		caps.Platform, caps.HasAVX2, caps.HasSSE42, caps.HasNEON)

	// Test word counting
	counter := NewSIMDWordCounter()
	testCases := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello   world  ", 2},
		{"one two three four five", 5},
	}

	for _, tc := range testCases {
		count := counter.CountWords(tc.text)
		if count != tc.expected {
			t.Errorf("CountWords(%q): expected %d, got %d", tc.text, tc.expected, count)
		}
	}
}

// BenchmarkOptimizedParsing benchmarks the optimized parsing pipeline
func BenchmarkOptimizedParsing(b *testing.B) {
	config := Config{
		Delimiters:                `\s+`,
		ChildBranchThreshold:      3,
		UseEnhancedPostProcessing: true,
		UseStatisticalThreshold:   true,
	}

	logs := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		switch i % 4 {
		case 0:
			logs[i] = "User user" + string(rune('A'+(i%26))) + " logged in from 192.168.1." + string(rune('1'+(i%254)))
		case 1:
			logs[i] = "System process " + string(rune('A'+(i%26))) + " started with PID " + string(rune('1'+(i%9999)))
		case 2:
			logs[i] = "Database query " + string(rune('1'+(i%100))) + " executed in " + string(rune('1'+(i%1000))) + "ms"
		case 3:
			logs[i] = "HTTP request returned status " + string(rune('2'+(i%3))) + "00"
		}
	}

	processor := NewAdaptiveProcessor(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.ProcessAdaptive(ctx, logs)
	}
}
