package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// The size threshold after which to use memory mapping instead of regular file reading
const memoryMapThreshold = 10 * 1024 * 1024 // 10MB

// Maximum number of files to process in parallel
var maxParallelFiles = runtime.GOMAXPROCS(0) * 2

// Common binary file signatures
var binaryFileSignatures = [][]byte{
	{0x7F, 0x45, 0x4C, 0x46}, // ELF
	{0x4D, 0x5A},             // PE/DOS
	{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O
	{0x50, 0x4B, 0x03, 0x04}, // ZIP
	{0x1F, 0x8B},             // gzip
}

// GrepMatch represents a single match result from a grep operation
type GrepMatch struct {
	Path    string // File path where the match was found
	Line    int    // Line number of the match
	Content string // The matching line content
}

// grepResult is used to collect results from parallel workers
type grepResult struct {
	matches []GrepMatch
	err     error
}

// grep performs a regex search across files in the project
// pattern: regex pattern to search for
// includePattern: glob pattern to filter which files to search in
// caseSensitive: whether the search should be case-sensitive
func grep(pattern string, includePattern string, caseSensitive bool) ([]GrepMatch, error) {
	// Pre-compile the regex pattern
	flags := regexp.Compile
	if !caseSensitive {
		flags = regexp.CompilePOSIX
	}
	
	re, err := flags(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Find all files matching the include pattern
	matchingFiles, err := findMatchingFiles(includePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find matching files: %w", err)
	}

	// Initialize result channel and wait group
	results := make(chan grepResult, len(matchingFiles))
	var wg sync.WaitGroup
	
	// Create a buffered channel to limit parallel processing
	semaphore := make(chan struct{}, maxParallelFiles)
	
	// Initialize an atomic counter for progress tracking
	var filesProcessed uint64
	totalFiles := uint64(len(matchingFiles))

	// Process files in parallel
	for _, filePath := range matchingFiles {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Search the file
			matches, err := searchFile(path, re, caseSensitive)
			results <- grepResult{matches: matches, err: err}
			
			// Update progress
			processed := atomic.AddUint64(&filesProcessed, 1)
			if processed%100 == 0 || processed == totalFiles {
				fmt.Fprintf(os.Stderr, "\rProcessed %d/%d files...", processed, totalFiles)
			}
		}(filePath)
	}

	// Start a goroutine to close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
		fmt.Fprintln(os.Stderr) // New line after progress
	}()

	// Collect results
	var allMatches []GrepMatch
	for result := range results {
		if result.err != nil {
			// Log error but continue processing
			fmt.Fprintf(os.Stderr, "Error processing file: %v\n", result.err)
			continue
		}
		allMatches = append(allMatches, result.matches...)
	}

	return allMatches, nil
}

// searchFile searches a single file for matches
func searchFile(filePath string, re *regexp.Regexp, caseSensitive bool) ([]GrepMatch, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Get file info for size
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Skip directories
	if info.IsDir() {
		return nil, nil
	}

	// Check if it's likely a binary file
	if isBinaryFile(file) {
		return nil, nil
	}

	// Reset file pointer after binary check
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	var matches []GrepMatch
	lineNum := 0

	// Use a buffer pool for line reading
	bufPool := sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 64*1024) // 64KB initial capacity
			return &buf
		},
	}
	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(*buf, 1024*1024) // 1MB max line length

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		var searchLine string
		if !caseSensitive {
			searchLine = strings.ToLower(line)
		} else {
			searchLine = line
		}

		if re.MatchString(searchLine) {
			matches = append(matches, GrepMatch{
				Path:    filePath,
				Line:    lineNum,
				Content: line,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	return matches, nil
}

// isBinaryFile checks if a file is likely binary by looking at its first few bytes
func isBinaryFile(file *os.File) bool {
	// Read first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return true // Assume binary on error
	}
	buf = buf[:n]

	// Check for common binary signatures
	for _, sig := range binaryFileSignatures {
		if bytes.HasPrefix(buf, sig) {
			return true
		}
	}

	// Count zero bytes
	zeros := 0
	for _, b := range buf {
		if b == 0 {
			zeros++
		}
	}

	// If more than 10% of the bytes are zero, likely binary
	return zeros > len(buf)/10
}

// findMatchingFiles returns a list of files that match the given glob pattern
func findMatchingFiles(pattern string) ([]string, error) {
	// Read .gitignore if it exists
	ignorePatterns := make(map[string]bool)
	if ignoreFile, err := os.Open(".gitignore"); err == nil {
		defer ignoreFile.Close()
		scanner := bufio.NewScanner(ignoreFile)
		for scanner.Scan() {
			pattern := strings.TrimSpace(scanner.Text())
			if pattern != "" && !strings.HasPrefix(pattern, "#") {
				ignorePatterns[pattern] = true
			}
		}
	}

	var matches []string
	var mu sync.Mutex // Protect matches slice

	// Use multiple goroutines for walking directories
	var wg sync.WaitGroup
	errChan := make(chan error, 1) // Buffer of 1 to prevent goroutine leak

	// Walk the directory tree
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files and files matching .gitignore patterns
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check .gitignore patterns
		for ignorePattern := range ignorePatterns {
			if matched, _ := filepath.Match(ignorePattern, info.Name()); matched {
				return nil
			}
		}

		// Check if file matches the pattern
		if match, err := filepath.Match(pattern, info.Name()); err != nil {
			return err
		} else if match {
			mu.Lock()
			matches = append(matches, path)
			mu.Unlock()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Wait for all workers and check for errors
	wg.Wait()
	select {
	case err := <-errChan:
		return nil, err
	default:
		return matches, nil
	}
}