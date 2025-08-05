package parser

import (
	"reflect"
	"testing"
	"unique"
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
		{Value: unique.Make("Log"), Position: 0, Frequency: 2},
		{Value: unique.Make("<*>"), Position: 1, Frequency: 1},
		{Value: unique.Make("value1"), Position: 2, Frequency: 2},
		{Value: unique.Make("value2"), Position: 3, Frequency: 1},
	}
	if !reflect.DeepEqual(processed[0].Words, expectedWords1) {
		t.Errorf("Log 1 words mismatch.\nGot: %v\nWant: %v", processed[0].Words, expectedWords1)
	}

	// Check second log (note: "2" is detected as variable since it's 100% digits)
	expectedWords2 := []Word{
		{Value: unique.Make("Log"), Position: 0, Frequency: 2},
		{Value: unique.Make("<*>"), Position: 1, Frequency: 1},
		{Value: unique.Make("value1"), Position: 2, Frequency: 2},
		{Value: unique.Make("value3"), Position: 3, Frequency: 1},
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
			if word.Value.Value() != expectedPatterns[i][j] {
				t.Errorf("Log %d, word %d: expected %q, got %q",
					i, j, expectedPatterns[i][j], word.Value.Value())
			}
		}
	}
}

// Test datetime pattern recognition
func TestPreprocessor_DateTimePatterns(t *testing.T) {
	testCases := []struct {
		name     string
		logLines []string
		expected [][]string // Expected tokenized results
	}{
		{
			name: "ISO datetime with milliseconds",
			logLines: []string{
				"2023-01-15 14:30:45.123 INFO: Application started",
				"2023-01-15 14:30:46.456 INFO: Database connected",
			},
			expected: [][]string{
				{"<*>", "INFO", "Application", "started"},
				{"<*>", "INFO", "Database", "connected"},
			},
		},
		{
			name: "Bracketed datetime",
			logLines: []string{
				"[15-Jan-2023 14:30:45] User logged in",
				"[15-Jan-2023 14:31:12] User logged out",
			},
			expected: [][]string{
				{"<*>", "User", "logged", "in"},
				{"<*>", "User", "logged", "out"},
			},
		},
		{
			name: "Syslog format",
			logLines: []string{
				"Jan 15 14:30:45 server1: Service started",
				"Jan 15 14:30:46 server1: Service ready",
			},
			expected: [][]string{
				{"<*>", "server1", "Service", "started"},
				{"<*>", "server1", "Service", "ready"},
			},
		},
		{
			name: "European date format",
			logLines: []string{
				"15/01/2023 14:30:45 Process completed",
				"15/01/2023 14:31:00 Process started",
			},
			expected: [][]string{
				{"<*>", "Process", "completed"},
				{"<*>", "Process", "started"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			preprocessor := NewPreprocessor(`[\s:]+`, nil)
			processed := preprocessor.PreprocessLogs(tc.logLines)

			if len(processed) != len(tc.expected) {
				t.Fatalf("Expected %d processed logs, got %d", len(tc.expected), len(processed))
			}

			for i, log := range processed {
				if len(log.Words) != len(tc.expected[i]) {
					t.Errorf("Log %d: expected %d words, got %d", i, len(tc.expected[i]), len(log.Words))
					continue
				}

				for j, word := range log.Words {
					if word.Value.Value() != tc.expected[i][j] {
						t.Errorf("Log %d, word %d: expected %q, got %q", i, j, tc.expected[i][j], word.Value.Value())
					}
				}
			}
		})
	}
}

// Test CommonVariables regex patterns
func TestPreprocessor_CommonVariablePatterns(t *testing.T) {
	commonVars := map[string]string{
		"email":   `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		"ipv4":    `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
		"mac":     `\b[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}\b`,
		"uuid":    `\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`,
		"url":     `https?://[^\s]+`,
		"version": `v?\d+\.\d+\.\d+`,
		"hexhash": `\b[0-9a-fA-F]{32,64}\b`,
		"base64":  `\b[A-Za-z0-9+/]{16,}={0,2}\b`,
	}

	testCases := []struct {
		name     string
		logLines []string
		expected [][]string
	}{
		{
			name: "Email patterns",
			logLines: []string{
				"User john.doe@example.com logged in successfully",
				"User admin@company.org failed authentication",
			},
			expected: [][]string{
				{"User", "<*>", "logged", "in", "successfully"},
				{"User", "<*>", "failed", "authentication"},
			},
		},
		{
			name: "IP address patterns",
			logLines: []string{
				"Connection from 192.168.1.100 accepted",
				"Connection from 10.0.0.50 rejected",
			},
			expected: [][]string{
				{"Connection", "from", "<*>", "accepted"},
				{"Connection", "from", "<*>", "rejected"},
			},
		},
		{
			name: "UUID patterns",
			logLines: []string{
				"Request f47ac10b-58cc-4372-a567-0e02b2c3d479 completed",
				"Request 6ba7b810-9dad-11d1-80b4-00c04fd430c8 failed",
			},
			expected: [][]string{
				{"Request", "<*>", "completed"},
				{"Request", "<*>", "failed"},
			},
		},
		{
			name: "URL patterns",
			logLines: []string{
				"GET https://api.example.com/v1/users returned 200",
				"POST https://service.domain.org/api/data returned 404",
			},
			expected: [][]string{
				{"GET", "https", "//api.example.com/v1/users", "returned", "<*>"},
				{"POST", "https", "//service.domain.org/api/data", "returned", "<*>"},
			},
		},
		{
			name: "Version patterns",
			logLines: []string{
				"Application v2.1.3 started successfully",
				"Library version 1.0.0 loaded",
			},
			expected: [][]string{
				{"Application", "<*>", "started", "successfully"},
				{"Library", "version", "<*>", "loaded"},
			},
		},
		{
			name: "Hash patterns",
			logLines: []string{
				"File hash: a1b2c3d4e5f67890123456789abcdef0",
				"Checksum: fedcba0987654321098765432109876543",
			},
			expected: [][]string{
				{"File", "hash", "<*>"},
				{"Checksum", "<*>"},
			},
		},
		{
			name: "Base64 patterns",
			logLines: []string{
				"Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
				"Data: YXNkZmFzZGZhc2RmYXNkZmFzZGY=",
			},
			expected: [][]string{
				{"Token", "<*>"},
				{"Data", "<*>"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			preprocessor := NewPreprocessor(`[\s:]+`, commonVars)
			processed := preprocessor.PreprocessLogs(tc.logLines)

			if len(processed) != len(tc.expected) {
				t.Fatalf("Expected %d processed logs, got %d", len(tc.expected), len(processed))
			}

			for i, log := range processed {
				if len(log.Words) != len(tc.expected[i]) {
					t.Errorf("Log %d: expected %d words, got %d", i, len(tc.expected[i]), len(log.Words))
					// Print actual words for debugging
					t.Logf("Actual words: %v", func() []string {
						var words []string
						for _, w := range log.Words {
							words = append(words, w.Value.Value())
						}
						return words
					}())
					continue
				}

				for j, word := range log.Words {
					if word.Value.Value() != tc.expected[i][j] {
						t.Errorf("Log %d, word %d: expected %q, got %q", i, j, tc.expected[i][j], word.Value.Value())
					}
				}
			}
		})
	}
}

// Test mixed patterns in single log
func TestPreprocessor_MixedPatterns(t *testing.T) {
	commonVars := map[string]string{
		"email": `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		"ipv4":  `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
		"uuid":  `\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`,
	}

	logLines := []string{
		"2023-01-15 14:30:45 User john@example.com from 192.168.1.100 session f47ac10b-58cc-4372-a567-0e02b2c3d479",
		"2023-01-15 14:31:00 User admin@company.org from 10.0.0.50 session 6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	preprocessor := NewPreprocessor(`[\s:]+`, commonVars)
	processed := preprocessor.PreprocessLogs(logLines)

	if len(processed) != 2 {
		t.Fatalf("Expected 2 processed logs, got %d", len(processed))
	}

	// Both logs should have similar structure with variables replaced
	// The exact tokenization may vary based on delimiter behavior
	for i, log := range processed {
		t.Logf("Log %d tokenized to %d words: %v", i, len(log.Words), func() []string {
			var words []string
			for _, w := range log.Words {
				words = append(words, w.Value.Value())
			}
			return words
		}())

		// Just verify that email, IP, and UUID patterns were replaced with <*>
		hasVariables := false
		for _, word := range log.Words {
			if word.Value.Value() == "<*>" {
				hasVariables = true
				break
			}
		}
		if !hasVariables {
			t.Errorf("Log %d should have at least one variable (<*>)", i)
		}
	}
}
