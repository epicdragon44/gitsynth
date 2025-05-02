package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"github.com/google/go-github/v71/github"
	"github.com/joho/godotenv"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type PRMergeHandler struct {
	githubapp.ClientCreator
	workdir string
}

func (h *PRMergeHandler) Handles() []string {
	return []string{"pull_request"}
}

func (h *PRMergeHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse pull request event payload")
	}

	// Only process when PR is not mergeable
	if event.GetPullRequest().GetMergeable() != false {
		return nil
	}

	repo := event.GetRepo()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	prNum := event.GetPullRequest().GetNumber()

	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, prNum)

	client, err := h.NewInstallationClient(installationID)
	if err != nil {
		return err
	}

	// Create temporary directory for this operation
	tmpDir, err := ioutil.TempDir(h.workdir, fmt.Sprintf("pr-%d-*", prNum))
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(tmpDir)

	logger.Info().Msgf("Processing PR %d in temporary directory %s", prNum, tmpDir)

	// Get repository information
	repoOwner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	baseBranch := event.GetPullRequest().GetBase().GetRef()
	headBranch := event.GetPullRequest().GetHead().GetRef()

	// Get installation token for git operations
	token, err := getInstallationToken(ctx, client)
	if err != nil {
		return errors.Wrap(err, "failed to get installation token")
	}

	// Clone repository
	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, repoOwner, repoName)
	if err := gitClone(ctx, cloneURL, tmpDir); err != nil {
		return errors.Wrap(err, "failed to clone repository")
	}

	// Configure git
	if err := gitConfig(ctx, tmpDir); err != nil {
		return errors.Wrap(err, "failed to configure git")
	}

	// Checkout base branch
	if err := gitCheckout(ctx, tmpDir, baseBranch); err != nil {
		return errors.Wrap(err, "failed to checkout base branch")
	}

	// Try to merge head branch
	if err := gitMerge(ctx, tmpDir, headBranch); err != nil {
		logger.Debug().Msg("Merge failed as expected, continuing with gitsynth")
	}

	// Run gitsynth
	if err := runGitsynth(ctx, tmpDir); err != nil {
		return errors.Wrap(err, "failed to run gitsynth")
	}

	// Push changes
	if err := gitPush(ctx, tmpDir); err != nil {
		return errors.Wrap(err, "failed to push changes")
	}

	logger.Info().Msg("Successfully processed merge conflicts")
	return nil
}

func getInstallationToken(ctx context.Context, client *github.Client) (string, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("Loading environment variables and requesting new installation token")

	// Load .env file
	if err := godotenv.Load(); err != nil {
		return "", errors.Wrap(err, "failed to load .env file")
	}

	// Get installation ID from environment
	installIDStr := os.Getenv("GITHUB_CLIENT_ID")
	if installIDStr == "" {
		return "", errors.New("GITHUB_CLIENT_ID not found in environment")
	}

	// Convert installation ID to int64
	installID, err := strconv.ParseInt(installIDStr, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "invalid GITHUB_CLIENT_ID format")
	}

	// Request a new access token for the installation
	token, _, err := client.Apps.CreateInstallationToken(ctx, installID, &github.InstallationTokenOptions{
		// Request minimal permissions needed for the operation
		Permissions: &github.InstallationPermissions{
			Contents:     github.String("write"),
			PullRequests: github.String("write"),
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to create installation token")
	}

	if token.Token == nil || *token.Token == "" {
		return "", errors.New("received empty installation token")
	}

	logger.Debug().
		Str("expires_at", token.GetExpiresAt().String()).
		Msg("Successfully obtained installation token")

	return *token.Token, nil
}

func gitClone(ctx context.Context, url, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", url, dir)
	return cmd.Run()
}

func gitConfig(ctx context.Context, dir string) error {
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

func gitCheckout(ctx context.Context, dir, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", branch)
	cmd.Dir = dir
	return cmd.Run()
}

func gitMerge(ctx context.Context, dir, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "merge", branch)
	cmd.Dir = dir
	return cmd.Run()
}

func runGitsynth(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "gitsynth")
	cmd.Dir = dir
	return cmd.Run()
}

func gitPush(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "push")
	cmd.Dir = dir
	return cmd.Run()
}
