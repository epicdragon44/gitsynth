package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/go-github/v71/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
)

type PRMergeHandler struct {
	githubapp.ClientCreator
	workdir    string
	gitService GitService
}

// NewPRMergeHandler creates a new PRMergeHandler with the given configuration
func NewPRMergeHandler(clientCreator githubapp.ClientCreator, workdir string) *PRMergeHandler {
	return &PRMergeHandler{
		ClientCreator: clientCreator,
		workdir:       workdir,
		gitService:    NewGitService(),
	}
}

func (h *PRMergeHandler) Handles() []string {
	return []string{"pull_request"}
}

func (h *PRMergeHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	// Parse event payload
	var event github.PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse pull request event payload")
	}

	// Only process when PR is not mergeable
	if event.GetPullRequest().GetMergeable() {
		return nil
	}

	// Get repo, PR number, and installation ID
	repo := event.GetRepo()
	prNum := event.GetPullRequest().GetNumber()
	installationID := githubapp.GetInstallationIDFromEvent(&event)

	client, err := h.NewInstallationClient(installationID)

	// Setup context and logger
	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, prNum)

	// Create temporary directories
	workingDir, err1 := os.MkdirTemp(h.workdir, fmt.Sprintf("work-pr-%d-*", prNum))
	backupDir, err2 := os.MkdirTemp(h.workdir, fmt.Sprintf("backup-pr-%d-*", prNum))
	if err1 != nil || err2 != nil {
		return errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(workingDir)
	defer os.RemoveAll(backupDir)

	logger.Info().Msgf("Processing PR %d in temporary directory %s", prNum, workingDir)

	// Get repository information
	repoOwner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	baseBranch := event.GetPullRequest().GetBase().GetRef()
	headBranch := event.GetPullRequest().GetHead().GetRef()

	// Clone repository and run Gitsynth binary
	if err := h.gitService.Clone(ctx, repoOwner, repoName, workingDir); err != nil {
		return errors.Wrap(err, "failed to clone repository into working directory")
	}
	if err := h.gitService.Clone(ctx, repoOwner, repoName, backupDir); err != nil {
		return errors.Wrap(err, "failed to clone repository into backup directory")
	}
	if err := h.gitService.Configure(ctx, workingDir); err != nil {
		return errors.Wrap(err, "failed to configure git")
	}
	if err := h.gitService.Checkout(ctx, workingDir, headBranch); err != nil {
		return errors.Wrap(err, "failed to checkout HEAD branch")
	}
	if err := h.gitService.Merge(ctx, workingDir, baseBranch); err != nil {
		logger.Debug().Msg("Merge failed as expected, continuing with gitsynth")
	}
	output, err := h.gitService.RunGitsynth(ctx, workingDir)
	if err != nil {
		logger.Error().Str("output", output).Msg("Gitsynth output before failure")
		return errors.Wrap(err, "failed to run gitsynth")
	}
	logger.Info().Str("output", output).Msg("Gitsynth completed successfully")

	// Inspect Gitsynth's modified files and copy changes over via SDK API
	lastCommitMsg, commitErr1 := h.gitService.GetLatestCommitMsg(ctx, workingDir)
	touchedFiles, commitErr2 := h.gitService.InspectLatestCommit(ctx, workingDir)
	if commitErr1 != nil {
		return errors.Wrap(commitErr1, "failed to get last commit msg")
	}
	if commitErr2 != nil {
		return errors.Wrap(commitErr2, "failed to get last commit files")
	}
	logger.Info().Msgf("Processing %d touched files...", len(touchedFiles))
	lastCommit := ""
	for _, file := range touchedFiles {
		fileContents, contentErr1 := h.gitService.ReadFile(ctx, workingDir, file.Path, true)
		decodedBytes, contentErr2 := base64.StdEncoding.DecodeString(fileContents)
		currentFileContents, _, _, err := client.Repositories.GetContents(ctx, repoOwner, repoName, file.Path, &github.RepositoryContentGetOptions{
			Ref: headBranch,
		})

		if file.Status == "D" {
			if err != nil {
				logger.Err(err).Msgf("Failed to get file info from GitHub for %s", file.Path)
				continue
			}
			logger.Info().Msgf("Deleting %s", file.Path)
			res, _, dErr := client.Repositories.DeleteFile(ctx, repoOwner, repoName, file.Path, &github.RepositoryContentFileOptions{
				SHA:     github.Ptr(currentFileContents.GetSHA()),
				Message: &lastCommitMsg,
				Branch:  &headBranch,
			})
			if dErr != nil {
				logger.Err(dErr).Msgf("Failed to delete %s", file.Path)
				continue
			}
			lastCommit = *res.SHA
		} else if file.Status == "M" {
			if err != nil || contentErr1 != nil || contentErr2 != nil {
				logger.Error().Msgf("Failed to get either file info or contents of %s", file.Path)
				continue
			}
			logger.Info().Msgf("Modifying %s", file.Path)
			res, _, mErr := client.Repositories.CreateFile(ctx, repoOwner, repoName, file.Path, &github.RepositoryContentFileOptions{
				SHA:     github.Ptr(currentFileContents.GetSHA()),
				Message: &lastCommitMsg,
				Content: decodedBytes,
				Branch:  &headBranch,
			})
			if mErr != nil {
				logger.Err(mErr).Msgf("Failed to modify %s", file.Path)
				continue
			}
			lastCommit = *res.SHA
		} else if file.Status == "A" {
			if contentErr1 != nil || contentErr2 != nil {
				logger.Error().Msgf("Failed to get contents of %s", file.Path)
				continue
			}
			logger.Info().Msgf("Creating %s", file.Path)
			res, _, aErr := client.Repositories.CreateFile(ctx, repoOwner, repoName, file.Path, &github.RepositoryContentFileOptions{
				Message: &lastCommitMsg,
				Content: decodedBytes,
				Branch:  &headBranch,
			})
			if aErr != nil {
				logger.Err(aErr).Msgf("Failed to create %s", file.Path)
				continue
			}
			lastCommit = *res.SHA
		}
	}

	// This tells GitHub the conflict is resolved for these files
	_, _, updateErr := client.PullRequests.UpdateBranch(ctx, repoOwner, repoName, prNum, &github.PullRequestBranchUpdateOptions{
		ExpectedHeadSHA: &lastCommit,
	})
	if updateErr != nil {
		logger.Err(updateErr).Msg("Failed to mark all as resolved")
		return errors.Wrap(updateErr, "Didn't mark as resolved")
	}

	// Success!
	logger.Info().Msg("Successfully processed merge conflicts!")
	return nil
}
