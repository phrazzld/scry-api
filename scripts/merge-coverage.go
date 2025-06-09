package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s file1.out file2.out [...]\n", os.Args[0])
		os.Exit(1)
	}

	// Read mode from first file
	firstFile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(firstFile)
	if !scanner.Scan() {
		fmt.Fprintf(os.Stderr, "Error reading mode line from %s\n", os.Args[1])
		os.Exit(1)
	}
	modeLine := scanner.Text()
	defer func() { _ = firstFile.Close() }()

	// Print mode line once
	fmt.Println(modeLine)

	// Process all files
	for _, filename := range os.Args[1:] {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", filename, err)
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "mode:") && line != "" {
				fmt.Println(line)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filename, err)
		}
		_ = file.Close()
	}
}
