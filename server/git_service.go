package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// GitService interface defines the contract for git operations
type GitService interface {
	Clone(ctx context.Context, repoOwner string, repoName string, dir string) error
	Configure(ctx context.Context, dir string) error
	Checkout(ctx context.Context, dir, branch string) error
	Merge(ctx context.Context, dir, branch string) error
	RunGitsynth(ctx context.Context, dir string) (string, error)
	InspectLatestCommit(ctx context.Context, dir string) ([]CommitFileChange, error)
	ReadFile(ctx context.Context, dir, path string, base64Encode bool) (string, error)
	GetSha(ctx context.Context, dir, path string) (string, error)
	GetLatestCommitMsg(ctx context.Context, dir string) (string, error)
}

// DefaultGitService implements GitService using actual git commands
type DefaultGitService struct {
	// Configuration fields can be added here if needed
}

// NewGitService creates a new instance of DefaultGitService
func NewGitService() GitService {
	return &DefaultGitService{}
}

// Clone clones a git repository
func (s *DefaultGitService) Clone(ctx context.Context, repoOwner string, repoName string, dir string) error {
	url := fmt.Sprintf("https://github.com/%s/%s.git", repoOwner, repoName)
	cmd := exec.CommandContext(ctx, "git", "clone", url, dir)
	return cmd.Run()
}

// Configure sets up git configuration
func (s *DefaultGitService) Configure(ctx context.Context, dir string) error {
	cmds := [][]string{
		{"git", "config", "user.name", "GitSynth Bot"},
		{"git", "config", "user.email", "gitsynth[bot]@users.noreply.github.com"},
	}

	for _, args := range cmds {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// Checkout checks out a specific branch
func (s *DefaultGitService) Checkout(ctx context.Context, dir, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", branch)
	cmd.Dir = dir
	return cmd.Run()
}

// Merge merges a branch into the current branch
func (s *DefaultGitService) Merge(ctx context.Context, dir, branch string) error {
	// Use --no-commit to prevent auto-commit on successful merge
	// Use --no-ff to ensure we always create a merge commit
	cmd := exec.CommandContext(ctx, "git", "merge", "--no-commit", "--no-ff", branch)
	cmd.Dir = dir
	err := cmd.Run()

	// We expect this to error in the conflict case - that's what we want
	// The error means we're in a conflicted state ready for gitsynth
	return err
}

// RunGitsynth executes the gitsynth command and returns its output
func (s *DefaultGitService) RunGitsynth(ctx context.Context, dir string) (string, error) {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		return "", fmt.Errorf("error loading .env file: %w", err)
	}

	// Get API key from .env
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY not found in .env file")
	}

	// Build command with API key export
	cmdStr := fmt.Sprintf("export ANTHROPIC_API_KEY=%s && yes | npx gitsynth --debug", apiKey)
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = dir

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("gitsynth command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// CommitFileChange represents a file change in a commit
type CommitFileChange struct {
	Path   string
	Status string // "M" for modified, "D" for deleted, and "A" for added
}

// InspectLatestCommit returns the files changed in the latest commit
func (s *DefaultGitService) InspectLatestCommit(ctx context.Context, dir string) ([]CommitFileChange, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", "git diff --name-status origin/HEAD | cat")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var changes []CommitFileChange
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		path := parts[1]

		normalizedStatus := "M"
		if strings.Contains(status, "M") {
			normalizedStatus = "M"
		} else if strings.Contains(status, "D") {
			normalizedStatus = "D"
		} else if strings.Contains(status, "A") {
			normalizedStatus = "A"
		} else {
			// TODO: handle stuff like R
		}

		changes = append(changes, CommitFileChange{
			Path:   path,
			Status: normalizedStatus,
		})
	}

	return changes, nil
}

// ReadFile reads a file from the repository and returns its contents.
// If base64Encode is true, the content is returned as a base64 encoded string.
func (s *DefaultGitService) ReadFile(ctx context.Context, dir, path string, base64Encode bool) (string, error) {
	fullPath := filepath.Join(dir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	if base64Encode {
		return base64.StdEncoding.EncodeToString(content), nil
	}
	return string(content), nil
}

// GetSha returns the SHA-1 hash of a file in the repository
func (s *DefaultGitService) GetSha(ctx context.Context, dir, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "hash-object", path)
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get SHA for file %s: %w", path, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetLatestCommitMsg returns the message of the most recent commit
func (s *DefaultGitService) GetLatestCommitMsg(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=%B")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit message: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
