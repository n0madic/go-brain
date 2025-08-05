package parser

import (
	"testing"
	"unique"
)

func TestCreateInitialGroups(t *testing.T) {
	// Create test processed logs
	logs := []*LogMessage{
		{ // Group A
			Words: []Word{
				{Value: unique.Make("A"), Position: 0, Frequency: 2},
				{Value: unique.Make("common"), Position: 1, Frequency: 2},
				{Value: unique.Make("var1"), Position: 2, Frequency: 1},
			},
		},
		{ // Group A
			Words: []Word{
				{Value: unique.Make("A"), Position: 0, Frequency: 2},
				{Value: unique.Make("common"), Position: 1, Frequency: 2},
				{Value: unique.Make("var2"), Position: 2, Frequency: 1},
			},
		},
		{ // Group B
			Words: []Word{
				{Value: unique.Make("B"), Position: 0, Frequency: 1},
				{Value: unique.Make("another"), Position: 1, Frequency: 1},
			},
		},
		{ // Log with different length
			Words: []Word{
				{Value: unique.Make("C"), Position: 0, Frequency: 1},
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
	var groupAFound, groupBFound, groupCFound bool
	for _, group := range groups {
		if len(group.Logs) == 2 { // Group A
			if group.Pattern.Words[0].Value.Value() == "A" && group.Pattern.Words[1].Value.Value() == "common" {
				groupAFound = true
			}
		}
		if len(group.Logs) == 1 {
			if len(group.Pattern.Words) == 2 && group.Pattern.Words[0].Value.Value() == "B" { // Group B
				groupBFound = true
			}
			if len(group.Pattern.Words) == 1 && group.Pattern.Words[0].Value.Value() == "C" { // Group C
				groupCFound = true
			}
		}
	}

	if !groupAFound || !groupBFound || !groupCFound {
		t.Errorf("Failed to find all expected groups. A: %v, B: %v, C: %v", groupAFound, groupBFound, groupCFound)
	}
}
