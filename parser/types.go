package parser

import "fmt"

// LogMessage represents one log line after preprocessing.
type LogMessage struct {
	ID      int    // Original log index
	Content string // Original content
	Words   []Word // Words the log is split into
}

// Word represents one word in a log with its metadata.
type Word struct {
	Value     string // Text value of the word
	Position  int    // Position (index) in the log line
	Frequency int    // Global frequency of the word across all logs
}

// WordCombination - is a set of words from one log with the same frequency.
type WordCombination struct {
	Frequency int
	Words     []Word
}

// Key generates a unique key for WordCombination for use in map.
func (wc WordCombination) Key() string {
	key := fmt.Sprintf("freq:%d-", wc.Frequency)
	for _, word := range wc.Words {
		key += fmt.Sprintf("pos:%d,val:%s|", word.Position, word.Value)
	}
	return key
}

// LogGroup represents a group of logs unified by a common pattern.
type LogGroup struct {
	Pattern LogPattern
	Logs    []*LogMessage
}

// LogPattern - is the longest common pattern (Longest Common Pattern),
// which is the core (root) for the bidirectional tree.
type LogPattern struct {
	Words     []Word
	Frequency int
}

// Key generates a unique key for LogPattern.
func (lp LogPattern) Key() string {
	return WordCombination{Frequency: lp.Frequency, Words: lp.Words}.Key()
}

// Node - node in the bidirectional tree.
type Node struct {
	Value       string
	IsVariable  bool
	Position    int              // Column position for this node
	Children    map[string]*Node // Child nodes (for child direction)
	ParentWords []string         // Words in parent direction
	Logs        []*LogMessage    // Logs passing through this node
}

// BidirectionalTree represents a bidirectional parallel tree for one log group.
type BidirectionalTree struct {
	RootNodes          []Word                   // Core/root of the tree (Longest Common Pattern)
	ParentDirection    map[int]*Node            // Nodes in "parent" direction (frequency > root)
	ChildDirectionRoot *Node                    // Root for "child" direction (frequency < root)
	LogGroups          map[string][]*LogMessage // Final groups of logs by templates
	ParentColumns      []int                    // Columns that are in parent direction for iterative updates
	RootPattern        LogPattern               // Original root pattern for reference
}

// ParseResult represents the final result of parsing.
type ParseResult struct {
	Template string
	Count    int
	LogIDs   []int
}

// Config contains the configuration of the Brain algorithm.
type Config struct {
	Delimiters                  string            // Regex for splitting tokens
	CommonVariables             map[string]string // Map of patterns for filtering common variables: "name" -> "regex"
	ChildBranchThreshold        int               // Threshold for creating new branches in child direction (fallback value)
	Weight                      float64           // Weight parameter for frequency threshold (0.0-1.0)
	UseDynamicThreshold         bool              // Whether to use dynamic threshold calculation
	DynamicThresholdFactor      float64           // Factor for dynamic threshold (default: 2.0)
	UseEnhancedPostProcessing   bool              // Enable enhanced post-processing from Drain+ (default: false)
	UseStatisticalThreshold     bool              // Use statistical analysis for threshold calculation (default: false)
	ParallelProcessingThreshold int               // Minimum log count in group to enable parallel processing (default: 1000)

	// Enhanced Features Tuning Parameters
	EntropyThreshold        float64 // Threshold for entropy-based variable detection (default: 0.85, lower = more aggressive)
	MinEntropyLength        int     // Minimum word length for entropy analysis (default: 10)
	MaxConsecutiveWildcards int     // Maximum consecutive <*> tokens in template (default: 5, 0 = no limit)
	MinContentWordsRatio    float64 // Minimum ratio of non-<*> words in template (default: 0.3)
	TimestampMinDigits      int     // Minimum digits for timestamp detection (default: 8)
	TimestampMinSeparators  int     // Minimum separators for timestamp detection (default: 2)

	// Internal flags
	isReparsing bool // Internal flag to prevent infinite recursion during reparsing
}
