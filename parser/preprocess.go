package parser

import (
	"regexp"
	"strings"
	"unique"
)

// DateTime preprocessing constants
const (
	dtSpacePlaceholder = "_DTSPACE_"
	dtColonPlaceholder = "_DTCOLON_"
	dtCommaPlaceholder = "_DTCOMMA_"
	dtEqualPlaceholder = "_DTEQUAL_"
)

// Pre-compiled datetime patterns for performance
var dateTimePatterns = []struct {
	pattern *regexp.Regexp
	name    string
}{
	// Bracketed datetime formats (most specific first)
	// [DD-MMM-YYYY HH:mm:ss] format (like [31-Jul-2025 01:17:58])
	{regexp.MustCompile(`\[\d{1,2}-[A-Z][a-z]{2}-\d{4} \d{2}:\d{2}:\d{2}\]`), "bracketed_datetime_full"},

	// ISO datetime with milliseconds and space
	// YYYY-MM-DD HH:mm:ss.SSS format (like 2024-01-15 10:30:15.123)
	{regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}`), "iso_datetime_ms"},

	// ISO datetime with space
	// YYYY-MM-DD HH:mm:ss format (like 2024-01-15 10:30:15)
	{regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`), "iso_datetime_space"},

	// European datetime formats
	// DD/MM/YYYY HH:mm:ss format (like 15/01/2024 10:30:15)
	{regexp.MustCompile(`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}`), "european_datetime"},

	// US datetime formats
	// MM/DD/YYYY HH:mm:ss format (like 01/15/2024 10:30:15)
	{regexp.MustCompile(`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}`), "us_datetime"},

	// Year-first slash format
	// YYYY/MM/DD HH:mm:ss format (like 2025/07/31 01:17:58)
	{regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`), "slash_datetime"},

	// Syslog format - very common in system logs
	// MMM DD HH:mm:ss format (like Jan 15 10:30:15, Jul 31 01:17:58)
	{regexp.MustCompile(`[A-Z][a-z]{2} +\d{1,2} +\d{2}:\d{2}:\d{2}`), "syslog_datetime"},

	// Extended syslog with year
	// MMM DD YYYY HH:mm:ss format (like Jan 15 2024 10:30:15)
	{regexp.MustCompile(`[A-Z][a-z]{2} +\d{1,2} +\d{4} +\d{2}:\d{2}:\d{2}`), "syslog_with_year"},

	// Apache/Nginx log format
	// DD/MMM/YYYY HH:mm:ss format (like 15/Jan/2024 10:30:15)
	{regexp.MustCompile(`\d{1,2}/[A-Z][a-z]{2}/\d{4} \d{2}:\d{2}:\d{2}`), "apache_datetime"},

	// Dotted date format
	// DD.MM.YYYY HH:mm:ss format (like 15.01.2024 10:30:15)
	{regexp.MustCompile(`\d{2}\.\d{2}\.\d{4} \d{2}:\d{2}:\d{2}`), "dotted_datetime"},
}

// Preprocessor contains logic for log preprocessing.
type Preprocessor struct {
	delimiters      *regexp.Regexp
	commonVariables map[string]*regexp.Regexp // Compiled regex for common variables
}

// NewPreprocessor creates a new preprocessor.
func NewPreprocessor(delimiters string, commonVariables map[string]string) *Preprocessor {
	compiledVariables := make(map[string]*regexp.Regexp)
	for name, pattern := range commonVariables {
		compiledVariables[name] = regexp.MustCompile(pattern)
	}

	return &Preprocessor{
		delimiters:      regexp.MustCompile(delimiters),
		commonVariables: compiledVariables,
	}
}

// PreprocessLogs performs full preprocessing of a set of log lines.
func (p *Preprocessor) PreprocessLogs(logLines []string) []*LogMessage {
	// 1. Preprocess datetime patterns to protect spaces within them
	preprocessedLines := make([]string, len(logLines))
	for i, line := range logLines {
		preprocessedLines[i] = preprocessDateTimePatterns(line)
	}

	// 2. Split logs without filtering to get original words
	wordFrequencies := make(map[string]int)
	var rawSplitLogs [][]string
	for _, line := range preprocessedLines {
		words := p.splitWithoutFiltering(line)
		rawSplitLogs = append(rawSplitLogs, words)
		for _, word := range words {
			wordFrequencies[word]++
		}
	}

	// 3. Create LogMessage structures, applying filtering while preserving original frequencies
	processedLogs := make([]*LogMessage, len(logLines))
	for i, rawWords := range rawSplitLogs {
		// Use pooled LogMessage
		logMessage := GetLogMessage()
		logMessage.ID = i
		logMessage.Content = unique.Make(logLines[i]) // Intern the content string

		// Use pooled word slice if available, otherwise allocate
		if logMessage.Words == nil || cap(logMessage.Words) < len(rawWords) {
			if logMessage.Words != nil {
				PutWordSlice(logMessage.Words) // Return previous slice to pool
			}
			logMessage.Words = GetWordSlice()
		}

		// Ensure sufficient capacity
		if cap(logMessage.Words) < len(rawWords) {
			logMessage.Words = make([]Word, len(rawWords))
		} else {
			logMessage.Words = logMessage.Words[:len(rawWords)]
		}

		for j, rawWord := range rawWords {
			// Apply common variable filtering to the word value
			filteredWord := p.filterCommonVariables(rawWord)
			logMessage.Words[j] = Word{
				Value:     unique.Make(filteredWord), // Intern the word value
				Position:  j,
				Frequency: wordFrequencies[rawWord], // Use original word frequency
			}
		}
		processedLogs[i] = logMessage
	}

	return processedLogs
}

// splitWithoutFiltering divides a string into words using given delimiters without applying variable filtering.
func (p *Preprocessor) splitWithoutFiltering(line string) []string {
	// Replace all delimiters with one (space) and then split
	normalized := p.delimiters.ReplaceAllString(line, " ")
	words := strings.Fields(normalized)

	// Restore datetime delimiters that were protected during preprocessing
	for i, word := range words {
		restored := word
		restored = strings.ReplaceAll(restored, dtSpacePlaceholder, " ")
		restored = strings.ReplaceAll(restored, dtColonPlaceholder, ":")
		restored = strings.ReplaceAll(restored, dtCommaPlaceholder, ",")
		restored = strings.ReplaceAll(restored, dtEqualPlaceholder, "=")
		words[i] = restored
	}

	return words
}

// filterCommonVariables replaces common variables with wildcards according to configuration.
func (p *Preprocessor) filterCommonVariables(word string) string {
	// Find all matching patterns and select the most specific one
	var bestMatch struct {
		matched bool
		pattern *regexp.Regexp
		name    string
	}

	// Check all patterns and find the best match
	for name, regex := range p.commonVariables {
		if regex.MatchString(word) {
			// If this is the first match or a more specific match
			if !bestMatch.matched || isBetterMatch(regex, bestMatch.pattern, word) {
				bestMatch.matched = true
				bestMatch.pattern = regex
				bestMatch.name = name
			}
		}
	}

	if bestMatch.matched {
		return "<*>"
	}

	// Check if word is numeric-heavy (30% or more digits)
	if isNumericVariable(word) {
		return "<*>"
	}

	return word
}

// isBetterMatch determines if newPattern is more specific than currentPattern for the given word
func isBetterMatch(newPattern, currentPattern *regexp.Regexp, word string) bool {
	// Get the pattern strings
	newPatternStr := newPattern.String()
	currentPatternStr := currentPattern.String()

	// Count non-wildcard characters (excluding regex metacharacters)
	newSpecificity := countSpecificChars(newPatternStr)
	currentSpecificity := countSpecificChars(currentPatternStr)

	// More specific pattern (more non-wildcard chars) wins
	if newSpecificity > currentSpecificity {
		return true
	}

	// If equal specificity, test which pattern matches more precisely
	if newSpecificity == currentSpecificity {
		// Check match length - pattern that matches closer to full word length is better
		newMatch := newPattern.FindString(word)
		currentMatch := currentPattern.FindString(word)

		if len(newMatch) > len(currentMatch) {
			return true
		}

		// If match lengths are equal, longer pattern wins (covers more cases)
		if len(newMatch) == len(currentMatch) && len(newPatternStr) > len(currentPatternStr) {
			return true
		}
	}

	return false
}

// countSpecificChars counts non-wildcard characters in a regex pattern
func countSpecificChars(pattern string) int {
	count := 0
	escaped := false

	for _, ch := range pattern {
		if escaped {
			count++ // Escaped characters are literal
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		// Count literal characters (not regex metacharacters)
		switch ch {
		case '^', '$', '.', '*', '+', '?', '[', ']', '(', ')', '{', '}', '|':
			// Regex metacharacters - don't count
		case '-', ':', '/', ' ', 'T', 'Z': // Common literal separators in datetime patterns
			count += 2 // Weight separators higher as they indicate specific formats
		default:
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				count++ // Letters are specific
			}
		}
	}

	return count
}

// isNumericVariable checks if a token contains 30% or more digits, making it likely a variable
func isNumericVariable(word string) bool {
	if len(word) == 0 {
		return false
	}

	digitCount := 0
	for _, ch := range word {
		if ch >= '0' && ch <= '9' {
			digitCount++
		}
	}

	// If 30% or more of the characters are digits, consider it a variable
	return float64(digitCount)/float64(len(word)) >= 0.3
}

// preprocessDateTimePatterns finds datetime patterns in log lines and protects spaces within them
// This prevents datetime from being split into multiple tokens during tokenization
func preprocessDateTimePatterns(line string) string {
	result := line

	// Apply each pre-compiled pattern and replace ALL delimiter characters with placeholders
	for _, p := range dateTimePatterns {
		result = p.pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Replace all potential delimiters within the datetime with placeholders
			protected := match
			protected = strings.ReplaceAll(protected, " ", dtSpacePlaceholder)
			protected = strings.ReplaceAll(protected, ":", dtColonPlaceholder)
			protected = strings.ReplaceAll(protected, ",", dtCommaPlaceholder)
			protected = strings.ReplaceAll(protected, "=", dtEqualPlaceholder)
			return protected
		})
	}

	return result
}
