package parser

import (
	"regexp"
	"sort"
)

// CreateInitialGroups creates initial groups of logs.
func CreateInitialGroups(logs []*LogMessage, config *Config) map[string]*LogGroup {
	// 1. Group logs by length
	logsByLength := make(map[int][]*LogMessage)
	for _, log := range logs {
		length := len(log.Words)
		logsByLength[length] = append(logsByLength[length], log)
	}

	finalGroups := make(map[string]*LogGroup)

	// 2. For each group with the same length, group by Longest Common Pattern
	for _, group := range logsByLength {
		logsByPattern := make(map[string][]*LogMessage)
		patterns := make(map[string]LogPattern)

		for _, log := range group {
			lcp := findLongestWordCombination(log, config)
			key := lcp.Key()
			logsByPattern[key] = append(logsByPattern[key], log)
			if _, exists := patterns[key]; !exists {
				patterns[key] = LogPattern{Words: lcp.Words, Frequency: lcp.Frequency}
			}
		}

		// Form final groups
		for key, logList := range logsByPattern {
			finalGroups[key] = &LogGroup{
				Pattern: patterns[key],
				Logs:    logList,
			}
		}
	}

	return finalGroups
}

// findLongestWordCombination finds the longest combination of words with the same frequency.
// Implements frequency threshold according to the paper: threshold = highest_frequency * weight.
// Also handles two-frequency logs as mentioned in the paper.
func findLongestWordCombination(log *LogMessage, config *Config) WordCombination {
	combosByFreq := make(map[int][]Word)
	for _, word := range log.Words {
		combosByFreq[word.Frequency] = append(combosByFreq[word.Frequency], word)
	}

	if len(combosByFreq) == 0 {
		return WordCombination{}
	}

	// Handle two-frequency logs as mentioned in the paper
	if len(combosByFreq) == 2 {
		if isTwoFrequencyVariableLog(combosByFreq) {
			// Return the smaller frequency group as it's more likely to be constants
			return selectConstantCombinationFromTwoFrequency(combosByFreq)
		}
	}

	// Find maximum frequency for threshold calculation
	maxFrequency := 0
	for freq := range combosByFreq {
		if freq > maxFrequency {
			maxFrequency = freq
		}
	}

	// Calculate frequency threshold according to the paper
	frequencyThreshold := float64(maxFrequency) * config.Weight

	var longestCombo WordCombination
	maxLen := 0
	maxTokenLength := 0 // Secondary metric for tie-breaking

	// Sort frequencies to ensure deterministic result
	freqs := make([]int, 0, len(combosByFreq))
	for f := range combosByFreq {
		freqs = append(freqs, f)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(freqs))) // Priority to more frequent

	for _, freq := range freqs {
		// Filter by frequency threshold
		if float64(freq) < frequencyThreshold {
			continue
		}

		words := combosByFreq[freq]
		tokenLength := calculateTotalTokenLength(words)

		// Primary criterion: number of words, secondary: total token length
		if len(words) > maxLen || (len(words) == maxLen && tokenLength > maxTokenLength) {
			maxLen = len(words)
			maxTokenLength = tokenLength
			longestCombo = WordCombination{Frequency: freq, Words: words}
		}
	}

	// If no combination passed threshold, return the most frequent
	if maxLen == 0 && len(freqs) > 0 {
		freq := freqs[0] // Highest frequency
		words := combosByFreq[freq]
		longestCombo = WordCombination{Frequency: freq, Words: words}
	}

	return longestCombo
}

// isTwoFrequencyVariableLog checks if a two-frequency log likely contains variable parts
// by looking for numeric patterns, IP addresses, or other variable-like patterns
func isTwoFrequencyVariableLog(combosByFreq map[int][]Word) bool {
	// Get the two frequency groups
	var freqs []int
	for freq := range combosByFreq {
		freqs = append(freqs, freq)
	}
	if len(freqs) != 2 {
		return false
	}

	// Check both groups for variable patterns
	for _, freq := range freqs {
		words := combosByFreq[freq]
		if hasVariablePatterns(words) {
			return true
		}
	}
	return false
}

// hasVariablePatterns checks if words contain patterns typical for variables
func hasVariablePatterns(words []Word) bool {
	// Common variable patterns from the paper examples
	numericPattern := regexp.MustCompile(`^\d+$`)                   // Pure numbers
	ipPattern := regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)         // IP addresses
	idPattern := regexp.MustCompile(`^[a-zA-Z]+_?\d+$`)             // ID patterns like "blk_123"
	channelPattern := regexp.MustCompile(`^\d+\s+\d+\s+\d+\s+\d+$`) // Space-separated numbers

	variableCount := 0
	for _, word := range words {
		if numericPattern.MatchString(word.Value) ||
			ipPattern.MatchString(word.Value) ||
			idPattern.MatchString(word.Value) ||
			channelPattern.MatchString(word.Value) {
			variableCount++
		}
	}

	// If more than half the words look like variables, consider it a variable group
	return float64(variableCount)/float64(len(words)) > 0.5
}

// selectConstantCombinationFromTwoFrequency selects the combination more likely to contain constants
func selectConstantCombinationFromTwoFrequency(combosByFreq map[int][]Word) WordCombination {
	var freqs []int
	for freq := range combosByFreq {
		freqs = append(freqs, freq)
	}
	sort.Ints(freqs) // Lower frequency first

	// Try lower frequency first (more likely to be constants)
	for _, freq := range freqs {
		words := combosByFreq[freq]
		if !hasVariablePatterns(words) {
			return WordCombination{Frequency: freq, Words: words}
		}
	}

	// If both seem like variables, choose the lower frequency
	freq := freqs[0]
	return WordCombination{Frequency: freq, Words: combosByFreq[freq]}
}

// calculateTotalTokenLength calculates the sum of token lengths for tie-breaking
func calculateTotalTokenLength(words []Word) int {
	total := 0
	for _, word := range words {
		total += len(word.Value)
	}
	return total
}
