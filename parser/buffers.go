package parser

import (
	"fmt"
	"strings"
	"sync"
)

// BufferPools contains all buffer pools for string operations using pointer-safe design
type BufferPools struct {
	StringBuilders sync.Pool // *strings.Builder
	ByteBuffers    sync.Pool // *PooledByteBuffer
}

// PooledByteBuffer wraps []byte to make it pointer-safe for sync.Pool
type PooledByteBuffer struct {
	Data []byte
}

// PooledStringSlice wraps []string to make it pointer-safe for sync.Pool
type PooledStringSlice struct {
	Data []string
}

// globalBufferPools is the singleton buffer pool instance with optimized pointer design
var globalBufferPools = &BufferPools{}

// initBufferPools initializes all buffer pools with pointer-safe design
func init() { //nolint:gochecknoinits // Required for pool initialization
	globalBufferPools.StringBuilders.New = func() any {
		return &strings.Builder{}
	}

	globalBufferPools.ByteBuffers.New = func() any {
		return &PooledByteBuffer{
			Data: make([]byte, 0, 256), // Start with 256 byte capacity
		}
	}
}

// GetStringBuilder gets a StringBuilder from the pool
func GetStringBuilder() *strings.Builder {
	sb, ok := globalBufferPools.StringBuilders.Get().(*strings.Builder)
	if !ok {
		sb = &strings.Builder{}
	}
	sb.Reset() // Clear any previous content
	return sb
}

// PutStringBuilder returns a StringBuilder to the pool
func PutStringBuilder(sb *strings.Builder) {
	if sb != nil {
		globalBufferPools.StringBuilders.Put(sb) // ✅ No SA6002 warnings!
	}
}

// GetByteBuffer gets a byte slice from the pool
func GetByteBuffer() []byte {
	wrapper, ok := globalBufferPools.ByteBuffers.Get().(*PooledByteBuffer)
	if !ok {
		wrapper = &PooledByteBuffer{
			Data: make([]byte, 0, 256),
		}
	}
	wrapper.Data = wrapper.Data[:0] // Reset length, keep capacity
	return wrapper.Data
}

// PutByteBuffer returns a byte slice to the pool
func PutByteBuffer(buf []byte) {
	if buf != nil && cap(buf) > 0 {
		wrapper := &PooledByteBuffer{Data: buf}
		globalBufferPools.ByteBuffers.Put(wrapper) // ✅ No SA6002 warnings!
	}
}

// StringConcat performs optimized string concatenation
func StringConcat(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	sb := GetStringBuilder()
	defer PutStringBuilder(sb)

	// Pre-calculate capacity
	totalLen := 0
	for _, str := range strs {
		totalLen += len(str)
	}
	sb.Grow(totalLen)

	for _, str := range strs {
		sb.WriteString(str)
	}

	return sb.String()
}

// BufferedStringBuilder provides a reusable string builder with automatic pooling
type BufferedStringBuilder struct {
	builder *strings.Builder
	inUse   bool
}

// NewBufferedStringBuilder creates a new buffered string builder
func NewBufferedStringBuilder() *BufferedStringBuilder {
	return &BufferedStringBuilder{
		builder: GetStringBuilder(),
		inUse:   true,
	}
}

// WriteString writes a string to the buffer
func (bsb *BufferedStringBuilder) WriteString(s string) {
	if bsb.inUse && bsb.builder != nil {
		bsb.builder.WriteString(s)
	}
}

// WriteByte writes a byte to the buffer
func (bsb *BufferedStringBuilder) WriteByte(b byte) error {
	if bsb.inUse && bsb.builder != nil {
		if err := bsb.builder.WriteByte(b); err != nil {
			return fmt.Errorf("failed to write byte to buffer: %w", err)
		}
	}
	return nil
}

// String returns the accumulated string and releases the builder
func (bsb *BufferedStringBuilder) String() string {
	if !bsb.inUse || bsb.builder == nil {
		return ""
	}

	result := bsb.builder.String()
	PutStringBuilder(bsb.builder)
	bsb.builder = nil
	bsb.inUse = false
	return result
}

// Reset resets the builder for reuse (gets a new one from pool)
func (bsb *BufferedStringBuilder) Reset() {
	if bsb.inUse && bsb.builder != nil {
		PutStringBuilder(bsb.builder)
	}
	bsb.builder = GetStringBuilder()
	bsb.inUse = true
}

// Release manually releases the builder back to pool
func (bsb *BufferedStringBuilder) Release() {
	if bsb.inUse && bsb.builder != nil {
		PutStringBuilder(bsb.builder)
		bsb.builder = nil
		bsb.inUse = false
	}
}

// KeyGeneration provides optimized key generation for grouping
func KeyGeneration(frequency int, words []Word) string {
	if len(words) == 0 {
		return "freq:0-"
	}

	sb := GetStringBuilder()
	defer PutStringBuilder(sb)

	// Pre-calculate approximate capacity
	// freq:X- + (pos:X,val:Y|) * numWords
	approxCapacity := 10 + len(words)*20 // Rough estimate
	sb.Grow(approxCapacity)

	sb.WriteString("freq:")
	writeInt(sb, frequency)
	sb.WriteByte('-')

	for _, word := range words {
		sb.WriteString("pos:")
		writeInt(sb, word.Position)
		sb.WriteString(",val:")
		sb.WriteString(word.Value.Value())
		sb.WriteByte('|')
	}

	return sb.String()
}

// writeInt efficiently writes an integer to strings.Builder
func writeInt(sb *strings.Builder, value int) {
	if value == 0 {
		sb.WriteByte('0')
		return
	}

	// Handle negative numbers
	negative := value < 0
	if negative {
		value = -value
		sb.WriteByte('-')
	}

	// Convert digits in reverse order
	buf := GetByteBuffer()
	defer PutByteBuffer(buf)

	for value > 0 {
		buf = append(buf, byte('0'+(value%10)))
		value /= 10
	}

	// Write digits in correct order
	for i := len(buf) - 1; i >= 0; i-- {
		sb.WriteByte(buf[i])
	}
}

// StringCache provides a simple string cache for frequently used strings
type StringCache struct {
	cache   map[string]string
	mutex   sync.RWMutex
	maxSize int
}

// NewStringCache creates a new string cache
func NewStringCache(maxSize int) *StringCache {
	return &StringCache{
		cache:   make(map[string]string, maxSize),
		maxSize: maxSize,
	}
}

// Get retrieves a string from cache or stores it if not present
func (sc *StringCache) Get(key string) string {
	sc.mutex.RLock()
	if cached, exists := sc.cache[key]; exists {
		sc.mutex.RUnlock()
		return cached
	}
	sc.mutex.RUnlock()

	// Not in cache, add it
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// Double-check in case another goroutine added it
	if cached, exists := sc.cache[key]; exists {
		return cached
	}

	// Check size limit
	if len(sc.cache) >= sc.maxSize {
		// Simple eviction: clear cache when full
		// In production, might use LRU or other eviction policy
		sc.cache = make(map[string]string, sc.maxSize)
	}

	sc.cache[key] = key
	return key
}

// Clear clears the cache
func (sc *StringCache) Clear() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.cache = make(map[string]string, sc.maxSize)
}

// Size returns current cache size
func (sc *StringCache) Size() int {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	return len(sc.cache)
}
