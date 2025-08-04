package parser

import (
	"strings"
)

// GenerateTemplatesFromTree extracts templates from the ready tree.
func (p *BrainParser) GenerateTemplatesFromTree(tree *BidirectionalTree) []*ParseResult {
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
			if word != "<*>" && shouldBeVariable(word) {
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
