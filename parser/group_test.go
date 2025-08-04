package parser

import (
	"testing"
)

func TestCreateInitialGroups(t *testing.T) {
	// Create test processed logs
	logs := []*LogMessage{
		{ // Group A
			Words: []Word{
				{Value: "A", Position: 0, Frequency: 2},
				{Value: "common", Position: 1, Frequency: 2},
				{Value: "var1", Position: 2, Frequency: 1},
			},
		},
		{ // Group A
			Words: []Word{
				{Value: "A", Position: 0, Frequency: 2},
				{Value: "common", Position: 1, Frequency: 2},
				{Value: "var2", Position: 2, Frequency: 1},
			},
		},
		{ // Group B
			Words: []Word{
				{Value: "B", Position: 0, Frequency: 1},
				{Value: "another", Position: 1, Frequency: 1},
			},
		},
		{ // Log with different length
			Words: []Word{
				{Value: "C", Position: 0, Frequency: 1},
			},
		},
	}

	config := &Config{
		Weight: 0.0, // Test weight
	}
	groups := CreateInitialGroups(logs, config)

	if len(groups) != 3 {
		t.Fatalf("Expected 3 initial groups, got %d", len(groups))
	}

	// Check that logs are grouped correctly
	var groupA_found, groupB_found, groupC_found bool
	for _, group := range groups {
		if len(group.Logs) == 2 { // Group A
			if group.Pattern.Words[0].Value == "A" && group.Pattern.Words[1].Value == "common" {
				groupA_found = true
			}
		}
		if len(group.Logs) == 1 {
			if len(group.Pattern.Words) == 2 && group.Pattern.Words[0].Value == "B" { // Group B
				groupB_found = true
			}
			if len(group.Pattern.Words) == 1 && group.Pattern.Words[0].Value == "C" { // Group C
				groupC_found = true
			}
		}
	}

	if !groupA_found || !groupB_found || !groupC_found {
		t.Errorf("Failed to find all expected groups. A: %v, B: %v, C: %v", groupA_found, groupB_found, groupC_found)
	}
}
