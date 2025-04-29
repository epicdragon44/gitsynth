package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// GitHubService provides methods for interacting with GitHub API
type GitHubService struct {
	client *github.Client
}

// PullRequestDetails contains information about a pull request
type PullRequestDetails struct {
	BaseOwner  string
	BaseRepo   string
	BaseBranch string
	HeadOwner  string
	HeadRepo   string
	HeadBranch string
	CloneURL   string
}

// NewGitHubService creates a new GitHub service with authentication
func NewGitHubService(token string) *GitHubService {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &GitHubService{
		client: client,
	}
}

// GetPullRequestDetails fetches information about a pull request
func (s *GitHubService) GetPullRequestDetails(ctx context.Context, owner, repo string, prID int) (*PullRequestDetails, error) {
	log.Printf("Fetching details for PR #%d in %s/%s", prID, owner, repo)

	pr, _, err := s.client.PullRequests.Get(ctx, owner, repo, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR details: %w", err)
	}

	if pr.Base == nil || pr.Head == nil {
		return nil, fmt.Errorf("PR has invalid base or head information")
	}

	// Extract repository information
	baseRepo := pr.GetBase().GetRepo()
	headRepo := pr.GetHead().GetRepo()

	if baseRepo == nil || headRepo == nil {
		return nil, fmt.Errorf("PR has invalid repository information")
	}

	details := &PullRequestDetails{
		BaseOwner:  baseRepo.GetOwner().GetLogin(),
		BaseRepo:   baseRepo.GetName(),
		BaseBranch: pr.GetBase().GetRef(),
		HeadOwner:  headRepo.GetOwner().GetLogin(),
		HeadRepo:   headRepo.GetName(),
		HeadBranch: pr.GetHead().GetRef(),
		CloneURL:   baseRepo.GetCloneURL(),
	}

	log.Printf("PR details: Base=%s/%s@%s, Head=%s/%s@%s",
		details.BaseOwner, details.BaseRepo, details.BaseBranch,
		details.HeadOwner, details.HeadRepo, details.HeadBranch)

	return details, nil
}