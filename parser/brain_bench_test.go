package parser

import (
	"context"
	"testing"
)

// BenchmarkBrain_SmallDataset benchmarks parsing performance on small datasets
func BenchmarkBrain_SmallDataset(b *testing.B) {
	logLines := []string{
		"User alice logged in successfully",
		"User bob logged in successfully",
		"User charlie logged in successfully",
		"User david failed to login",
		"User eve failed to login",
		"System backup completed successfully",
		"System backup started",
		"Database connection established",
		"Database connection failed",
		"Application started on port 8080",
	}

	config := Config{
		Delimiters:                `\s+`,
		ChildBranchThreshold:      2,
		UseEnhancedPostProcessing: true,
	}

	processor := NewAdaptiveProcessor(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.ProcessAdaptive(ctx, logLines)
	}
}

// BenchmarkBrain_LargeDataset benchmarks parsing performance on large datasets
func BenchmarkBrain_LargeDataset(b *testing.B) {
	// Generate test data that triggers streaming processing
	logLines := make([]string, 6000)

	patterns := []string{
		"User user%d logged in from 192.168.1.%d",
		"System process proc%d started with PID %d",
		"Database query%d executed in %dms",
		"HTTP request %d returned status %d",
		"ERROR: Connection timeout on server%d port %d",
	}

	for i := 0; i < 6000; i++ {
		pattern := patterns[i%len(patterns)]
		switch pattern {
		case patterns[0]:
			logLines[i] = "User user" + string(rune('A'+(i%26))) + " logged in from 192.168.1." + string(rune('1'+(i%254)))
		case patterns[1]:
			logLines[i] = "System process proc" + string(rune('A'+(i%26))) + " started with PID " + string(rune('1'+(i%9999)))
		case patterns[2]:
			logLines[i] = "Database query" + string(rune('1'+(i%100))) + " executed in " + string(rune('1'+(i%1000))) + "ms"
		case patterns[3]:
			logLines[i] = "HTTP request " + string(rune('1'+(i%1000))) + " returned status " + string(rune('2'+(i%3))) + "00"
		case patterns[4]:
			logLines[i] = "ERROR: Connection timeout on server" + string(rune('A'+(i%10))) + " port " + string(rune('8'+(i%10))) + "080"
		}
	}

	config := Config{
		Delimiters:                  `[\s:]+`,
		ChildBranchThreshold:        10,
		ParallelProcessingThreshold: 1000,
		UseEnhancedPostProcessing:   true,
		UseStatisticalThreshold:     true,
	}

	processor := NewAdaptiveProcessor(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, _ := processor.ProcessAdaptive(ctx, logLines)
		_ = results
	}
}

// BenchmarkBrain_MemoryOptimized benchmarks memory-optimized operations
func BenchmarkBrain_MemoryOptimized(b *testing.B) {
	logLines := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		logLines[i] = "Process " + string(rune('A'+(i%26))) + " executed task " + string(rune('1'+(i%10))) + " successfully at " + string(rune('0'+(i%24))) + ":00"
	}

	config := Config{
		Delimiters:                `\s+`,
		ChildBranchThreshold:      5,
		UseEnhancedPostProcessing: true,
	}

	processor := NewAdaptiveProcessor(config)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		results, _ := processor.ProcessAdaptive(ctx, logLines)
		_ = results
	}
}
