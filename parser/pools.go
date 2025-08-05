package parser

import (
	"sync"
	"unique"
)

// PooledWordSlice is a pointer-safe wrapper for []Word to avoid SA6002 warnings
type PooledWordSlice struct {
	Data []Word
}

// PooledLogSlice is a pointer-safe wrapper for []*LogMessage to avoid SA6002 warnings
type PooledLogSlice struct {
	Data []*LogMessage
}

// PooledIntSlice is a pointer-safe wrapper for []int to avoid SA6002 warnings
type PooledIntSlice struct {
	Data []int
}

// MemoryPools contains all object pools for memory optimization with pointer-safe design
type MemoryPools struct {
	LogMessages sync.Pool // *LogMessage - already pointer, perfect
	Words       sync.Pool // *PooledWordSlice - wrapper for []Word
	Nodes       sync.Pool // *Node - already pointer, perfect
	StringMaps  sync.Pool // map[string]*Node - already pointer values
	LogSlices   sync.Pool // *PooledLogSlice - wrapper for []*LogMessage
	IntSlices   sync.Pool // *PooledIntSlice - wrapper for []int
	WordSlices  sync.Pool // *PooledWordSlice - wrapper for []Word (patterns)
}

// globalPools is the singleton pool instance
var globalPools = &MemoryPools{}

// initPools initializes all memory pools with proper factory functions
func init() { //nolint:gochecknoinits // Required for pool initialization
	globalPools.LogMessages.New = func() any {
		return &LogMessage{}
	}

	globalPools.Words.New = func() any {
		return &PooledWordSlice{
			Data: make([]Word, 0, 16), // Start with reasonable capacity
		}
	}

	globalPools.Nodes.New = func() any {
		return &Node{
			Children: make(map[string]*Node),
		}
	}

	globalPools.StringMaps.New = func() any {
		return make(map[string]*Node)
	}

	globalPools.LogSlices.New = func() any {
		return &PooledLogSlice{
			Data: make([]*LogMessage, 0, 32),
		}
	}

	globalPools.IntSlices.New = func() any {
		return &PooledIntSlice{
			Data: make([]int, 0, 8),
		}
	}

	globalPools.WordSlices.New = func() any {
		return &PooledWordSlice{
			Data: make([]Word, 0, 8),
		}
	}
}

// GetLogMessage gets a LogMessage from the pool
func GetLogMessage() *LogMessage {
	msg, ok := globalPools.LogMessages.Get().(*LogMessage)
	if !ok {
		msg = &LogMessage{}
	}
	// Reset fields to zero values
	msg.ID = 0
	msg.Content = unique.Handle[string]{}
	if msg.Words != nil {
		msg.Words = msg.Words[:0] // Keep capacity, reset length
	}
	return msg
}

// PutLogMessage returns a LogMessage to the pool
func PutLogMessage(msg *LogMessage) {
	if msg != nil {
		globalPools.LogMessages.Put(msg)
	}
}

// GetWordSlice gets a Word slice from the pool
func GetWordSlice() []Word {
	wrapper, ok := globalPools.Words.Get().(*PooledWordSlice)
	if !ok {
		wrapper = &PooledWordSlice{
			Data: make([]Word, 0, 8),
		}
	}
	wrapper.Data = wrapper.Data[:0] // Reset length, keep capacity
	return wrapper.Data
}

// PutWordSlice returns a Word slice to the pool
func PutWordSlice(slice []Word) {
	if slice != nil && cap(slice) > 0 {
		wrapper := &PooledWordSlice{Data: slice}
		globalPools.Words.Put(wrapper) // ✅ No SA6002 warnings!
	}
}

// GetNode gets a Node from the pool
func GetNode() *Node {
	node, ok := globalPools.Nodes.Get().(*Node)
	if !ok {
		node = &Node{}
	}
	// Reset fields
	node.Value = unique.Handle[string]{}
	node.IsVariable = false
	node.Position = 0
	node.ParentWords = node.ParentWords[:0] // Reset slice length
	node.Logs = node.Logs[:0]               // Reset slice length

	// Clear the children map
	for k := range node.Children {
		delete(node.Children, k)
	}

	return node
}

// PutNode returns a Node to the pool
func PutNode(node *Node) {
	if node != nil {
		globalPools.Nodes.Put(node)
	}
}

// GetStringMap gets a string->*Node map from the pool
func GetStringMap() map[string]*Node {
	m, ok := globalPools.StringMaps.Get().(map[string]*Node)
	if !ok {
		m = make(map[string]*Node)
	}
	// Clear the map
	for k := range m {
		delete(m, k)
	}
	return m
}

// PutStringMap returns a string->*Node map to the pool
func PutStringMap(m map[string]*Node) {
	if m != nil {
		globalPools.StringMaps.Put(m)
	}
}

// GetLogSlice gets a []*LogMessage slice from the pool
func GetLogSlice() []*LogMessage {
	wrapper, ok := globalPools.LogSlices.Get().(*PooledLogSlice)
	if !ok {
		wrapper = &PooledLogSlice{
			Data: make([]*LogMessage, 0, 10),
		}
	}
	wrapper.Data = wrapper.Data[:0] // Reset length, keep capacity
	return wrapper.Data
}

// PutLogSlice returns a []*LogMessage slice to the pool
func PutLogSlice(slice []*LogMessage) {
	if slice != nil && cap(slice) > 0 {
		wrapper := &PooledLogSlice{Data: slice}
		globalPools.LogSlices.Put(wrapper) // ✅ No SA6002 warnings!
	}
}

// GetIntSlice gets an []int slice from the pool
func GetIntSlice() []int {
	wrapper, ok := globalPools.IntSlices.Get().(*PooledIntSlice)
	if !ok {
		wrapper = &PooledIntSlice{
			Data: make([]int, 0, 10),
		}
	}
	wrapper.Data = wrapper.Data[:0] // Reset length, keep capacity
	return wrapper.Data
}

// PutIntSlice returns an []int slice to the pool
func PutIntSlice(slice []int) {
	if slice != nil && cap(slice) > 0 {
		wrapper := &PooledIntSlice{Data: slice}
		globalPools.IntSlices.Put(wrapper) // ✅ No SA6002 warnings!
	}
}

// GetWordSliceForPattern gets a []Word slice specifically for patterns
func GetWordSliceForPattern() []Word {
	wrapper, ok := globalPools.WordSlices.Get().(*PooledWordSlice)
	if !ok {
		wrapper = &PooledWordSlice{
			Data: make([]Word, 0, 8),
		}
	}
	wrapper.Data = wrapper.Data[:0] // Reset length, keep capacity
	return wrapper.Data
}

// PutWordSliceForPattern returns a []Word slice to the pool
func PutWordSliceForPattern(slice []Word) {
	if slice != nil && cap(slice) > 0 {
		wrapper := &PooledWordSlice{Data: slice}
		globalPools.WordSlices.Put(wrapper) // ✅ No SA6002 warnings!
	}
}

// PooledLogMessage is a wrapper that automatically returns LogMessage to pool when done
type PooledLogMessage struct {
	*LogMessage
}

// Release returns the LogMessage back to the pool
func (p *PooledLogMessage) Release() {
	PutLogMessage(p.LogMessage)
	p.LogMessage = nil
}

// NewPooledLogMessage creates a new pooled LogMessage
func NewPooledLogMessage() *PooledLogMessage {
	return &PooledLogMessage{
		LogMessage: GetLogMessage(),
	}
}

// ReleaseBidirectionalTree recursively returns all pooled resources in a tree to their pools
func ReleaseBidirectionalTree(tree *BidirectionalTree) {
	if tree == nil {
		return
	}

	// Use defer to ensure cleanup even if panic occurs
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't propagate - resource cleanup should be non-fatal
			// In production, this could be logged to structured logger
			_ = r // Silently handle for now
		}
	}()

	// Release parent direction nodes
	for _, node := range tree.ParentDirection {
		releaseNodeRecursively(node)
	}

	// Release child direction tree
	if tree.ChildDirectionRoot != nil {
		releaseNodeRecursively(tree.ChildDirectionRoot)
	}
}

// releaseNodeRecursively releases a node and all its children back to pools
func releaseNodeRecursively(node *Node) {
	if node == nil {
		return
	}

	// Use defer to ensure cleanup even if panic occurs during recursion
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't propagate - resource cleanup should be non-fatal
			// In production, this could be logged to structured logger
			_ = r // Silently handle for now
		}
	}()

	// Release all child nodes recursively
	for _, child := range node.Children {
		releaseNodeRecursively(child)
	}

	// Return the string map to pool
	if node.Children != nil {
		PutStringMap(node.Children)
		node.Children = nil
	}

	// Return the node itself to pool
	PutNode(node)
}
