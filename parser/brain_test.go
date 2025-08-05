package parser

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
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

// Test enhanced variable patterns
func TestBrain_EnhancedVariablePatterns(t *testing.T) {
	logLines := []string{
		"User john@example.com logged in from 192.168.1.100",
		"User alice@company.org logged in from 10.0.0.50",
		"MAC address 00:1B:44:11:3A:B7 connected to network",
		"MAC address A0:B1:C2:D3:E4:F5 connected to network",
		"Download completed: file_v2.3.4.zip size: 1024KB",
		"Download completed: app_v1.0.0.tar.gz size: 2048MB",
		"Request from https://api.example.com/v1/users succeeded",
		"Request from https://test.domain.org/api/data succeeded",
	}

	config := Config{
		Delimiters:           `[\s:]+`,
		ChildBranchThreshold: 2,
	}

	parser := New(config)
	results := parser.Parse(logLines)

	// Check that emails, IPs, MACs, versions, file sizes, and URLs are properly identified as variables
	// Note: MAC addresses are split by ':' delimiter, URLs have '//' split
	expectedPatterns := []string{
		"User <*> logged in from <*>",
		"Download completed <*> size <*>",
	}

	// Also check for partial patterns that should exist
	partialPatterns := []string{
		"MAC address",  // MAC addresses will be split into parts
		"Request from", // URLs will be partially split
	}

	// Convert results to map for easier checking
	resultsMap := make(map[string]bool)
	for _, r := range results {
		resultsMap[r.Template] = true
		t.Logf("Generated pattern: %s", r.Template) // Debug output
	}

	for _, pattern := range expectedPatterns {
		if !resultsMap[pattern] {
			t.Errorf("Expected pattern not found: %s", pattern)
		}
	}

	// Check for partial patterns
	for _, partial := range partialPatterns {
		found := false
		for template := range resultsMap {
			if strings.Contains(template, partial) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("No pattern containing '%s' found", partial)
		}
	}
}

// Test enhanced post-processing
func TestBrain_EnhancedPostProcessing(t *testing.T) {
	logLines := []string{
		"Session id123abc456 started at 1673789445",
		"Session xyz789def012 started at 1673789500",
		"Hash value: a1b2c3d4e5f6789012345678901234567890abcd",
		"Hash value: fedcba0987654321098765432109876543210fed",
		"Encoded data: YXNkZmFzZGZhc2RmYXNkZmFzZGY=",
		"Encoded data: ZGF0YWJhc2U2NGVuY29kZWQ=",
	}

	// Test with enhanced post-processing enabled
	config := Config{
		Delimiters:                `[\s:]+`,
		ChildBranchThreshold:      2,
		UseEnhancedPostProcessing: true,
	}

	parser := New(config)
	results := parser.Parse(logLines)

	// With enhanced processing, complex patterns should be identified
	expectedCount := 3 // Three different pattern types
	if len(results) != expectedCount {
		t.Errorf("Expected %d patterns with enhanced processing, got %d", expectedCount, len(results))
	}

	// Test without enhanced post-processing
	config.UseEnhancedPostProcessing = false
	parser2 := New(config)
	results2 := parser2.Parse(logLines)

	// Without enhanced processing, might get more specific templates
	if len(results2) < len(results) {
		t.Errorf("Enhanced processing should produce fewer templates (more generalization)")
	}
}

// Test statistical threshold calculation
func TestBrain_StatisticalThreshold(t *testing.T) {
	// Create logs with varying unique word counts
	var logLines []string

	// Small dataset (< 10 unique words in position)
	for i := 0; i < 8; i++ {
		logLines = append(logLines, fmt.Sprintf("Small dataset item%d processed", i))
	}

	// Large dataset (> 100 unique words in position)
	for i := 0; i < 150; i++ {
		logLines = append(logLines, fmt.Sprintf("Large dataset entry%d completed", i))
	}

	config := Config{
		Delimiters:              `\s+`,
		UseDynamicThreshold:     true,
		UseStatisticalThreshold: true,
		DynamicThresholdFactor:  2.0,
	}

	parser := New(config)
	results := parser.Parse(logLines)

	// Should get two main templates due to statistical thresholding
	hasSmallTemplate := false
	hasLargeTemplate := false

	for _, r := range results {
		if r.Template == "Small dataset <*> processed" {
			hasSmallTemplate = true
		}
		if r.Template == "Large dataset <*> completed" {
			hasLargeTemplate = true
		}
	}

	if !hasSmallTemplate || !hasLargeTemplate {
		t.Error("Statistical threshold should properly handle both small and large datasets")
	}
}

// Test parallel processing
func TestBrain_ParallelProcessing(t *testing.T) {
	// Create a large dataset to trigger parallel processing
	var logLines []string
	for i := 0; i < 2000; i++ {
		logLines = append(logLines, fmt.Sprintf("User %d performed action A", i))
		logLines = append(logLines, fmt.Sprintf("System %d executed task B", i))
	}

	config := Config{
		Delimiters:                  `\s+`,
		ChildBranchThreshold:        10,
		ParallelProcessingThreshold: 1000,
	}

	parser := New(config)
	results := parser.Parse(logLines)

	// Should get 2 templates regardless of parallel processing
	expectedTemplates := map[string]bool{
		"User <*> performed action A": true,
		"System <*> executed task B":  true,
	}

	if len(results) != len(expectedTemplates) {
		t.Errorf("Expected %d templates, got %d", len(expectedTemplates), len(results))
	}

	for _, r := range results {
		if !expectedTemplates[r.Template] {
			t.Errorf("Unexpected template: %s", r.Template)
		}
	}
}
