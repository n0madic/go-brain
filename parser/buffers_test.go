package parser

import (
	"strings"
	"testing"
)

// TestPointerBasedBufferPools verifies the pointer-based implementation works correctly
func TestPointerBasedBufferPools(t *testing.T) {
	// Test StringBuilder pooling
	sb1 := GetStringBuilder()
	if sb1 == nil {
		t.Fatal("GetStringBuilder returned nil")
	}

	sb1.WriteString("test")
	result1 := sb1.String()
	PutStringBuilder(sb1)

	sb2 := GetStringBuilder()
	if sb2 == nil {
		t.Fatal("GetStringBuilder returned nil after put")
	}

	// Should be empty after reset
	if sb2.Len() != 0 {
		t.Error("StringBuilder not properly reset")
	}

	sb2.WriteString("another test")
	result2 := sb2.String()
	PutStringBuilder(sb2)

	if result1 != "test" || result2 != "another test" {
		t.Errorf("StringBuilder pooling failed: got %q and %q", result1, result2)
	}

	// Test byte buffer pooling
	buf1 := GetByteBuffer()
	buf1 = append(buf1, []byte("buffer test")...)
	PutByteBuffer(buf1)

	buf2 := GetByteBuffer()
	if len(buf2) != 0 {
		t.Error("Byte buffer not properly reset")
	}

	// Test optimized operations
	parts := []string{"hello", "world", "from", "golang"}
	joined := strings.Join(parts, " ")
	expected := "hello world from golang"
	if joined != expected {
		t.Errorf("StringJoin: expected %q, got %q", expected, joined)
	}

	concat := StringConcat("hello", " ", "world")
	expected = "hello world"
	if concat != expected {
		t.Errorf("StringConcat: expected %q, got %q", expected, concat)
	}

	t.Log("Pointer-based buffer pools work correctly!")
}

// BenchmarkPointerVsInterface compares the new pointer-based approach
func BenchmarkPointerBasedPools(b *testing.B) {
	parts := []string{"hello", "world", "from", "golang", "optimization"}

	b.Run("StringConcat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = StringConcat(parts[0], " ", parts[1], " ", parts[2])
		}
	})

	b.Run("ByteBuffer", func(b *testing.B) {
		testData := []byte("Hello, World! Testing byte buffer performance.")
		for i := 0; i < b.N; i++ {
			buf := GetByteBuffer()
			buf = append(buf, testData...)
			PutByteBuffer(buf)
		}
	})
}

// TestStringCachePointerSafe tests string cache with pointer-safe design
func TestStringCachePointerSafe(t *testing.T) {
	cache := NewStringCache(100)

	// Test basic functionality
	key1 := "test_key_1"
	result1 := cache.Get(key1)
	if result1 != key1 {
		t.Errorf("Expected %q, got %q", key1, result1)
	}

	// Test cache hit
	result2 := cache.Get(key1)
	if result2 != key1 {
		t.Errorf("Expected cache hit for %q, got %q", key1, result2)
	}

	// Test size
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Test clear
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}

	t.Log("String cache works correctly with pointer-safe design")
}
