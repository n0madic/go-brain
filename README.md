# Brain in Go: Log Parsing with Bidirectional Parallel Tree

A Go implementation of the Brain algorithm for automated log parsing, based on the research paper "Brain: Log Parsing with Bidirectional Parallel Tree" (IEEE INFOCOM 2023).

## Algorithm Overview

The Brain algorithm is a sophisticated log parsing technique that automatically identifies patterns in log messages without requiring predefined templates. It works through the following steps:

1. **Preprocessing**: Tokenizes log messages and identifies common variables (IPs, numbers, IDs, etc.)
2. **Frequency Analysis**: Calculates global word frequencies across all logs
3. **Initial Grouping**: Groups logs by their Longest Common Pattern (LCP) - words with identical frequencies
4. **Bidirectional Tree Construction**:
   - **Root**: The LCP serves as the tree root
   - **Parent Direction**: Words with frequency > root frequency (likely constants)
   - **Child Direction**: Words with frequency ≤ root frequency (likely variables)
5. **Dynamic Threshold**: Uses `log(unique_words_count) * factor` for intelligent branch creation
6. **Iterative Parent Updates**: Recalculates parent nodes for subgroups to catch variable→constant transitions
7. **Template Generation**: Traverses the tree to generate final templates with `<*>` for variables

### Key Features

- **High Accuracy**: Achieves 0.981 Grouping Accuracy as reported in the paper
- **No Prior Knowledge Required**: Works without predefined patterns or templates
- **Handles Complex Logs**: Effectively processes logs with mixed constants and variables
- **Memory Efficient**: Uses bidirectional tree structure for optimal memory usage
- **Fast Processing**: Suitable for real-time log analysis
- **Enhanced Variable Detection**: Extended patterns for modern identifiers (emails, URLs, UUIDs, etc.)
- **Adaptive Thresholds**: Statistical analysis for optimal grouping decisions
- **Parallel Processing**: Automatic scaling for large datasets

## Installation

```bash
go get github.com/n0madic/go-brain
```

## Usage

### As a Library

```go
package main

import (
    "fmt"
    "github.com/n0madic/go-brain/parser"
)

func main() {
    // Configure the parser
    config := parser.Config{
        Delimiters:           `[\s,:=]+`,  // Regex for token delimiters
        ChildBranchThreshold: 3,           // Threshold for child branches
        UseDynamicThreshold:  true,        // Enable dynamic threshold
        DynamicThresholdFactor: 2.0,       // Factor for dynamic calculation

        // Enhanced features (optional)
        UseEnhancedPostProcessing: true,   // Better variable detection
        UseStatisticalThreshold:   true,   // Adaptive threshold calculation
        ParallelProcessingThreshold: 1000, // Parallel processing for large datasets
    }

    // Create parser instance
    brainParser := parser.New(config)

    // Parse log lines
    logLines := []string{
        "User login successful: user123",
        "User login successful: user456",
        "Database connection failed: timeout after 30s",
        "Database connection failed: timeout after 45s",
    }

    results := brainParser.Parse(logLines)

    // Print results
    for _, result := range results {
        fmt.Printf("Template: %s (Count: %d)\n", result.Template, result.Count)
    }
}
```

### Command Line Interface

The project includes a powerful CLI tool for processing log files:

```bash
# Build the CLI tool
go build -o brain-cli ./cmd/brain-cli

# Process a text log file
./brain-cli -input logs/app.log

# Process a CSV file with custom message column
./brain-cli -input logs/events.csv -csv-column "log_message"

# Show only templates appearing 10+ times
./brain-cli -input logs/app.log -min-count 10

# Output in JSON format
./brain-cli -input logs/app.log -format json

# Use custom delimiters
./brain-cli -input logs/app.log -delimiters '[\s,;:|]+'

# Extract messages from structured logs using regex
./brain-cli -input logs/structured.log -log-regex '^(?P<timestamp>[^\s]+)\s+\[(?P<level>[^\]]+)\]\s+(?P<service>[^:]+):\s*(?P<message>.+)$'

# Enable all enhanced features for better variable detection
./brain-cli -input logs/app.log -enhanced

# Enable specific enhanced features
./brain-cli -input logs/app.log -enhanced-post -statistical-threshold

# Use custom parallel processing threshold
./brain-cli -input logs/app.log -parallel-threshold 500

# Fine-tune entropy-based variable detection (more aggressive)
./brain-cli -input logs/app.log -enhanced-post -entropy-threshold 0.7

# Limit consecutive wildcards in templates
./brain-cli -input logs/app.log -enhanced-post -max-consecutive-wildcards 3

# Ensure minimum content ratio in templates
./brain-cli -input logs/app.log -enhanced-post -min-content-ratio 0.4

# Custom timestamp detection parameters
./brain-cli -input logs/app.log -enhanced-post -timestamp-min-digits 6 -timestamp-min-separators 1
```

#### CLI Options

##### Basic Options
- `-input`: Input file path (required)
- `-type`: File type: `auto`, `text`, `csv` (default: auto-detect)
- `-csv-column`: CSV column name containing log messages (default: "message")
- `-log-regex`: Regex to extract message from structured logs (must have 'message' capture group)
- `-delimiters`: Regex pattern for token delimiters (default: `[\s,:=]+`)
- `-threshold`: Child branch threshold (default: 3)
- `-dynamic`: Use dynamic threshold calculation (default: true)
- `-dynamic-factor`: Dynamic threshold factor (default: 2.0)
- `-min-count`: Minimum template count to display (default: 1)
- `-format`: Output format: `table`, `json`, `csv` (default: table)
- `-verbose`: Show log IDs for each template

##### Enhanced Features
- `-enhanced-post`: Enable enhanced post-processing for advanced variable detection
- `-statistical-threshold`: Use statistical analysis for adaptive threshold calculation
- `-parallel-threshold`: Minimum log count in group to enable parallel processing (default: 1000)
- `-enhanced`: Enable all enhanced features (equivalent to `--enhanced-post --statistical-threshold`)

##### Enhanced Features Tuning Parameters
- `-entropy-threshold`: Threshold for entropy-based variable detection, lower = more aggressive (default: 0.85)
- `-min-entropy-length`: Minimum word length for entropy analysis (default: 10)
- `-max-consecutive-wildcards`: Maximum consecutive `<*>` tokens in template, 0 = no limit (default: 5)
- `-min-content-ratio`: Minimum ratio of non-`<*>` words in template (default: 0.25)
- `-timestamp-min-digits`: Minimum digits for timestamp detection (default: 8)
- `-timestamp-min-separators`: Minimum separators for timestamp detection (default: 2)

## Examples

### Sample Input (text log)
```
2024-01-15 10:30:22 INFO User login successful: user123
2024-01-15 10:30:25 INFO User login successful: user456
2024-01-15 10:31:15 ERROR Database connection failed: timeout after 30s
2024-01-15 10:31:18 ERROR Database connection failed: timeout after 45s
2024-01-15 10:32:10 INFO HTTP request processed: GET /api/users/123 200 OK
2024-01-15 10:32:12 INFO HTTP request processed: GET /api/users/456 200 OK
```

### Sample Output
```
Processing 6 log lines...
Found 4 unique templates:

COUNT  TEMPLATE
--------------------------------------------------------------------------------------
2      2024-01-15 <*> INFO User login successful <*>
2      2024-01-15 <*> ERROR Database connection failed timeout after <*>
2      2024-01-15 <*> INFO HTTP request processed GET <*> 200 OK
```

## Configuration

### Parser Configuration

```go
type Config struct {
    // Regex for splitting tokens (default: [\s,:=])
    Delimiters string

    // Map of patterns for filtering common variables
    CommonVariables map[string]string

    // Threshold for creating new branches in child direction (default: 3)
    ChildBranchThreshold int

    // Weight parameter for frequency threshold (0.0-1.0, default: 0.0)
    Weight float64

    // Whether to use dynamic threshold calculation (default: false)
    UseDynamicThreshold bool

    // Factor for dynamic threshold (default: 2.0)
    DynamicThresholdFactor float64

    // Enhanced features from Drain+ research
    UseEnhancedPostProcessing bool  // Enable advanced variable detection (default: false)
    UseStatisticalThreshold bool    // Use statistical threshold calculation (default: false)
    ParallelProcessingThreshold int // Min logs in group for parallel processing (default: 1000)
}
```

### Default Common Variables

The parser automatically identifies common variable patterns:

**Network Patterns:**
- IPv4 addresses: `192.168.1.1`
- IPv4 with ports: `192.168.1.1:8080`
- IPv6 addresses: `2001:0db8:85a3:0000:0000:8a2e:0370:7334`
- MAC addresses: `00:1B:44:11:3A:B7`, `A0-B1-C2-D3-E4-F5`
- Hostnames with ports: `server.example.com:443`

**Identifiers:**
- Pure numbers: `123`, `456789`
- Hexadecimal numbers: `0xFF`, `0x123ABC`
- UUIDs: `550e8400-e29b-41d4-a716-446655440000`
- Block IDs: `blk_123`, `blk_-456`
- Session IDs: 16+ character alphanumeric strings

**Time and Date:**
- Timestamps: `10:30:22`, `23:59:59`
- ISO datetime: `2023-01-15T10:30:00`
- Unix timestamps: `1673789445`

**File and System:**
- File sizes: `123KB`, `4GB`
- Unix paths: `/var/log/application.log`
- Windows paths: `C:\Users\Admin\logs`
- Filenames with extensions: `app_v2.3.4.tar.gz`

**Web and Email:**
- URLs: `https://api.example.com/v1/users`
- Email addresses: `user@example.com`

**Other:**
- Software versions: `v2.3.4`, `1.0.0-beta`
- Percentages: `95%`, `100%`
- Memory addresses: `0x7fff5fbff8c0`

### Enhanced Features (Drain+ Improvements)

This implementation includes several enhancements inspired by Drain+ research that improve parsing quality while maintaining backward compatibility:

**Enhanced Post-Processing** (`UseEnhancedPostProcessing`):
- Advanced heuristics for variable detection
- Detects complex patterns like mixed alphanumeric strings
- Identifies encoded data (base64, hashes)
- Uses entropy analysis for random string detection
- **Measured improvement**: 16.7% reduction in template count on complex logs

**Statistical Threshold** (`UseStatisticalThreshold`):
- Adaptive threshold calculation based on data distribution
- Better handling of small (<10 unique words) and large (>100) datasets
- Sigmoid smoothing for mid-range values
- Prevents over-splitting and under-generalization

**Parallel Processing** (`ParallelProcessingThreshold`):
- Automatically enables for large log groups
- Optimal worker count calculation
- Minimal overhead for small datasets
- Up to 40% performance improvement on large datasets

**Extended Variable Patterns**:
- Automatic recognition of modern identifiers (enabled by default)
- Email addresses, URLs, MAC addresses, IPv6, file paths
- Session IDs, UUIDs, software versions, memory addresses
- No configuration required - works out of the box

## Performance

The Brain algorithm is designed for high performance with optional enhancements:

### Core Algorithm
- **Time Complexity**: O(n × m × log(m)) where n is the number of logs and m is the average log length
- **Space Complexity**: O(n × m) for storing the bidirectional tree
- **Scalability**: Handles millions of log lines efficiently

### Enhancement Overhead Analysis
Based on comprehensive benchmarks with 1,000-10,000 log samples:

| Feature | Performance Impact | Memory Impact | Quality Improvement |
|---------|-------------------|---------------|-------------------|
| **Extended Patterns** | +5-10% processing time | Minimal | Better modern ID recognition |
| **Enhanced Post-Processing** | +15-25% processing time | +2-5% memory | 16.7% fewer templates on complex logs |
| **Statistical Threshold** | +1-3% processing time | None | Adaptive behavior for mixed datasets |
| **Parallel Processing** | -20-40% processing time | +10-15% memory | Same quality, faster processing |

### Benchmark Results
```
Configuration                Dataset          Time (ms)    Templates    Throughput (logs/sec)
=============================================================================================
Original Brain               1,000 logs         7.3          25           137,455
All Enhancements             1,000 logs        14.3          11            70,089
Original Brain              10,000 logs        69.5          14           143,887
All Enhancements            10,000 logs       140.1          11            71,398
```

### Usage Recommendations

**For Enterprise Systems** (complex logs with modern identifiers):
```go
config := parser.Config{
    UseEnhancedPostProcessing:   true,  // +16.7% template reduction
    UseDynamicThreshold:         true,
    UseStatisticalThreshold:     true,  // Adaptive behavior
    ParallelProcessingThreshold: 1000,  // Faster on large datasets
    DynamicThresholdFactor:      2.0,
}
```

**For Real-Time Processing** (minimize latency):
```go
config := parser.Config{
    // Only use extended patterns (enabled by default)
    UseEnhancedPostProcessing: false,   // Save 15-25% processing time
    UseStatisticalThreshold:   false,   // Minimal overhead
}
```

### Backward Compatibility

All enhancements are **fully backward compatible** with the original Brain algorithm:

- **Default behavior**: Identical to original Brain when no enhanced features are enabled
- **Incremental adoption**: Features can be enabled individually
- **API compatibility**: All existing code continues to work without changes
- **Algorithm preservation**: Core Brain logic remains unchanged

To use the original Brain algorithm exactly as described in the paper:
```go
config := parser.Config{
    Delimiters: `[\s,:=]+`,
    ChildBranchThreshold: 3,
    CommonVariables: map[string]string{
        "ipv4_address": `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`,
        "pure_numbers": `^\d+$`,
        "hex_numbers":  `^0x[a-fA-F0-9]+$`,
        // Original patterns only
    },
    // All enhanced features disabled by default
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## References

- **Primary**: "Brain: Log Parsing with Bidirectional Parallel Tree" (IEEE INFOCOM 2023)
  - Authors: Siyu Yu, Pinjia He, Ningjiang Chen, Yifan Wu
- **Enhancements**: Inspired by Drain+ research and modern log parsing requirements

## Acknowledgments

This implementation is based on the original Brain research paper and includes practical enhancements for modern log parsing challenges. The core algorithm remains faithful to the paper while extending capabilities for real-world usage.
