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
		{Value: "<*>", Position: 1, Frequency: 1},
		{Value: "value1", Position: 2, Frequency: 2},
		{Value: "value2", Position: 3, Frequency: 1},
	}
	if !reflect.DeepEqual(processed[0].Words, expectedWords1) {
		t.Errorf("Log 1 words mismatch.\nGot: %v\nWant: %v", processed[0].Words, expectedWords1)
	}

	// Check second log (note: "2" is detected as variable since it's 100% digits)
	expectedWords2 := []Word{
		{Value: "Log", Position: 0, Frequency: 2},
		{Value: "<*>", Position: 1, Frequency: 1},
		{Value: "value1", Position: 2, Frequency: 2},
		{Value: "value3", Position: 3, Frequency: 1},
	}
	if !reflect.DeepEqual(processed[1].Words, expectedWords2) {
		t.Errorf("Log 2 words mismatch.\nGot: %v\nWant: %v", processed[1].Words, expectedWords2)
	}
}

func TestIsNumericVariable(t *testing.T) {
	tests := []struct {
		word     string
		expected bool
		desc     string
	}{
		// Should be detected as variables (>= 30% digits)
		{"user123", true, "alphanumeric with 3/7 digits (43%)"},
		{"id_456789", true, "id with 6/9 digits (67%)"},
		{"v2.3.4", true, "version with 3/6 digits (50%)"},
		{"192.168.1.1", true, "IP address with 9/11 digits (82%)"},
		{"abc123def456", true, "mixed with 6/12 digits (50%)"},
		{"12345", true, "pure numbers (100%)"},
		{"0xFF123", true, "hex with 5/7 digits (71%)"},

		// Should NOT be detected as variables (< 30% digits)
		{"username", false, "pure text (0%)"},
		{"error", false, "pure text (0%)"},
		{"log1", false, "mostly text with 1/4 digits (25%)"},
		{"v1", true, "version with 1/2 digits (50%)"},
		{"abc", false, "pure text (0%)"},
		{"user1a", false, "mostly text with 1/6 digits (17%)"},
		{"", false, "empty string"},
	}

	for _, test := range tests {
		result := isNumericVariable(test.word)
		if result != test.expected {
			t.Errorf("isNumericVariable(%q) = %v, want %v (%s)",
				test.word, result, test.expected, test.desc)
		}
	}
}

func TestPreprocessor_NumericVariableDetection(t *testing.T) {
	logLines := []string{
		"User user123 logged in from 192.168.1.100",
		"Error code ERR_404 occurred",
		"Processing batch job_456789 with id ABC123DEF",
	}

	// Test without any configured patterns
	preprocessor := NewPreprocessor(`[\s]+`, nil)
	processed := preprocessor.PreprocessLogs(logLines)

	// Check that numeric-heavy tokens are replaced with <*>
	expectedPatterns := [][]string{
		{"User", "<*>", "logged", "in", "from", "<*>"},      // user123 and IP should be variables
		{"Error", "code", "<*>", "occurred"},                // ERR_404 should be variable
		{"Processing", "batch", "<*>", "with", "id", "<*>"}, // job_456789 and ABC123DEF should be variables
	}

	for i, log := range processed {
		if len(log.Words) != len(expectedPatterns[i]) {
			t.Errorf("Log %d: expected %d words, got %d", i, len(expectedPatterns[i]), len(log.Words))
			continue
		}

		for j, word := range log.Words {
			if word.Value != expectedPatterns[i][j] {
				t.Errorf("Log %d, word %d: expected %q, got %q",
					i, j, expectedPatterns[i][j], word.Value)
			}
		}
	}
}
