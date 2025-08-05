// Package main demonstrates the Brain log parser with example data from the research paper.
package main

import (
	"fmt"

	"github.com/n0madic/go-brain/parser"
)

func main() {
	// Example logs based on Fig. 2 and Fig. 3 from the paper
	logLines := []string{
		"proxy.cse.cuhk.edu.hk:5070 open through proxy proxy.cse.cuhk.edu.hk:5070 HTTPS",       // Log0
		"proxy.cse.cuhk.edu.hk:5070 close, 0 bytes sent, 0 bytes received, lifetime 00:01",     // Log1
		"proxy.cse.cuhk.edu.hk:5070 open through proxy p3p.sogou.com:80 HTTPS",                 // Log2
		"proxy.cse.cuhk.edu.hk:5070 open through proxy 182.254.114.110:80 SOCKS5",              // Log3
		"182.254.114.110:80 open through proxy 182.254.114.110:80 HTTPS",                       // Log4
		"proxy.cse.cuhk.edu.hk:5070 close, 403 bytes sent, 426 bytes received, lifetime 00:02", // Log5
		"get.sogou.com:80 close, 651 bytes sent, 546 bytes received, lifetime 00:03",           // Log6
		"proxy.cse.cuhk.edu.hk:5070 close, 108 bytes sent, 411 bytes received, lifetime 00:03", // Log7
		"183.62.156.108:27 open through proxy socks.cse.cuhk.edu.hk:5070 SOCKS5",               // Log8
		"proxy.cse.cuhk.edu.hk:5070 open through proxy proxy.cse.cuhk.edu.hk:5070 SOCKS5",      // Log9
	}

	// Parser configuration
	config := parser.Config{
		Delimiters:           `[\s,]`, // Space, comma (remove colon for correct time and host parsing)
		ChildBranchThreshold: 2,       // In the example from Fig. 4 threshold > 2, so 2 nodes is constant
		Weight:               0.6,     // Weight parameter for frequency threshold (60% of maximum frequency)
		CommonVariables: map[string]string{
			// Common variable patterns for replacement with wildcards
			"ip_port":  `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+$`,   // IP:port
			"hostname": `^[a-zA-Z0-9.-]+\.(edu|com|org|net|hk):\d+$`, // hostname:port
			"time":     `^\d{2}:\d{2}$`,                              // time MM:SS
			"numbers":  `^\d+$`,                                      // pure numbers
		},
	}

	// Create and run parser
	brainParser := parser.New(config)
	results := brainParser.Parse(logLines)

	// Output results
	fmt.Println("\n--- Parsing Results ---")
	for _, result := range results {
		fmt.Printf("Template: %s\n", result.Template)
		fmt.Printf("  Found: %d times\n", result.Count)
		// fmt.Printf("  Log IDs: %v\n", result.LogIDs)
		fmt.Println("---------------------------")
	}

	fmt.Println("\nExpected templates from the paper:")
	fmt.Println("<*> open through proxy <*> HTTPS")
	fmt.Println("<*> open through proxy <*> SOCKS5")
	fmt.Println("<*> close <*> bytes sent <*> bytes received lifetime <*>")
}
