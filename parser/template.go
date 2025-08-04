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

	// Assemble final template
	result := make([]string, maxPos+1)
	for i := 0; i <= maxPos; i++ {
		word, ok := completeTemplate[i]
		if ok {
			result[i] = word
		} else {
			// If position is not filled, it's a variable
			result[i] = "<*>"
		}
	}

	return strings.Join(result, " ")
}
