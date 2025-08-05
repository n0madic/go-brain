package parser

import (
	"fmt"
	"strings"
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
		Delimiters:           `\s+`,
		ChildBranchThreshold: 2,
	}

	parser := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(logLines)
	}
}

// BenchmarkBrain_MediumDataset benchmarks parsing performance on medium datasets
func BenchmarkBrain_MediumDataset(b *testing.B) {
	// Generate 1000 log lines with variations
	var logLines []string
	patterns := []string{
		"User %d logged in from IP %d.%d.%d.%d",
		"System process %d started with PID %d",
		"Database query %d executed in %dms",
		"HTTP request %d returned status %d",
		"File %d.txt uploaded successfully",
		"Cache entry %d expired after %d seconds",
		"Error code %d occurred in module %d",
		"Session %d created for user %d",
	}

	for i := 0; i < 1000; i++ {
		pattern := patterns[i%len(patterns)]
		logLine := fmt.Sprintf(pattern, i%100, (i*7)%255, (i*11)%255, (i*13)%255, (i*17)%255)
		logLines = append(logLines, logLine)
	}

	config := Config{
		Delimiters:           `[\s.]+`,
		ChildBranchThreshold: 5,
	}

	parser := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(logLines)
	}
}

// BenchmarkBrain_LargeDataset benchmarks parsing performance on large datasets
func BenchmarkBrain_LargeDataset(b *testing.B) {
	// Generate 10000 log lines
	var logLines []string
	patterns := []string{
		"2023-01-15 %02d:%02d:%02d INFO User %d performed action %s",
		"2023-01-15 %02d:%02d:%02d ERROR System %d encountered error %d",
		"2023-01-15 %02d:%02d:%02d WARN Process %d memory usage %dMB",
		"2023-01-15 %02d:%02d:%02d DEBUG Query %d took %dms to execute",
	}

	actions := []string{"login", "logout", "upload", "download", "delete", "create", "update"}

	for i := 0; i < 10000; i++ {
		pattern := patterns[i%len(patterns)]
		hour := (i / 3600) % 24
		minute := (i / 60) % 60
		second := i % 60
		action := actions[i%len(actions)]

		logLine := fmt.Sprintf(pattern, hour, minute, second, i%1000, action, i%5000, (i*13)%1000)
		logLines = append(logLines, logLine)
	}

	config := Config{
		Delimiters:                  `[\s:]+`,
		ChildBranchThreshold:        10,
		ParallelProcessingThreshold: 1000,
	}

	parser := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(logLines)
	}
}

// BenchmarkBrain_EnhancedFeatures benchmarks enhanced features performance
func BenchmarkBrain_EnhancedFeatures(b *testing.B) {
	// Complex logs with hashes, UUIDs, emails, etc.
	var logLines []string
	for i := 0; i < 1000; i++ {
		logLines = append(logLines,
			fmt.Sprintf("User user%d@example.com session %08x-%04x-%04x-%04x-%012x logged in from %d.%d.%d.%d",
				i, i, i, i, i, i, i%255, (i*7)%255, (i*11)%255, (i*13)%255))
		logLines = append(logLines,
			fmt.Sprintf("Hash verification: %032x for file data_%d.bin size %dKB",
				i*31, i, i*13))
		logLines = append(logLines,
			fmt.Sprintf("API request to https://api.service%d.com/v1/endpoint%d returned %d",
				i%10, i%50, 200+(i%400)))
	}

	config := Config{
		Delimiters:                `[\s:@./]+`,
		ChildBranchThreshold:      5,
		UseEnhancedPostProcessing: true,
		UseStatisticalThreshold:   true,
	}

	parser := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(logLines)
	}
}

// BenchmarkBrain_Preprocessing benchmarks preprocessing performance specifically
func BenchmarkBrain_Preprocessing(b *testing.B) {
	var logLines []string
	for i := 0; i < 5000; i++ {
		logLines = append(logLines, fmt.Sprintf("2023-01-15 14:30:%02d.%03d INFO: User user%d@domain%d.com from %d.%d.%d.%d completed action_%d",
			i%60, i%1000, i, i%10, i%255, (i*7)%255, (i*11)%255, (i*13)%255, i%100))
	}

	preprocessor := NewPreprocessor(`[\s:@.]+`, map[string]string{
		"email": `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		"ipv4":  `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		preprocessor.PreprocessLogs(logLines)
	}
}

// BenchmarkBrain_TreeBuilding benchmarks tree building performance
func BenchmarkBrain_TreeBuilding(b *testing.B) {
	// Create a large log group for tree building
	var logs []*LogMessage
	for i := 0; i < 1000; i++ {
		logs = append(logs, &LogMessage{
			ID:      i,
			Content: fmt.Sprintf("Process %d completed task %s with status %d", i, []string{"A", "B", "C"}[i%3], i%10),
			Words: []Word{
				{Value: "Process", Position: 0, Frequency: 1000},
				{Value: fmt.Sprintf("%d", i), Position: 1, Frequency: 1},
				{Value: "completed", Position: 2, Frequency: 1000},
				{Value: "task", Position: 3, Frequency: 1000},
				{Value: []string{"A", "B", "C"}[i%3], Position: 4, Frequency: 333},
				{Value: "with", Position: 5, Frequency: 1000},
				{Value: "status", Position: 6, Frequency: 1000},
				{Value: fmt.Sprintf("%d", i%10), Position: 7, Frequency: 100},
			},
		})
	}

	group := &LogGroup{
		Pattern: LogPattern{
			Words: []Word{
				{Value: "Process", Position: 0, Frequency: 1000},
				{Value: "completed", Position: 2, Frequency: 1000},
				{Value: "task", Position: 3, Frequency: 1000},
				{Value: "with", Position: 5, Frequency: 1000},
				{Value: "status", Position: 6, Frequency: 1000},
			},
		},
		Logs: logs,
	}

	config := Config{ChildBranchThreshold: 10}
	parser := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.BuildTreeForGroup(group)
	}
}

// BenchmarkBrain_TemplateGeneration benchmarks template generation performance
func BenchmarkBrain_TemplateGeneration(b *testing.B) {
	// Create a complex tree structure
	var logs []*LogMessage
	for i := 0; i < 500; i++ {
		logs = append(logs, &LogMessage{ID: i})
	}

	// Create a tree with multiple branches
	children := make(map[string]*Node)
	for i := 0; i < 20; i++ {
		children[fmt.Sprintf("branch_%d", i)] = &Node{
			Position:   1,
			Value:      fmt.Sprintf("branch_%d", i),
			IsVariable: false,
			Logs:       logs[i*25 : (i+1)*25],
			Children:   make(map[string]*Node),
		}
	}

	tree := &BidirectionalTree{
		RootNodes: []Word{
			{Value: "System", Position: 0, Frequency: 500},
			{Value: "completed", Position: 2, Frequency: 500},
		},
		ParentDirection: make(map[int]*Node),
		ChildDirectionRoot: &Node{
			Value:    "ROOT",
			Children: children,
			Logs:     logs,
		},
	}

	config := Config{}
	parser := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.GenerateTemplatesFromTree(tree, logs)
	}
}

// BenchmarkBrain_ParallelProcessing benchmarks parallel processing performance
func BenchmarkBrain_ParallelProcessing(b *testing.B) {
	// Create a very large dataset to trigger parallel processing
	var logLines []string
	patterns := []string{
		"Worker %d processing job %d with priority %d",
		"Task %d completed by worker %d in %dms",
		"Queue %d has %d pending items",
		"Batch %d processed %d records successfully",
	}

	for i := 0; i < 5000; i++ {
		pattern := patterns[i%len(patterns)]
		logLine := fmt.Sprintf(pattern, i%100, i, i%10)
		logLines = append(logLines, logLine)
	}

	config := Config{
		Delimiters:                  `\s+`,
		ChildBranchThreshold:        10,
		ParallelProcessingThreshold: 500, // Lower threshold to trigger parallel processing
	}

	parser := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(logLines)
	}
}

// BenchmarkBrain_MemoryUsage is a simple memory usage benchmark
func BenchmarkBrain_MemoryUsage(b *testing.B) {
	// Generate a realistic dataset
	var logLines []string
	for i := 0; i < 10000; i++ {
		logLines = append(logLines, strings.Repeat(fmt.Sprintf("Long message %d with many words ", i), 10))
	}

	config := Config{
		Delimiters:           `\s+`,
		ChildBranchThreshold: 5,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parser := New(config)
		parser.Parse(logLines)
	}
}
