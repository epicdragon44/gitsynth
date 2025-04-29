package main

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// GitService provides methods for Git operations
type GitService struct {
	docker *DockerService
}

// NewGitService creates a new Git service
func NewGitService(dockerService *DockerService) *GitService {
	return &GitService{
		docker: dockerService,
	}
}

// SetupGitConfig configures Git in the container
func (s *GitService) SetupGitConfig(ctx context.Context, containerID, email, username string) error {
	log.Printf("Setting up Git configuration in container %s", containerID)

	// Configure user email
	emailCmd := []string{"git", "config", "--global", "user.email", email}
	result, err := s.docker.ExecuteCommand(ctx, containerID, emailCmd)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to set git email: %v, stderr: %s", err, result.Stderr)
	}

	// Configure username
	nameCmd := []string{"git", "config", "--global", "user.name", username}
	result, err = s.docker.ExecuteCommand(ctx, containerID, nameCmd)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to set git username: %v, stderr: %s", err, result.Stderr)
	}

	log.Printf("Git configuration completed")
	return nil
}

// CloneRepository clones a Git repository in the container
func (s *GitService) CloneRepository(ctx context.Context, containerID, repoURL, token, directory string) error {
	log.Printf("Cloning repository %s in container %s", repoURL, containerID)

	// Insert token into URL if provided
	cloneURL := repoURL
	if token != "" {
		// Replace https:// with https://x-access-token:TOKEN@
		cloneURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://x-access-token:%s@", token), 1)
	}

	// Create directory if specified
	if directory != "" {
		mkdirCmd := []string{"mkdir", "-p", directory}
		result, err := s.docker.ExecuteCommand(ctx, containerID, mkdirCmd)
		if err != nil || result.ExitCode != 0 {
			return fmt.Errorf("failed to create directory: %v, stderr: %s", err, result.Stderr)
		}
	}

	// Clone the repository
	cloneArgs := []string{"git", "clone", cloneURL}
	if directory != "" {
		cloneArgs = append(cloneArgs, directory)
	}

	// Use a sanitized URL for logging (without token)
	logURL := repoURL
	result, err := s.docker.ExecuteCommand(ctx, containerID, cloneArgs)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to clone repository: %v, stderr: %s", err, result.Stderr)
	}

	log.Printf("Repository cloned successfully: %s", logURL)
	return nil
}

// CheckoutBranch checks out a branch in the repository
func (s *GitService) CheckoutBranch(ctx context.Context, containerID, directory, branch string) error {
	log.Printf("Checking out branch %s in container %s", branch, containerID)

	// Change to repo directory
	cdCmd := []string{"cd", directory, "&&", "git", "checkout", branch}
	cmd := []string{"/bin/sh", "-c", strings.Join(cdCmd, " ")}

	result, err := s.docker.ExecuteCommand(ctx, containerID, cmd)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to checkout branch: %v, stderr: %s", err, result.Stderr)
	}

	log.Printf("Branch %s checked out successfully", branch)
	return nil
}

// MergeBranch merges a branch into the current branch
func (s *GitService) MergeBranch(ctx context.Context, containerID, directory, branch string) error {
	log.Printf("Merging branch %s in container %s", branch, containerID)

	// Change to repo directory and merge the branch
	cdCmd := []string{"cd", directory, "&&", "git", "merge", branch}
	cmd := []string{"/bin/sh", "-c", strings.Join(cdCmd, " ")}

	result, err := s.docker.ExecuteCommand(ctx, containerID, cmd)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to merge branch: %v, stderr: %s", err, result.Stderr)
	}

	log.Printf("Branch %s merged successfully", branch)
	return nil
}

// PushChanges pushes changes to the remote repository
func (s *GitService) PushChanges(ctx context.Context, containerID, directory, token string) error {
	log.Printf("Pushing changes in container %s", containerID)

	// Set up credential helper if token is provided
	if token != "" {
		helperCmd := []string{
			"cd", directory, "&&",
			"git", "config", "--local", "credential.helper",
			"'!f() { echo \"password=$GIT_TOKEN\"; }; f'",
		}
		cmd := []string{"/bin/sh", "-c", strings.Join(helperCmd, " ")}

		result, err := s.docker.ExecuteCommand(ctx, containerID, cmd)
		if err != nil || result.ExitCode != 0 {
			return fmt.Errorf("failed to set credential helper: %v, stderr: %s", err, result.Stderr)
		}
	}

	// Push the changes
	pushCmd := []string{"cd", directory, "&&", "git", "push"}
	if token != "" {
		// Include the token as an environment variable
		pushCmd = []string{"cd", directory, "&&", fmt.Sprintf("GIT_TOKEN=%s", token), "git", "push"}
	}

	cmd := []string{"/bin/sh", "-c", strings.Join(pushCmd, " ")}
	result, err := s.docker.ExecuteCommand(ctx, containerID, cmd)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to push changes: %v, stderr: %s", err, result.Stderr)
	}

	log.Printf("Changes pushed successfully")
	return nil
}

// InstallNpmPackage installs an npm package globally
func (s *GitService) InstallNpmPackage(ctx context.Context, containerID, packageName string) error {
	log.Printf("Installing npm package %s in container %s", packageName, containerID)

	cmd := []string{"npm", "install", "-g", packageName}
	result, err := s.docker.ExecuteCommand(ctx, containerID, cmd)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to install npm package: %v, stderr: %s", err, result.Stderr)
	}

	log.Printf("Package %s installed successfully", packageName)
	return nil
}

// RunGitSynth runs the GitSynth tool in the repository directory
func (s *GitService) RunGitSynth(ctx context.Context, containerID, directory string) error {
	log.Printf("Running GitSynth in container %s", containerID)

	// Change to repo directory and run gitsynth
	cdCmd := []string{"cd", directory, "&&", "gitsynth"}
	cmd := []string{"/bin/sh", "-c", strings.Join(cdCmd, " ")}

	result, err := s.docker.ExecuteCommand(ctx, containerID, cmd)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to run GitSynth: %v, stderr: %s", err, result.Stderr)
	}

	log.Printf("GitSynth executed successfully")
	return nil
}