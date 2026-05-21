package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Extracts and normalizes path data for comparison
func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: compare_paths <our-svg> <visio-svg>")
		os.Exit(1)
	}

	ourSVG, _ := os.ReadFile(os.Args[1])
	visioSVG, _ := os.ReadFile(os.Args[2])

	fmt.Println("=== Our SVG ===")
	analyzeSVG(string(ourSVG))

	fmt.Println("\n=== Visio SVG ===")
	analyzeSVG(string(visioSVG))
}

func analyzeSVG(svg string) {
	// Extract path d attribute
	pathRe := regexp.MustCompile(`d="([^"]+)"`)
	matches := pathRe.FindStringSubmatch(svg)
	if len(matches) < 2 {
		fmt.Println("No path found")
		return
	}

	pathData := matches[1]
	fmt.Printf("Path data: %s\n", pathData)

	// Parse commands
	cmds := parsePathCommands(pathData)
	fmt.Printf("Commands: %d\n", len(cmds))
	for _, cmd := range cmds {
		fmt.Printf("  %s\n", cmd)
	}
}

func parsePathCommands(pathData string) []string {
	var cmds []string
	re := regexp.MustCompile(`([MLCQAZmlcqaz])([^MLCQAZmlcqaz]*)`)
	matches := re.FindAllStringSubmatch(pathData, -1)
	for _, m := range matches {
		cmd := m[1]
		args := strings.TrimSpace(m[2])
		// Normalize numbers
		args = normalizeNumbers(args)
		cmds = append(cmds, fmt.Sprintf("%s %s", cmd, args))
	}
	return cmds
}

func normalizeNumbers(s string) string {
	parts := strings.Fields(s)
	var normalized []string
	for _, p := range parts {
		f, err := strconv.ParseFloat(p, 64)
		if err == nil {
			normalized = append(normalized, fmt.Sprintf("%.2f", f))
		} else {
			normalized = append(normalized, p)
		}
	}
	return strings.Join(normalized, " ")
}
