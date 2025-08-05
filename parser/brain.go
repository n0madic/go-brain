// Package parser implements the Brain log parsing algorithm for extracting templates from log files.
package parser

import (
	"math"
	"sort"
	"sync"
	"unique"
)

// BrainParser - main parser structure.
type BrainParser struct {
	config       Config
	preprocessor *Preprocessor // Cached preprocessor with compiled regexes
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
	if config.ParallelProcessingThreshold == 0 {
		config.ParallelProcessingThreshold = 1000 // Default: enable parallel processing for groups with 1000+ logs
	}

	// Enhanced Features Tuning Parameters defaults
	if config.EntropyThreshold == 0 {
		config.EntropyThreshold = 0.85 // More conservative than original 0.7
	}
	if config.MinEntropyLength == 0 {
		config.MinEntropyLength = 10 // Longer than original 8
	}
	if config.MaxConsecutiveWildcards == 0 {
		config.MaxConsecutiveWildcards = 5 // Limit consecutive <*>
	}
	if config.MinContentWordsRatio == 0 {
		config.MinContentWordsRatio = 0.25 // At least 25% content words
	}
	if config.TimestampMinDigits == 0 {
		config.TimestampMinDigits = 8 // More conservative than original 6
	}
	if config.TimestampMinSeparators == 0 {
		config.TimestampMinSeparators = 2 // Same as original
	}

	// Add default CommonVariables patterns if none provided
	if config.CommonVariables == nil {
		config.CommonVariables = getDefaultCommonVariables()
	}

	// Create preprocessor once with compiled regexes for performance
	preprocessor := NewPreprocessor(config.Delimiters, config.CommonVariables)

	return &BrainParser{
		config:       config,
		preprocessor: preprocessor,
	}
}

// getDefaultCommonVariables returns default patterns for common variable types
// These patterns help identify variables that should be replaced with <*> during preprocessing
// Patterns are ordered by specificity - more specific patterns first to prevent simple patterns from matching
func getDefaultCommonVariables() map[string]string {
	return map[string]string{
		// Time and date patterns (FIRST - most specific)
		// Full datetime patterns
		"iso_datetime_with_ms": `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z?$`,  // 2024-01-15T10:30:15.123Z
		"iso_datetime":         `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z?$`,         // 2024-01-15T10:30:15Z
		"iso_datetime_space":   `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d{3})?$`, // 2024-01-15 10:30:15.123
		"european_datetime":    `^\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}$`,           // 15/01/2024 10:30:15
		"us_datetime":          `^\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}$`,           // 01/15/2024 10:30:15
		"syslog_datetime":      `^[A-Z][a-z]{2} \d{1,2} \d{2}:\d{2}:\d{2}$`,       // Jan 15 10:30:15

		// Date-only patterns (after full datetime)
		"iso_date":             `^\d{4}-\d{2}-\d{2}$`,           // 2024-01-15
		"european_date":        `^\d{2}/\d{2}/\d{4}$`,           // 15/01/2024
		"us_date":              `^\d{2}/\d{2}/\d{4}$`,           // 01/15/2024
		"date_with_dots":       `^\d{2}\.\d{2}\.\d{4}$`,         // 15.01.2024
		"date_with_slashes":    `^\d{4}/\d{2}/\d{2}$`,           // 2025/07/31
		"date_with_month_name": `^\d{1,2}-[A-Z][a-z]{2}-\d{4}$`, // 31-Jul-2025

		// Time-only patterns
		"time_with_seconds": `^\d{2}:\d{2}:\d{2}$`,        // 10:30:15
		"time_with_ms":      `^\d{2}:\d{2}:\d{2}\.\d{3}$`, // 10:30:15.123
		"time_simple":       `^\d{2}:\d{2}$`,              // 10:30

		// Unix timestamps
		"unix_timestamp_ms": `^\d{13}$`, // Milliseconds timestamp
		"unix_timestamp":    `^\d{10}$`, // Seconds timestamp

		// Network-related patterns
		"ipv4_address":  `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`,      // IPv4 addresses
		"ipv4_port":     `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+$`,  // IPv4:port combinations
		"ipv6_address":  `^([0-9a-fA-F]{0,4}:){7}[0-9a-fA-F]{0,4}$`,  // IPv6 addresses
		"mac_address":   `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`, // MAC addresses
		"hostname_port": `^[a-zA-Z0-9.-]+:\d+$`,                      // hostname:port combinations

		// File and system patterns (before pure numbers)
		"file_sizes":   `^\d+[KMGT]?B$`,                       // File sizes like 123KB, 4GB
		"unix_path":    `^(/[a-zA-Z0-9._-]+)+/?$`,             // Unix file paths
		"windows_path": `^[A-Za-z]:\\(\\[^\\/:*?"<>|]+)*\\?$`, // Windows file paths
		"filename_ext": `^[a-zA-Z0-9._-]+\.[a-zA-Z]{2,4}$`,    // Filenames with extensions

		// Web and email patterns
		"url":   `^https?://[^\s]+$`,                                // URLs
		"email": `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, // Email addresses

		// Identifiers and special numbers (before pure numbers)
		"hex_numbers": `^0x[a-fA-F0-9]+$`,                                                              // Hexadecimal numbers
		"uuid":        `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`, // UUIDs
		"block_ids":   `^blk_[-]?\d+$`,                                                                 // Block IDs like blk_123
		"session_id":  `^[a-zA-Z0-9]{16,}$`,                                                            // Session IDs (16+ alphanumeric)
		"version":     `^v?\d+\.\d+(\.\d+)?(-[a-zA-Z0-9._-]+)?$`,                                       // Software versions
		"percentages": `^\d{1,3}%$`,                                                                    // Percentage values
		"memory_addr": `^0x[0-9a-fA-F]+$`,                                                              // Memory addresses

		// Common log datetime fragments (after being split by tokenizer)
		// Month names for syslog format
		"month_names": `^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)$`, // Month abbreviations
		// Common datetime fragments in logs after tokenization
		"bracket_date":          `^\[\d{1,2}-[A-Z][a-z]{2}-\d{4}$`,                     // [31-Jul-2025
		"bracket_time":          `^\d{2}:\d{2}:\d{2}\]$`,                               // 01:17:58]
		"bracket_datetime_full": `^\[\d{1,2}-[A-Z][a-z]{2}-\d{4} \d{2}:\d{2}:\d{2}\]$`, // [31-Jul-2025 01:17:58]

		// Pure numbers (LAST - least specific, catches everything else)
		"pure_numbers": `^\d+$`, // Pure numeric values
	}
}

// Parse analyzes a slice of log lines and returns found patterns.
func (p *BrainParser) Parse(logLines []string) []*ParseResult {
	// Use cached preprocessor with pre-compiled regexes for performance
	processedLogs := p.preprocessor.PreprocessLogs(logLines)

	initialGroups := CreateInitialGroups(processedLogs, &p.config)

	var allTemplates []*ParseResult

	// Convert map to slice for processing
	groupSlice := make([]*LogGroup, 0, len(initialGroups))
	for _, group := range initialGroups {
		groupSlice = append(groupSlice, group)
	}

	// Determine if we should use parallel processing
	shouldUseParallel := false
	for _, group := range groupSlice {
		if len(group.Logs) >= p.config.ParallelProcessingThreshold {
			shouldUseParallel = true
			break
		}
	}

	if shouldUseParallel {
		// Parallel processing for large groups
		allTemplates = p.processGroupsParallel(groupSlice, processedLogs)
	} else {
		// Sequential processing for small groups
		for _, group := range groupSlice {
			// Steps 3 and 4: Build tree for each group
			tree := p.BuildTreeForGroup(group)

			// Step 5: Generate templates from tree
			// GenerateTemplatesFromTree performs complete template extraction with full
			// bidirectional tree traversal, iterative parent updates, and proper log tracking
			templates := p.GenerateTemplatesFromTree(tree, processedLogs)
			allTemplates = append(allTemplates, templates...)

			// Release tree resources back to pools after processing
			ReleaseBidirectionalTree(tree)
		}
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
			// Use pooled int slice for better memory management
			if cap(existing.LogIDs) < len(existing.LogIDs)+len(res.LogIDs) {
				newSlice := GetIntSlice()
				// Ensure capacity
				for cap(newSlice) < len(existing.LogIDs)+len(res.LogIDs) {
					PutIntSlice(newSlice)
					newSlice = make([]int, 0, len(existing.LogIDs)+len(res.LogIDs))
				}
				newSlice = append(newSlice, existing.LogIDs...)
				existing.LogIDs = newSlice
			}
			existing.LogIDs = append(existing.LogIDs, res.LogIDs...)
		} else {
			// Copy to avoid modifying the original slice
			newRes := *res
			// Use pooled int slice for LogIDs
			logIDsCopy := GetIntSlice()
			// Ensure capacity
			if cap(logIDsCopy) < len(res.LogIDs) {
				PutIntSlice(logIDsCopy)
				logIDsCopy = make([]int, 0, len(res.LogIDs))
			}
			logIDsCopy = append(logIDsCopy, res.LogIDs...)
			newRes.LogIDs = logIDsCopy
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

	var dynamicThreshold int

	if p.config.UseStatisticalThreshold {
		// Enhanced statistical threshold calculation from Drain+
		dynamicThreshold = p.calculateStatisticalThreshold(uniqueWordsCount)
	} else {
		// Original Brain algorithm
		// Use natural logarithm as suggested in the paper discussion
		dynamicThreshold = int(math.Log(float64(uniqueWordsCount)) * p.config.DynamicThresholdFactor)
	}

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

// calculateStatisticalThreshold uses statistical analysis for better threshold determination
func (p *BrainParser) calculateStatisticalThreshold(uniqueWordsCount int) int {
	// Base calculation using logarithm
	baseThreshold := math.Log(float64(uniqueWordsCount)) * p.config.DynamicThresholdFactor

	// Apply statistical adjustments based on Drain+ research
	// 1. Adjust for small datasets (< 10 unique words)
	if uniqueWordsCount < 10 {
		// For small datasets, use a more conservative threshold
		baseThreshold *= 1.5
	}

	// 2. Adjust for large datasets (> 100 unique words)
	if uniqueWordsCount > 100 {
		// For large datasets, use square root scaling to prevent over-splitting
		baseThreshold = math.Sqrt(float64(uniqueWordsCount)) * p.config.DynamicThresholdFactor * 0.7
	}

	// 3. Apply smoothing based on standard deviation principle
	// This helps handle edge cases where log growth might be too aggressive
	smoothedThreshold := baseThreshold
	if uniqueWordsCount > 20 && uniqueWordsCount < 100 {
		// Apply sigmoid-like smoothing for mid-range values
		x := float64(uniqueWordsCount-50) / 30.0
		sigmoid := 1.0 / (1.0 + math.Exp(-x))
		smoothedThreshold = baseThreshold * (0.7 + 0.6*sigmoid)
	}

	return int(smoothedThreshold)
}

// processGroupsParallel processes log groups in parallel for better performance on large datasets
func (p *BrainParser) processGroupsParallel(groups []*LogGroup, allLogs []*LogMessage) []*ParseResult {
	// Create channels for work distribution and result collection
	type workItem struct {
		group *LogGroup
		index int
	}

	workChan := make(chan workItem, len(groups))
	resultsChan := make(chan []*ParseResult, len(groups))

	// Use a WaitGroup to track completion
	var wg sync.WaitGroup

	// Determine optimal number of workers
	numWorkers := p.getOptimalWorkerCount(groups)

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				// Process the group
				tree := p.BuildTreeForGroup(work.group)
				templates := p.GenerateTemplatesFromTree(tree, allLogs)
				resultsChan <- templates

				// Release tree resources back to pools after processing
				ReleaseBidirectionalTree(tree)
			}
		}()
	}

	// Send work to workers
	for i, group := range groups {
		workChan <- workItem{group: group, index: i}
	}
	close(workChan)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var allTemplates []*ParseResult
	for templates := range resultsChan {
		allTemplates = append(allTemplates, templates...)
	}

	return allTemplates
}

// getOptimalWorkerCount determines the optimal number of workers based on groups and system
func (p *BrainParser) getOptimalWorkerCount(groups []*LogGroup) int {
	// Count groups that meet the parallel processing threshold
	largeGroupCount := 0
	for _, group := range groups {
		if len(group.Logs) >= p.config.ParallelProcessingThreshold {
			largeGroupCount++
		}
	}

	// Base worker count on number of large groups
	// But cap it to avoid excessive goroutine creation
	numWorkers := max(min(largeGroupCount, 8), 2)

	return numWorkers
}

// BuildTreeForGroup builds a bidirectional tree for one log group.
func (p *BrainParser) BuildTreeForGroup(group *LogGroup) *BidirectionalTree {
	// Use pooled Node for child direction root
	childRoot := GetNode()
	childRoot.Value = unique.Make("ROOT")
	childRoot.Children = GetStringMap() // Use pooled map
	childRoot.Logs = group.Logs

	tree := &BidirectionalTree{
		RootNodes:          group.Pattern.Words,
		ParentDirection:    make(map[int]*Node),
		ChildDirectionRoot: childRoot,
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
				word := log.Words[pos].Value.Value()
				uniqueWords[word] = true
				if constantWord == "" {
					constantWord = word
				}
			}
		}

		node := GetNode()
		node.IsVariable = len(uniqueWords) > 1
		node.Position = pos
		node.Logs = logs

		// If word is constant (only one unique), save its value
		if !node.IsVariable && constantWord != "" {
			node.Value = unique.Make(constantWord)
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
			word := log.Words[posToProcess].Value.Value()
			wordsInColumn[word] = append(wordsInColumn[word], log)
		}
	}

	// Calculate dynamic threshold based on unique words count
	uniqueWordsCount := len(wordsInColumn)
	threshold := p.calculateDynamicThreshold(uniqueWordsCount)

	// If number of branches > threshold, consider all as variables
	if uniqueWordsCount > threshold {
		variableNode := GetNode()
		variableNode.IsVariable = true
		variableNode.Children = GetStringMap()
		variableNode.Position = posToProcess
		variableNode.Logs = currentLogs
		rootNode.Children["<*>"] = variableNode
		// Continue recursion for the same group, but with remaining columns
		p.updateChildDirection(tree, rootNode.Children["<*>"], currentLogs, remainingCols)
	} else {
		// Otherwise create constant branches and split the group
		for word, subGroupLogs := range wordsInColumn {
			newNode := GetNode()
			newNode.Value = unique.Make(word)
			newNode.IsVariable = false
			newNode.Children = GetStringMap()
			newNode.Position = posToProcess
			newNode.Logs = subGroupLogs
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
				word := log.Words[parentPos].Value.Value()
				uniqueWords[word] = true
				if constantWord == "" {
					constantWord = word
				}
			}
		}

		// Create/update node specific to this subgroup
		parentNode := GetNode()
		parentNode.IsVariable = len(uniqueWords) > 1
		parentNode.Position = parentPos
		parentNode.Logs = subGroupLogs

		// If word became constant in this subgroup, save its value
		if !parentNode.IsVariable && constantWord != "" {
			parentNode.Value = unique.Make(constantWord)
		}

		// Store subgroup-specific parent information in the node
		if node.ParentWords == nil {
			node.ParentWords = make([]unique.Handle[string], len(subGroupLogs[0].Words))
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
					node.ParentWords = append(node.ParentWords, unique.Make(""))
				}
				node.ParentWords[parentPos] = unique.Make("<*>")
			}
		} else if constantWord != "" {
			// Ensure we have enough capacity
			for len(node.ParentWords) <= parentPos {
				node.ParentWords = append(node.ParentWords, unique.Make(""))
			}
			node.ParentWords[parentPos] = unique.Make(constantWord)
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
			unique[log.Words[position].Value.Value()] = true
		}
	}
	return len(unique)
}
