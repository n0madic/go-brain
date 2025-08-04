package parser

import (
	"reflect"
	"testing"
)

func TestPreprocessor_PreprocessLogs(t *testing.T) {
	logLines := []string{
		"Log 1: value1, value2",
		"Log 2: value1, value3",
	}
	preprocessor := NewPreprocessor(`[\s,:]+`, nil) // No common variables for simple test

	processed := preprocessor.PreprocessLogs(logLines)

	if len(processed) != 2 {
		t.Fatalf("Expected 2 processed logs, got %d", len(processed))
	}

	// Check first log
	expectedWords1 := []Word{
		{Value: "Log", Position: 0, Frequency: 2},
		{Value: "1", Position: 1, Frequency: 1},
		{Value: "value1", Position: 2, Frequency: 2},
		{Value: "value2", Position: 3, Frequency: 1},
	}
	if !reflect.DeepEqual(processed[0].Words, expectedWords1) {
		t.Errorf("Log 1 words mismatch.\nGot: %v\nWant: %v", processed[0].Words, expectedWords1)
	}

	// Check second log
	expectedWords2 := []Word{
		{Value: "Log", Position: 0, Frequency: 2},
		{Value: "2", Position: 1, Frequency: 1},
		{Value: "value1", Position: 2, Frequency: 2},
		{Value: "value3", Position: 3, Frequency: 1},
	}
	if !reflect.DeepEqual(processed[1].Words, expectedWords2) {
		t.Errorf("Log 2 words mismatch.\nGot: %v\nWant: %v", processed[1].Words, expectedWords2)
	}
}
