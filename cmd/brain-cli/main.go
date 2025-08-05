// Package main provides a command-line interface for the Brain log parser.
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/n0madic/go-brain/parser"
)

const (
	defaultDelimiters             = `[\s,:=]+`
	defaultChildBranchThreshold   = 3
	defaultDynamicThresholdFactor = 2.0
)

func main() {
	var (
		inputFile     = flag.String("input", "", "Input file path (required)")
		fileType      = flag.String("type", "auto", "File type: auto, text, csv")
		csvColumn     = flag.String("csv-column", "message", "CSV column name containing log messages")
		delimiters    = flag.String("delimiters", defaultDelimiters, "Regex pattern for token delimiters")
		threshold     = flag.Int("threshold", defaultChildBranchThreshold, "Child branch threshold")
		useDynamic    = flag.Bool("dynamic", true, "Use dynamic threshold calculation")
		dynamicFactor = flag.Float64("dynamic-factor", defaultDynamicThresholdFactor, "Dynamic threshold factor")
		verbose       = flag.Bool("verbose", false, "Verbose output with log IDs")
		outputFormat  = flag.String("format", "table", "Output format: table, json, csv")
		minCount      = flag.Int("min-count", 1, "Minimum template count to display")
		logRegex      = flag.String("log-regex", "", "Regex to extract message from structured logs (must have 'message' capture group)")
	)
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: input file is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Read input file
	logLines, err := readInputFile(*inputFile, *fileType, *csvColumn, *logRegex)
	if err != nil {
		log.Fatalf("Error reading input file: %v", err)
	}

	if len(logLines) == 0 {
		fmt.Println("No log lines found in input file")
		return
	}

	fmt.Printf("Processing %d log lines...\n", len(logLines))

	// Configure Brain parser
	config := parser.Config{
		Delimiters:             *delimiters,
		ChildBranchThreshold:   *threshold,
		UseDynamicThreshold:    *useDynamic,
		DynamicThresholdFactor: *dynamicFactor,
		Weight:                 0.0, // Offline mode
	}

	// Create parser and process logs
	brainParser := parser.New(config)
	results := brainParser.Parse(logLines)

	// Filter results by minimum count
	var filteredResults []*parser.ParseResult
	for _, result := range results {
		if result.Count >= *minCount {
			filteredResults = append(filteredResults, result)
		}
	}

	fmt.Printf("Found %d unique templates (showing %d with count >= %d):\n\n",
		len(results), len(filteredResults), *minCount)

	// Output results in specified format
	switch *outputFormat {
	case "json":
		outputJSON(filteredResults, *verbose)
	case "csv":
		outputCSV(filteredResults, *verbose)
	default:
		outputTable(filteredResults, *verbose)
	}
}

// readInputFile reads log lines from various file formats
func readInputFile(filename, fileType, csvColumn, logRegex string) ([]string, error) {
	file, err := os.Open(filename) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", closeErr)
		}
	}()

	// Auto-detect file type if not specified
	if fileType == "auto" {
		if strings.HasSuffix(strings.ToLower(filename), ".csv") {
			fileType = "csv"
		} else {
			fileType = "text"
		}
	}

	switch fileType {
	case "csv":
		return readCSVFile(file, csvColumn)
	case "text":
		return readTextFile(file, logRegex)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// readTextFile reads plain text log files (one log per line)
func readTextFile(reader io.Reader, logRegex string) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(reader)

	// Compile regex if provided
	var regex *regexp.Regexp
	var err error
	if logRegex != "" {
		regex, err = regexp.Compile(logRegex)
		if err != nil {
			return nil, fmt.Errorf("invalid log regex: %w", err)
		}
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" { // Skip empty lines
			continue
		}

		// Extract message using regex if provided
		if regex != nil {
			matches := regex.FindStringSubmatch(line)
			if len(matches) > 1 {
				// Look for named capture group "message"
				names := regex.SubexpNames()
				for i, name := range names {
					if name == "message" && i < len(matches) {
						line = matches[i]
						break
					}
				}
			} else {
				// If regex doesn't match, skip the line
				continue
			}
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading text file: %w", err)
	}

	return lines, nil
}

// readCSVFile reads CSV files and extracts the specified message column
func readCSVFile(reader io.Reader, columnName string) ([]string, error) {
	csvReader := csv.NewReader(reader)

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV header: %w", err)
	}

	// Find the message column index
	messageIndex := -1
	for i, col := range header {
		if strings.EqualFold(strings.TrimSpace(col), columnName) {
			messageIndex = i
			break
		}
	}

	if messageIndex == -1 {
		return nil, fmt.Errorf("column '%s' not found in CSV. Available columns: %v",
			columnName, header)
	}

	// Read all records
	var lines []string
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV record: %w", err)
		}

		if messageIndex < len(record) {
			message := strings.TrimSpace(record[messageIndex])
			if message != "" { // Skip empty messages
				lines = append(lines, message)
			}
		}
	}

	return lines, nil
}

// outputTable outputs results in a formatted table
func outputTable(results []*parser.ParseResult, verbose bool) {
	fmt.Printf("%-6s %-80s", "COUNT", "TEMPLATE")
	if verbose {
		fmt.Printf(" %s", "LOG_IDS")
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 86+func() int {
		if verbose {
			return 20
		}
		return 0
	}()))

	for _, result := range results {
		fmt.Printf("%-6d %-80s", result.Count, result.Template)
		if verbose {
			fmt.Printf(" %v", result.LogIDs)
		}
		fmt.Println()
	}
}

// outputJSON outputs results in JSON format
func outputJSON(results []*parser.ParseResult, verbose bool) {
	fmt.Println("[")
	for i, result := range results {
		fmt.Printf("  {\n")
		fmt.Printf("    \"template\": \"%s\",\n", escapeJSON(result.Template))
		fmt.Printf("    \"count\": %d", result.Count)
		if verbose {
			fmt.Printf(",\n    \"log_ids\": %v", result.LogIDs)
		}
		fmt.Printf("\n  }")
		if i < len(results)-1 {
			fmt.Printf(",")
		}
		fmt.Println()
	}
	fmt.Println("]")
}

// outputCSV outputs results in CSV format
func outputCSV(results []*parser.ParseResult, verbose bool) {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"template", "count"}
	if verbose {
		header = append(header, "log_ids")
	}
	if err := writer.Write(header); err != nil {
		log.Printf("Error writing CSV header: %v", err)
	}

	// Write data
	for _, result := range results {
		record := []string{result.Template, fmt.Sprintf("%d", result.Count)}
		if verbose {
			record = append(record, fmt.Sprintf("%v", result.LogIDs))
		}
		if err := writer.Write(record); err != nil {
			log.Printf("Error writing CSV record: %v", err)
		}
	}
}

// escapeJSON escapes special characters for JSON output
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
