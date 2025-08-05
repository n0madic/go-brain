package parser

import (
	"math"
	"strings"
)

// GenerateTemplatesFromTree extracts templates from the ready tree.
func (p *BrainParser) GenerateTemplatesFromTree(tree *BidirectionalTree, allLogs []*LogMessage) []*ParseResult {
	var results []*ParseResult
	baseTemplate := make(map[int]string)

	// Fill with words from parent direction
	for pos, node := range tree.ParentDirection {
		if node.IsVariable {
			baseTemplate[pos] = "<*>"
		} else {
			// Now we save the constant word in node.Value
			baseTemplate[pos] = node.Value
		}
	}

	// Fill with words from root (Longest Common Pattern)
	for _, word := range tree.RootNodes {
		baseTemplate[word.Position] = word.Value
	}

	// Recursively traverse child nodes and collect templates
	p.collectTemplatesFromNode(tree.ChildDirectionRoot, baseTemplate, make(map[int]string), &results)

	// Filter results to improve quality if enhanced features are enabled
	if p.config.UseEnhancedPostProcessing && !p.config.isReparsing {
		goodResults, badResults := p.filterLowQualityTemplates(results)

		// Try to reparse bad results with relaxed settings
		if len(badResults) > 0 {
			reparsedResults := p.reparseWithRelaxedSettings(badResults, allLogs)
			goodResults = append(goodResults, reparsedResults...)
		}

		return goodResults
	}

	return results
}

func (p *BrainParser) collectTemplatesFromNode(node *Node, baseTemplate map[int]string, pathTemplate map[int]string, results *[]*ParseResult) {
	if node == nil {
		return
	}

	// Add current node information to the path
	if node.Position >= 0 {
		if node.IsVariable {
			pathTemplate[node.Position] = "<*>"
		} else if node.Value != "" && node.Value != "ROOT" {
			pathTemplate[node.Position] = node.Value
		}
	}

	// CRITICAL IMPROVEMENT: Add iteratively updated parent information
	if node.ParentWords != nil {
		for pos, word := range node.ParentWords {
			if word != "" {
				pathTemplate[pos] = word
			}
		}
	}

	// If this is a leaf node (no children)
	if len(node.Children) == 0 && node.Logs != nil && len(node.Logs) > 0 {
		// Create final template by combining base template and path
		finalTemplate := p.buildCompleteTemplate(baseTemplate, pathTemplate)

		// Collect log IDs
		logIDs := make([]int, len(node.Logs))
		for i, log := range node.Logs {
			logIDs[i] = log.ID
		}

		*results = append(*results, &ParseResult{
			Template: finalTemplate,
			Count:    len(node.Logs),
			LogIDs:   logIDs,
		})
		return
	}

	// Recursively traverse child nodes
	for _, childNode := range node.Children {
		// Create a copy of pathTemplate for each branch
		newPathTemplate := make(map[int]string)
		for k, v := range pathTemplate {
			newPathTemplate[k] = v
		}
		p.collectTemplatesFromNode(childNode, baseTemplate, newPathTemplate, results)
	}
}

// buildCompleteTemplate combines base template and path into final template.
func (p *BrainParser) buildCompleteTemplate(baseTemplate, pathTemplate map[int]string) string {
	// Merge templates
	completeTemplate := make(map[int]string)

	// First copy base template
	for pos, word := range baseTemplate {
		completeTemplate[pos] = word
	}

	// Then add/overwrite with words from path
	for pos, word := range pathTemplate {
		completeTemplate[pos] = word
	}

	// Find maximum position
	maxPos := 0
	for pos := range completeTemplate {
		if pos > maxPos {
			maxPos = pos
		}
	}

	// Assemble final template with enhanced post-processing
	result := make([]string, maxPos+1)
	for i := 0; i <= maxPos; i++ {
		word, ok := completeTemplate[i]
		if ok {
			// Apply enhanced post-processing to catch missed variables
			if word != "<*>" && p.shouldBeVariableWithConfig(word) {
				result[i] = "<*>"
			} else {
				result[i] = word
			}
		} else {
			// If position is not filled, it's a variable
			result[i] = "<*>"
		}
	}

	return strings.Join(result, " ")
}

// shouldBeVariableWithConfig wraps the variable detection logic with config consideration
func (p *BrainParser) shouldBeVariableWithConfig(word string) bool {
	if p.config.UseEnhancedPostProcessing {
		return p.shouldBeVariableEnhanced(word)
	}
	return shouldBeVariable(word)
}

// shouldBeVariable checks if a token should be considered a variable during post-processing
// This catches variables that might have been missed during preprocessing
func shouldBeVariable(word string) bool {
	// Check if word contains significant numeric content
	if isNumericVariable(word) {
		return true
	}

	// Check for other variable patterns that might have been missed
	// Mixed alphanumeric with special characters often indicates variables
	if containsMixedPatterns(word) {
		return true
	}

	return false
}

// containsMixedPatterns checks for patterns that typically indicate variables
func containsMixedPatterns(word string) bool {
	if len(word) < 3 { // Too short to analyze patterns
		return false
	}

	// Skip common protocol/format names
	upperWord := strings.ToUpper(word)
	if upperWord == "HTTP" || upperWord == "HTTPS" || upperWord == "SOCKS5" ||
		upperWord == "FTP" || upperWord == "SSH" || upperWord == "TCP" ||
		upperWord == "UDP" || upperWord == "IPV4" || upperWord == "IPV6" {
		return false
	}

	hasLetters := false
	hasDigits := false
	hasSpecial := false
	digitCount := 0
	letterCount := 0

	for _, ch := range word {
		switch {
		case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'):
			hasLetters = true
			letterCount++
		case ch >= '0' && ch <= '9':
			hasDigits = true
			digitCount++
		case ch == '_' || ch == '-' || ch == '.' || ch == ':' || ch == '/':
			hasSpecial = true
		}
	}

	// If it's mostly letters with just one digit at the end (like SOCKS5), it's likely a constant
	if letterCount > digitCount*2 && digitCount <= 1 {
		return false
	}

	// Mixed patterns often indicate variables
	// Examples: user_123, id-456, v2.3.4, path/to/file123
	mixedCount := 0
	if hasLetters {
		mixedCount++
	}
	if hasDigits {
		mixedCount++
	}
	if hasSpecial {
		mixedCount++
	}

	// If we have at least 2 different types of characters AND meaningful digits, it's likely a variable
	return mixedCount >= 2 && hasDigits && digitCount > 1
}

// shouldBeVariableEnhanced implements enhanced variable detection from Drain+
// This version uses more sophisticated heuristics and pattern matching
func (p *BrainParser) shouldBeVariableEnhanced(word string) bool {
	// First, check with the standard algorithm
	if shouldBeVariable(word) {
		return true
	}

	// Additional Drain+ heuristics

	// 1. Check for mixed case with numbers (e.g., User123, ID_456)
	if hasComplexPattern(word) {
		return true
	}

	// 2. Check for timestamp-like patterns not caught by regex (with config)
	if p.looksLikeTimestamp(word) {
		return true
	}

	// 3. Check for hash-like patterns (common in logs)
	if looksLikeHash(word) {
		return true
	}

	// 4. Check for encoded data patterns
	if looksLikeEncoded(word) {
		return true
	}

	// 5. High entropy check (indicates randomness) with config
	if p.hasHighEntropy(word) {
		return true
	}

	return false
}

// hasComplexPattern checks for complex alphanumeric patterns
func hasComplexPattern(word string) bool {
	if len(word) < 4 {
		return false
	}

	// Count transitions between character types
	transitions := 0
	prevType := 0 // 0: none, 1: letter, 2: digit, 3: special

	for _, ch := range word {
		currType := 0
		switch {
		case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'):
			currType = 1
		case ch >= '0' && ch <= '9':
			currType = 2
		case ch == '_' || ch == '-' || ch == '.':
			currType = 3
		}

		if prevType != 0 && currType != 0 && prevType != currType {
			transitions++
		}
		prevType = currType
	}

	// High number of transitions indicates a variable
	return transitions >= 3
}

// looksLikeTimestamp checks for timestamp-like patterns
func (p *BrainParser) looksLikeTimestamp(word string) bool {
	// Check for patterns like 2023-01-15, 15:30:45, 1673789445
	digitCount := 0
	separatorCount := 0

	for _, ch := range word {
		if ch >= '0' && ch <= '9' {
			digitCount++
		} else if ch == ':' || ch == '-' || ch == '/' || ch == '.' {
			separatorCount++
		}
	}

	// Timestamp-like: mostly digits with some separators (using config values)
	return digitCount >= p.config.TimestampMinDigits && separatorCount >= p.config.TimestampMinSeparators
}

// looksLikeHash checks for hash-like patterns
func looksLikeHash(word string) bool {
	if len(word) < 8 {
		return false
	}

	// Count hex-like characters
	hexCount := 0
	for _, ch := range word {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			hexCount++
		}
	}

	// If mostly hex characters and long enough, likely a hash
	return float64(hexCount)/float64(len(word)) > 0.8 && len(word) >= 16
}

// looksLikeEncoded checks for base64 or other encoded patterns
func looksLikeEncoded(word string) bool {
	if len(word) < 8 {
		return false
	}

	// Check for base64-like patterns
	validChars := 0
	for _, ch := range word {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '+' || ch == '/' || ch == '=' {
			validChars++
		}
	}

	// High ratio of base64 chars and ends with = padding
	isBase64Like := float64(validChars)/float64(len(word)) > 0.95 &&
		(strings.HasSuffix(word, "=") || strings.HasSuffix(word, "=="))

	// Also check for high character diversity (typical in encoded data)
	uniqueChars := make(map[rune]bool)
	for _, ch := range word {
		uniqueChars[ch] = true
	}

	highDiversity := float64(len(uniqueChars))/float64(len(word)) > 0.6

	return isBase64Like || (len(word) >= 16 && highDiversity)
}

// hasHighEntropy calculates Shannon entropy to detect random strings
func (p *BrainParser) hasHighEntropy(word string) bool {
	if len(word) < p.config.MinEntropyLength {
		return false
	}

	// Count character frequencies
	freq := make(map[rune]int)
	for _, ch := range word {
		freq[ch]++
	}

	// Calculate Shannon entropy
	entropy := 0.0
	wordLen := float64(len(word))

	for _, count := range freq {
		probability := float64(count) / wordLen
		if probability > 0 {
			entropy -= probability * math.Log2(probability)
		}
	}

	// Normalize by word length (longer words naturally have higher entropy)
	normalizedEntropy := entropy / math.Log2(wordLen)

	// High entropy indicates randomness (likely a variable) - using config threshold
	return normalizedEntropy > p.config.EntropyThreshold
}

// filterLowQualityTemplates separates templates into quality and low-quality groups
func (p *BrainParser) filterLowQualityTemplates(results []*ParseResult) (good []*ParseResult, bad []*ParseResult) {
	for _, result := range results {
		if p.isQualityTemplate(result.Template) {
			good = append(good, result)
		} else {
			bad = append(bad, result)
		}
	}

	return good, bad
}

// isQualityTemplate checks if a template meets quality criteria
func (p *BrainParser) isQualityTemplate(template string) bool {
	tokens := strings.Fields(template)
	if len(tokens) == 0 {
		return false
	}

	// Count consecutive wildcards and content words
	wildcardCount := 0
	maxConsecutiveWildcards := 0
	currentConsecutive := 0
	contentWords := 0

	for _, token := range tokens {
		if token == "<*>" {
			wildcardCount++
			currentConsecutive++
			if currentConsecutive > maxConsecutiveWildcards {
				maxConsecutiveWildcards = currentConsecutive
			}
		} else {
			contentWords++
			currentConsecutive = 0
		}
	}

	// Filter based on config parameters

	// 1. Check maximum consecutive wildcards (if configured)
	if p.config.MaxConsecutiveWildcards > 0 && maxConsecutiveWildcards > p.config.MaxConsecutiveWildcards {
		return false
	}

	// 2. Check minimum content words ratio
	contentRatio := float64(contentWords) / float64(len(tokens))
	if contentRatio < p.config.MinContentWordsRatio {
		return false
	}

	// 3. Skip templates that are just wildcards
	if wildcardCount == len(tokens) {
		return false
	}

	return true
}

// extractLogsFromResults extracts all unique logs from a slice of ParseResults
func extractLogsFromResults(results []*ParseResult, allLogs []*LogMessage) []*LogMessage {
	var extractedLogs []*LogMessage
	seenLogIDs := make(map[int]bool)

	for _, result := range results {
		for _, logID := range result.LogIDs {
			if !seenLogIDs[logID] && logID < len(allLogs) {
				extractedLogs = append(extractedLogs, allLogs[logID])
				seenLogIDs[logID] = true
			}
		}
	}

	return extractedLogs
}

// reparseWithRelaxedSettings attempts to reparse low-quality templates with progressively relaxed settings
func (p *BrainParser) reparseWithRelaxedSettings(badResults []*ParseResult, allLogs []*LogMessage) []*ParseResult {
	if len(badResults) == 0 {
		return nil
	}

	// Extract logs from bad results
	logsToReparse := extractLogsFromResults(badResults, allLogs)
	if len(logsToReparse) == 0 {
		return badResults // Return original bad results if nothing to reparse
	}

	// Convert LogMessage slice to string slice for reparsing
	logLines := make([]string, len(logsToReparse))
	for i, log := range logsToReparse {
		logLines[i] = log.Content
	}

	var allGoodResults []*ParseResult

	// Level 1: Relaxed enhanced parameters
	relaxedConfig := p.config
	relaxedConfig.EntropyThreshold = 0.95
	relaxedConfig.MinEntropyLength = 15
	relaxedConfig.TimestampMinDigits = 10
	relaxedConfig.isReparsing = true

	if results := p.tryReparseWithConfig(logLines, relaxedConfig); len(results) > 0 {
		if goodResults, _ := p.filterLowQualityTemplatesWithConfig(results, relaxedConfig); len(goodResults) > 0 {
			allGoodResults = append(allGoodResults, goodResults...)
			// Remove processed logs and continue with remaining
			processedLogIDs := make(map[int]bool)
			for _, result := range goodResults {
				for _, logID := range result.LogIDs {
					processedLogIDs[logID] = true
				}
			}
			logLines = p.removeProcessedLogs(logLines, logsToReparse, processedLogIDs)
		}
	}

	// Level 2: Disable enhanced post-processing
	if len(logLines) > 0 {
		noEnhancedConfig := p.config
		noEnhancedConfig.UseEnhancedPostProcessing = false
		noEnhancedConfig.isReparsing = true

		if results := p.tryReparseWithConfig(logLines, noEnhancedConfig); len(results) > 0 {
			if goodResults, _ := p.filterLowQualityTemplatesWithConfig(results, noEnhancedConfig); len(goodResults) > 0 {
				allGoodResults = append(allGoodResults, goodResults...)
				// Remove processed logs and continue with remaining
				processedLogIDs := make(map[int]bool)
				for _, result := range goodResults {
					for _, logID := range result.LogIDs {
						processedLogIDs[logID] = true
					}
				}
				logLines = p.removeProcessedLogs(logLines, logsToReparse, processedLogIDs)
			}
		}
	}

	// Level 3: Original Brain algorithm (no enhancements)
	if len(logLines) > 0 {
		originalConfig := p.config
		originalConfig.UseEnhancedPostProcessing = false
		originalConfig.UseStatisticalThreshold = false
		originalConfig.isReparsing = true

		if results := p.tryReparseWithConfig(logLines, originalConfig); len(results) > 0 {
			// For original Brain, accept any results (no further filtering)
			allGoodResults = append(allGoodResults, results...)
		}
	}

	// Return combined results, or original bad results if nothing worked
	if len(allGoodResults) > 0 {
		return allGoodResults
	}
	return badResults
}

// removeProcessedLogs removes logs that have been successfully processed from the remaining log lines
func (p *BrainParser) removeProcessedLogs(logLines []string, originalLogs []*LogMessage, processedLogIDs map[int]bool) []string {
	var remaining []string
	for i, log := range originalLogs {
		if i < len(logLines) && !processedLogIDs[log.ID] {
			remaining = append(remaining, logLines[i])
		}
	}
	return remaining
}

// tryReparseWithConfig attempts to reparse logs with given configuration
func (p *BrainParser) tryReparseWithConfig(logLines []string, config Config) []*ParseResult {
	// Create new parser with modified config
	reparseParser := New(config)
	return reparseParser.Parse(logLines)
}

// filterLowQualityTemplatesWithConfig is a helper for reparsing with specific config
func (p *BrainParser) filterLowQualityTemplatesWithConfig(results []*ParseResult, config Config) (good []*ParseResult, bad []*ParseResult) {
	// Create temporary parser with the config to use its quality check
	tempParser := &BrainParser{config: config}
	return tempParser.filterLowQualityTemplates(results)
}
