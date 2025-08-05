package parser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
)

// StreamingProcessor handles large datasets efficiently using streaming approach
type StreamingProcessor struct {
	parser       *BrainParser
	batchSize    int
	maxWorkers   int
	bufferPool   sync.Pool
	resultBuffer chan *ParseResult
}

// StreamingConfig contains configuration for streaming processing
type StreamingConfig struct {
	BatchSize         int  // Number of logs to process in each batch
	MaxWorkers        int  // Maximum number of concurrent workers
	EnableCompression bool // Enable compressed intermediate storage
	MemoryThreshold   int  // Memory threshold in MB to switch to streaming
}

// NewStreamingProcessor creates a new streaming processor
func NewStreamingProcessor(config Config, streamConfig StreamingConfig) *StreamingProcessor {
	if streamConfig.BatchSize == 0 {
		streamConfig.BatchSize = 1000 // Default batch size
	}
	if streamConfig.MaxWorkers == 0 {
		streamConfig.MaxWorkers = 4 // Default workers
	}

	sp := &StreamingProcessor{
		parser:       New(config),
		batchSize:    streamConfig.BatchSize,
		maxWorkers:   streamConfig.MaxWorkers,
		resultBuffer: make(chan *ParseResult, streamConfig.MaxWorkers*2),
	}

	// Initialize buffer pool for line reading using pointer-safe wrapper
	sp.bufferPool.New = func() any {
		return &PooledByteBuffer{
			Data: make([]byte, 4096), // 4KB buffer for reading lines
		}
	}

	return sp
}

// ProcessReader processes logs from an io.Reader in streaming fashion
func (sp *StreamingProcessor) ProcessReader(ctx context.Context, reader io.Reader) ([]*ParseResult, error) {
	scanner := bufio.NewScanner(reader)

	// Use pooled buffer for scanning with pointer-safe wrapper
	wrapper, ok := sp.bufferPool.Get().(*PooledByteBuffer)
	if !ok {
		wrapper = &PooledByteBuffer{
			Data: make([]byte, 4096),
		}
	}
	buffer := wrapper.Data
	defer sp.bufferPool.Put(wrapper)  // ✅ No SA6002 warnings!
	scanner.Buffer(buffer, 1024*1024) // 1MB max line size

	var batch []string
	var allResults []*ParseResult
	var wg sync.WaitGroup

	// Channel for batches
	batchChan := make(chan []string, sp.maxWorkers)
	resultChan := make(chan []*ParseResult, sp.maxWorkers)

	// Start worker goroutines
	for i := 0; i < sp.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range batchChan {
				select {
				case <-ctx.Done():
					return
				default:
					results := sp.parser.Parse(batch)
					resultChan <- results
				}
			}
		}()
	}

	// Process lines in batches
	go func() {
		defer close(batchChan)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Text()
				if line != "" { // Skip empty lines
					batch = append(batch, line)

					if len(batch) >= sp.batchSize {
						// Send batch for processing
						batchCopy := make([]string, len(batch))
						copy(batchCopy, batch)
						batchChan <- batchCopy
						batch = batch[:0] // Reset batch
					}
				}
			}
		}

		// Process remaining batch
		if len(batch) > 0 {
			batchCopy := make([]string, len(batch))
			copy(batchCopy, batch)
			batchChan <- batchCopy
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for results := range resultChan {
		allResults = append(allResults, results...)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error during streaming processing: %w", err)
	}

	// Aggregate final results
	return sp.parser.aggregateResults(allResults), nil
}

// ProcessLargeSlice processes very large slices efficiently using streaming approach
func (sp *StreamingProcessor) ProcessLargeSlice(ctx context.Context, logs []string) ([]*ParseResult, error) {
	if len(logs) < sp.batchSize {
		// For small datasets, use regular processing
		return sp.parser.Parse(logs), nil
	}

	var allResults []*ParseResult
	var wg sync.WaitGroup

	// Channel for batches
	batchChan := make(chan []string, sp.maxWorkers)
	resultChan := make(chan []*ParseResult, sp.maxWorkers)

	// Start worker goroutines
	for i := 0; i < sp.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range batchChan {
				select {
				case <-ctx.Done():
					return
				default:
					results := sp.parser.Parse(batch)
					resultChan <- results
				}
			}
		}()
	}

	// Send batches
	go func() {
		defer close(batchChan)
		for i := 0; i < len(logs); i += sp.batchSize {
			select {
			case <-ctx.Done():
				return
			default:
				end := i + sp.batchSize
				if end > len(logs) {
					end = len(logs)
				}

				batch := make([]string, end-i)
				copy(batch, logs[i:end])
				batchChan <- batch
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for results := range resultChan {
		allResults = append(allResults, results...)
	}

	// Aggregate final results
	return sp.parser.aggregateResults(allResults), nil
}

// AdaptiveProcessor automatically selects the best processing strategy
type AdaptiveProcessor struct {
	regularParser   *BrainParser
	streamProcessor *StreamingProcessor
	memoryThreshold int // MB threshold to switch to streaming
	sizeThreshold   int // Number of logs threshold
}

// NewAdaptiveProcessor creates a processor that adapts to dataset characteristics
func NewAdaptiveProcessor(config Config) *AdaptiveProcessor {
	streamConfig := StreamingConfig{
		BatchSize:         1000,
		MaxWorkers:        4,
		EnableCompression: false,
		MemoryThreshold:   100, // 100MB threshold
	}

	return &AdaptiveProcessor{
		regularParser:   New(config),
		streamProcessor: NewStreamingProcessor(config, streamConfig),
		memoryThreshold: streamConfig.MemoryThreshold,
		sizeThreshold:   5000, // Switch to streaming for 5000+ logs
	}
}

// ProcessAdaptive automatically chooses the best processing strategy
func (ap *AdaptiveProcessor) ProcessAdaptive(ctx context.Context, logs []string) ([]*ParseResult, error) {
	// Estimate memory usage
	avgLogSize := ap.estimateAverageLogSize(logs)
	estimatedMemoryMB := (len(logs) * avgLogSize * 10) / (1024 * 1024) // Rough estimate with 10x multiplier

	// Decision logic
	useStreaming := len(logs) > ap.sizeThreshold || estimatedMemoryMB > ap.memoryThreshold

	if useStreaming {
		return ap.streamProcessor.ProcessLargeSlice(ctx, logs)
	}
	return ap.regularParser.Parse(logs), nil
}

// estimateAverageLogSize estimates average log size for memory calculation
func (ap *AdaptiveProcessor) estimateAverageLogSize(logs []string) int {
	if len(logs) == 0 {
		return 50 // Default estimate
	}

	// Sample first 100 logs or all if less
	sampleSize := min(100, len(logs))
	totalSize := 0

	for i := 0; i < sampleSize; i++ {
		totalSize += len(logs[i])
	}

	return totalSize / sampleSize
}
