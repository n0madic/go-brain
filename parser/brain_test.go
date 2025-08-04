package parser

import (
	"reflect"
	"sort"
	"testing"
)

func TestBrain_EndToEnd_Correctness(t *testing.T) {
	// Use a maximally simple and controlled data set.
	// This set is GUARANTEED to create 2 initial groups, which will then
	// turn into 3 final templates.
	logLines := []string{
		"event A happened", // Group 1
		"event B happened", // Group 1
		"event C happened", // Group 1
		"task X finished",  // Group 2
		"task Y finished",  // Group 2
	}

	config := Config{
		Delimiters:           `\s+`,
		ChildBranchThreshold: 2, // Threshold 2
	}

	parser := New(config)
	results := parser.Parse(logLines)

	// EXPECTED RESULT:
	// 1. The first initial group ("event..."): has 3 variants in the middle column (A,B,C).
	//    Since 3 > threshold (2), this column becomes a variable <*>. This gives 1 final template.
	// 2. The second initial group ("task..."): has 2 variants in the middle column (X,Y).
	//    Since 2 <= threshold (2), this column gives two constant branches. This gives 2 final templates.
	// TOTAL: 1 + 2 = 3 final templates.
	expected := []*ParseResult{
		{Template: "event <*> happened", Count: 3},
		{Template: "task X finished", Count: 1},
		{Template: "task Y finished", Count: 1},
	}

	// Sort for deterministic comparison
	sort.Slice(results, func(i, j int) bool { return results[i].Template < results[j].Template })
	sort.Slice(expected, func(i, j int) bool { return expected[i].Template < expected[j].Template })

	if len(results) != len(expected) {
		t.Logf("Got %d results:", len(results))
		for _, r := range results {
			t.Logf("- Template: '%s', Count: %d", r.Template, r.Count)
		}
		t.Fatalf("Expected %d templates, but got %d", len(expected), len(results))
	}

	for i := range results {
		if results[i].Template != expected[i].Template || results[i].Count != expected[i].Count {
			t.Errorf("Result mismatch at index %d.\nGot:  Template='%s', Count=%d\nWant: Template='%s', Count=%d",
				i, results[i].Template, results[i].Count, expected[i].Template, expected[i].Count)
		}
	}
}

func TestBrain_EndToEnd_PaperExample(t *testing.T) {
	// Logs from Figures 2 and 3 in the paper
	logLines := []string{
		"proxy.cse.cuhk.edu.hk:5070 open through proxy proxy.cse.cuhk.edu.hk:5070 HTTPS",
		"proxy.cse.cuhk.edu.hk:5070 close, 0 bytes sent, 0 bytes received, lifetime 00:01",
		"proxy.cse.cuhk.edu.hk:5070 open through proxy p3p.sogou.com:80 HTTPS",
		"proxy.cse.cuhk.edu.hk:5070 open through proxy 182.254.114.110:80 SOCKS5",
		"182.254.114.110:80 open through proxy 182.254.114.110:80 HTTPS",
		"proxy.cse.cuhk.edu.hk:5070 close, 403 bytes sent, 426 bytes received, lifetime 00:02",
		"get.sogou.com:80 close, 651 bytes sent, 546 bytes received, lifetime 00:03",
		"proxy.cse.cuhk.edu.hk:5070 close, 108 bytes sent, 411 bytes received, lifetime 00:03",
		"183.62.156.108:27 open through proxy socks.cse.cuhk.edu.hk:5070 SOCKS5",
		"proxy.cse.cuhk.edu.hk:5070 open through proxy proxy.cse.cuhk.edu.hk:5070 SOCKS5",
	}

	config := Config{
		Delimiters:             `[\s,]+`,
		ChildBranchThreshold:   1,
		UseDynamicThreshold:    true,
		DynamicThresholdFactor: 1.5,
	}

	parser := New(config)
	results := parser.Parse(logLines)

	expectedTemplates := []*ParseResult{
		{Template: "<*> open through proxy <*> HTTPS", Count: 3},
		{Template: "<*> open through proxy <*> SOCKS5", Count: 3},
		{Template: "<*> close <*> bytes sent <*> bytes received lifetime <*>", Count: 4},
	}

	// FIX: Use map for order-independent comparison.
	resultsMap := make(map[string]int)
	for _, r := range results {
		resultsMap[r.Template] = r.Count
	}

	expectedMap := make(map[string]int)
	for _, r := range expectedTemplates {
		expectedMap[r.Template] = r.Count
	}

	if !reflect.DeepEqual(resultsMap, expectedMap) {
		t.Logf("Detailed results:")
		for _, r := range results {
			t.Logf("- Template: '%s', Count: %d", r.Template, r.Count)
		}
		t.Errorf("Result mismatch.\nGot:  %v\nWant: %v", resultsMap, expectedMap)
	}
}

func TestBrain_EmptyInput(t *testing.T) {
	parser := New(Config{})
	results := parser.Parse([]string{})
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(results))
	}
}
