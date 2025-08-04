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
```

#### CLI Options

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
}
```

### Default Common Variables

The parser automatically identifies common variable patterns:
- IPv4 addresses: `192.168.1.1`
- IPv4 with ports: `192.168.1.1:8080`
- Hostnames with ports: `server.example.com:443`
- Pure numbers: `123`, `456789`
- Hexadecimal numbers: `0xFF`, `0x123ABC`
- Timestamps: `10:30:22`, `23:59:59`
- Block IDs: `blk_123`, `blk_-456`
- File sizes: `123KB`, `4GB`
- Percentages: `95%`, `100%`
- UUIDs: `550e8400-e29b-41d4-a716-446655440000`

## Performance

The Brain algorithm is designed for high performance:
- **Time Complexity**: O(n × m × log(m)) where n is the number of logs and m is the average log length
- **Space Complexity**: O(n × m) for storing the bidirectional tree
- **Scalability**: Handles millions of log lines efficiently

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## References

- Original Paper: "Brain: Log Parsing with Bidirectional Parallel Tree" (IEEE INFOCOM 2023)
- Authors: Siyu Yu, Pinjia He, Ningjiang Chen, Yifan Wu

## Acknowledgments

This implementation is based on the research paper and aims to provide a production-ready Go version of the Brain algorithm for the community.
