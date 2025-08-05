package parser

import (
	"context"
	"strings"
	"testing"
)

// TestStreamingProcessorPointerSafe verifies streaming processor works with pointer-safe buffers
func TestStreamingProcessorPointerSafe(t *testing.T) {
	config := Config{
		Delimiters:           `\s+`,
		ChildBranchThreshold: 2,
	}

	streamConfig := StreamingConfig{
		BatchSize:  3,
		MaxWorkers: 2,
	}

	processor := NewStreamingProcessor(config, streamConfig)

	logs := []string{
		"User alice logged in successfully",
		"User bob logged in successfully",
		"User charlie logged in successfully",
		"User david failed to login",
		"User eve failed to login",
		"System backup completed successfully",
	}

	ctx := context.Background()
	results, err := processor.ProcessLargeSlice(ctx, logs)
	if err != nil {
		t.Fatalf("StreamingProcessor failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected results from streaming processor")
	}

	// Verify all logs are processed
	totalCount := 0
	for _, result := range results {
		totalCount += result.Count
	}

	if totalCount != len(logs) {
		t.Errorf("Expected %d logs processed, got %d", len(logs), totalCount)
	}
}

// TestStreamingProcessorWithReader tests the ProcessReader functionality
func TestStreamingProcessorWithReader(t *testing.T) {
	config := Config{
		Delimiters:           `\s+`,
		ChildBranchThreshold: 2,
	}

	streamConfig := StreamingConfig{
		BatchSize:  2,
		MaxWorkers: 2,
	}

	processor := NewStreamingProcessor(config, streamConfig)

	// Create a reader with test data
	logData := `User alice logged in successfully
User bob logged in successfully
User charlie failed to login
System backup completed
System startup finished`

	reader := strings.NewReader(logData)
	ctx := context.Background()

	results, err := processor.ProcessReader(ctx, reader)
	if err != nil {
		t.Fatalf("ProcessReader failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected results from ProcessReader")
	}

	// Verify processing worked
	totalCount := 0
	for _, result := range results {
		totalCount += result.Count
		if result.Template == "" {
			t.Error("Empty template found")
		}
	}

	if totalCount != 5 { // 5 log lines in the test data
		t.Errorf("Expected 5 logs processed, got %d", totalCount)
	}
}
