package parser

import (
	"math"
	"sort"
)

// BrainParser - main parser structure.
type BrainParser struct {
	config Config
}

// New creates a new BrainParser instance with the given configuration.
func New(config Config) *BrainParser {
	if config.Delimiters == "" {
		// Default value as per the paper (space, colon, comma, equals)
		config.Delimiters = `[\s,:=]`
	}
	if config.ChildBranchThreshold == 0 {
		config.ChildBranchThreshold = 3 // Empirical value from the paper
	}
	if config.Weight == 0 {
		config.Weight = 0.0 // In offline mode weight = 0 (as per the paper)
	}
	if config.DynamicThresholdFactor == 0 {
		config.DynamicThresholdFactor = 2.0 // Default factor for dynamic threshold
	}

	// Add default CommonVariables patterns if none provided
	if config.CommonVariables == nil {
		config.CommonVariables = getDefaultCommonVariables()
	}

	return &BrainParser{config: config}
}

// getDefaultCommonVariables returns default patterns for common variable types
// These patterns help identify variables that should be replaced with <*> during preprocessing
func getDefaultCommonVariables() map[string]string {
	return map[string]string{
		"ipv4_address":  `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`,                                          // IPv4 addresses
		"ipv4_port":     `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+$`,                                      // IPv4:port combinations
		"hostname_port": `^[a-zA-Z0-9.-]+:\d+$`,                                                          // hostname:port combinations
		"pure_numbers":  `^\d+$`,                                                                         // Pure numeric values
		"hex_numbers":   `^0x[a-fA-F0-9]+$`,                                                              // Hexadecimal numbers
		"timestamps":    `^\d{2}:\d{2}:\d{2}$`,                                                           // Time stamps (HH:MM:SS)
		"block_ids":     `^blk_[-]?\d+$`,                                                                 // Block IDs like blk_123
		"file_sizes":    `^\d+[KMGT]?B$`,                                                                 // File sizes like 123KB, 4GB
		"percentages":   `^\d{1,3}%$`,                                                                    // Percentage values
		"uuid":          `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`, // UUIDs
	}
}

// Parse analyzes a slice of log lines and returns found patterns.
func (p *BrainParser) Parse(logLines []string) []*ParseResult {
	preprocessor := NewPreprocessor(p.config.Delimiters, p.config.CommonVariables)
	processedLogs := preprocessor.PreprocessLogs(logLines)

	initialGroups := CreateInitialGroups(processedLogs, &p.config)

	var allTemplates []*ParseResult

	for _, group := range initialGroups {
		// Steps 3 and 4: Build tree for each group
		tree := p.BuildTreeForGroup(group)

		// Step 5: Generate templates from tree
		// GenerateTemplatesFromTree performs complete template extraction with full
		// bidirectional tree traversal, iterative parent updates, and proper log tracking
		templates := p.GenerateTemplatesFromTree(tree)
		allTemplates = append(allTemplates, templates...)
	}

	// Aggregate identical templates
	return p.aggregateResults(allTemplates)
}

// aggregateResults combines duplicate templates into one.
func (p *BrainParser) aggregateResults(results []*ParseResult) []*ParseResult {
	aggMap := make(map[string]*ParseResult)
	for _, res := range results {
		if existing, ok := aggMap[res.Template]; ok {
			existing.Count += res.Count
			existing.LogIDs = append(existing.LogIDs, res.LogIDs...)
		} else {
			// Copy to avoid modifying the original slice
			newRes := *res
			aggMap[res.Template] = &newRes
		}
	}

	finalList := make([]*ParseResult, 0, len(aggMap))
	for _, res := range aggMap {
		finalList = append(finalList, res)
	}

	// Sort by popularity for nice output
	sort.Slice(finalList, func(i, j int) bool {
		return finalList[i].Count > finalList[j].Count
	})

	return finalList
}

// calculateDynamicThreshold calculates dynamic threshold based on unique words count in column
// according to the paper: threshold = log(unique_words_count) * factor
func (p *BrainParser) calculateDynamicThreshold(uniqueWordsCount int) int {
	if !p.config.UseDynamicThreshold || uniqueWordsCount <= 0 {
		return p.config.ChildBranchThreshold
	}

	// Use natural logarithm as suggested in the paper discussion
	dynamicThreshold := int(math.Log(float64(uniqueWordsCount)) * p.config.DynamicThresholdFactor)

	// Ensure minimum threshold of 2 to avoid too aggressive merging
	if dynamicThreshold < 2 {
		dynamicThreshold = 2
	}

	// Cap at reasonable maximum to avoid too conservative splitting
	if dynamicThreshold > 10 {
		dynamicThreshold = 10
	}

	return dynamicThreshold
}

// BuildTreeForGroup builds a bidirectional tree for one log group.
func (p *BrainParser) BuildTreeForGroup(group *LogGroup) *BidirectionalTree {
	tree := &BidirectionalTree{
		RootNodes:          group.Pattern.Words,
		ParentDirection:    make(map[int]*Node),
		ChildDirectionRoot: &Node{Value: "ROOT", Children: make(map[string]*Node), Logs: group.Logs},
		LogGroups:          make(map[string][]*LogMessage),
		RootPattern:        group.Pattern,
	}

	rootPositions := make(map[int]bool)
	for _, word := range tree.RootNodes {
		rootPositions[word.Position] = true
	}

	var parentCols, childCols []int
	columnWords := getColumnWords(group.Logs)

	for pos := range columnWords {
		if rootPositions[pos] {
			continue
		}
		// Determination of representative frequency for the column
		// According to the paper, this is the maximum frequency of a word in the column
		maxFreq := 0
		for _, word := range columnWords[pos] {
			if word.Frequency > maxFreq {
				maxFreq = word.Frequency
			}
		}

		if maxFreq > group.Pattern.Frequency {
			parentCols = append(parentCols, pos)
		} else {
			childCols = append(childCols, pos)
		}
	}

	// Store parent columns for iterative updates
	tree.ParentColumns = parentCols

	// Step 3: Node update in the parent direction
	p.updateParentDirection(tree, group.Logs, parentCols)

	// Step 4: Node update in the child direction with dynamic threshold
	// This process is recursive and is the heart of the algorithm
	p.updateChildDirection(tree, tree.ChildDirectionRoot, group.Logs, childCols)

	return tree
}

// updateParentDirection (Algorithm 2)
func (p *BrainParser) updateParentDirection(tree *BidirectionalTree, logs []*LogMessage, parentCols []int) {
	for _, pos := range parentCols {
		uniqueWords := make(map[string]bool)
		var constantWord string
		for _, log := range logs {
			if pos < len(log.Words) {
				word := log.Words[pos].Value
				uniqueWords[word] = true
				if constantWord == "" {
					constantWord = word
				}
			}
		}

		node := &Node{
			IsVariable: len(uniqueWords) > 1,
			Position:   pos,
			Logs:       logs,
		}

		// If word is constant (only one unique), save its value
		if !node.IsVariable && constantWord != "" {
			node.Value = constantWord
		}

		tree.ParentDirection[pos] = node
	}
}

// updateChildDirection (Algorithm 3, recursive part) with dynamic threshold support and iterative parent updates
func (p *BrainParser) updateChildDirection(tree *BidirectionalTree, rootNode *Node, currentLogs []*LogMessage, childCols []int) {
	if len(childCols) == 0 {
		return
	}

	// Sort columns by number of unique words (as in the paper)
	sort.Slice(childCols, func(i, j int) bool {
		posI, posJ := childCols[i], childCols[j]
		countI := countUniqueWordsInColumn(currentLogs, posI)
		countJ := countUniqueWordsInColumn(currentLogs, posJ)
		return countI < countJ
	})

	posToProcess := childCols[0]
	remainingCols := childCols[1:]

	wordsInColumn := make(map[string][]*LogMessage)
	for _, log := range currentLogs {
		if posToProcess < len(log.Words) {
			word := log.Words[posToProcess].Value
			wordsInColumn[word] = append(wordsInColumn[word], log)
		}
	}

	// Calculate dynamic threshold based on unique words count
	uniqueWordsCount := len(wordsInColumn)
	threshold := p.calculateDynamicThreshold(uniqueWordsCount)

	// If number of branches > threshold, consider all as variables
	if uniqueWordsCount > threshold {
		rootNode.Children["<*>"] = &Node{
			IsVariable: true,
			Children:   make(map[string]*Node),
			Position:   posToProcess,
			Logs:       currentLogs,
		}
		// Continue recursion for the same group, but with remaining columns
		p.updateChildDirection(tree, rootNode.Children["<*>"], currentLogs, remainingCols)
	} else {
		// Otherwise create constant branches and split the group
		for word, subGroupLogs := range wordsInColumn {
			newNode := &Node{
				Value:      word,
				IsVariable: false,
				Children:   make(map[string]*Node),
				Position:   posToProcess,
				Logs:       subGroupLogs,
			}
			rootNode.Children[word] = newNode

			// CRITICAL IMPROVEMENT: Iteratively update parent nodes for the new subgroup
			p.iterativelyUpdateParentNodes(tree, newNode, subGroupLogs)

			// Recursive call for each new subgroup
			p.updateChildDirection(tree, newNode, subGroupLogs, remainingCols)
		}
	}
}

// iterativelyUpdateParentNodes recalculates parent nodes for subgroups
// This is the critical improvement that addresses variable->constant reclassification
func (p *BrainParser) iterativelyUpdateParentNodes(tree *BidirectionalTree, node *Node, subGroupLogs []*LogMessage) {
	// For each parent column, check if it should be reclassified in this subgroup
	for _, parentPos := range tree.ParentColumns {
		uniqueWords := make(map[string]bool)
		var constantWord string

		for _, log := range subGroupLogs {
			if parentPos < len(log.Words) {
				word := log.Words[parentPos].Value
				uniqueWords[word] = true
				if constantWord == "" {
					constantWord = word
				}
			}
		}

		// Create/update node specific to this subgroup
		parentNode := &Node{
			IsVariable: len(uniqueWords) > 1,
			Position:   parentPos,
			Logs:       subGroupLogs,
		}

		// If word became constant in this subgroup, save its value
		if !parentNode.IsVariable && constantWord != "" {
			parentNode.Value = constantWord
		}

		// Store subgroup-specific parent information in the node
		if node.ParentWords == nil {
			node.ParentWords = make([]string, len(subGroupLogs[0].Words))
		}

		// Store the result for this position
		if parentNode.IsVariable {
			if len(node.ParentWords) > parentPos {
				// Ensure we don't have index out of bounds
				// Find the max position we need to support
				maxPos := parentPos
				for _, log := range subGroupLogs {
					if len(log.Words)-1 > maxPos {
						maxPos = len(log.Words) - 1
					}
				}
				// Extend slice if needed
				for len(node.ParentWords) <= maxPos {
					node.ParentWords = append(node.ParentWords, "")
				}
				node.ParentWords[parentPos] = "<*>"
			}
		} else if constantWord != "" {
			// Ensure we have enough capacity
			for len(node.ParentWords) <= parentPos {
				node.ParentWords = append(node.ParentWords, "")
			}
			node.ParentWords[parentPos] = constantWord
		}
	}
}

// Helper functions for tree building

func getColumnWords(logs []*LogMessage) map[int][]Word {
	columnWords := make(map[int][]Word)
	if len(logs) == 0 {
		return columnWords
	}
	numCols := len(logs[0].Words)
	for i := 0; i < numCols; i++ {
		for _, log := range logs {
			if i < len(log.Words) {
				columnWords[i] = append(columnWords[i], log.Words[i])
			}
		}
	}
	return columnWords
}

func countUniqueWordsInColumn(logs []*LogMessage, position int) int {
	unique := make(map[string]bool)
	for _, log := range logs {
		if position < len(log.Words) {
			unique[log.Words[position].Value] = true
		}
	}
	return len(unique)
}
