package parser

import (
	"regexp"
	"strings"
)

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
	// 1. Split logs without filtering to get original words
	wordFrequencies := make(map[string]int)
	var rawSplitLogs [][]string
	for _, line := range logLines {
		words := p.splitWithoutFiltering(line)
		rawSplitLogs = append(rawSplitLogs, words)
		for _, word := range words {
			wordFrequencies[word]++
		}
	}

	// 2. Create LogMessage structures, applying filtering while preserving original frequencies
	processedLogs := make([]*LogMessage, len(logLines))
	for i, rawWords := range rawSplitLogs {
		logMessage := &LogMessage{
			ID:      i,
			Content: logLines[i],
			Words:   make([]Word, len(rawWords)),
		}
		for j, rawWord := range rawWords {
			// Apply common variable filtering to the word value
			filteredWord := p.filterCommonVariables(rawWord)
			logMessage.Words[j] = Word{
				Value:     filteredWord,
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
	return strings.Fields(normalized)
}

// filterCommonVariables replaces common variables with wildcards according to configuration.
func (p *Preprocessor) filterCommonVariables(word string) string {
	for _, regex := range p.commonVariables {
		if regex.MatchString(word) {
			return "<*>"
		}
	}
	return word
}
