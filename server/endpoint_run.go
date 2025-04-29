package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// RunRequest represents the request payload for the run endpoint
type RunRequest struct {
	Author     string `json:"author"`      // Github Repo author or org
	Repo       string `json:"repo"`        // Github Repo name
	PRID       int    `json:"pr_id"`       // Github PR ID (numerical)
	GithubToken string `json:"github_token"` // Github token for authentication
}

// RunHandler handles POST requests to /api/run
func RunHandler(w http.ResponseWriter, r *http.Request) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()

	// Set response content type
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Method %s not allowed", r.Method)})
		return
	}

	// Parse the request body
	var requestBody RunRequest
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "Invalid request payload"})
		return
	}

	// Validate input parameters
	if requestBody.Author == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "Author/org cannot be empty"})
		return
	}

	if requestBody.Repo == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "Repository name cannot be empty"})
		return
	}

	if requestBody.PRID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "PR ID must be a positive number"})
		return
	}

	if requestBody.GithubToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "GitHub token cannot be empty"})
		return
	}

	// Log the start of processing (without exposing the token)
	log.Printf("Processing run request: Author=%s, Repo=%s, PR ID=%d", 
		requestBody.Author, requestBody.Repo, requestBody.PRID)

	// Initialize GitHub service
	githubService := NewGitHubService(requestBody.GithubToken)

	// Get PR details
	log.Printf("Fetching PR details from GitHub...")
	prDetails, err := githubService.GetPullRequestDetails(ctx, requestBody.Author, requestBody.Repo, requestBody.PRID)
	if err != nil {
		log.Printf("Error fetching PR details: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to get PR details: %v", err)})
		return
	}

	// Initialize Docker service
	dockerService, err := NewDockerService()
	if err != nil {
		log.Printf("Error initializing Docker service: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to initialize Docker service: %v", err)})
		return
	}

	// Initialize Git service
	gitService := NewGitService(dockerService)

	// Pull the Node.js Docker image with npm
	nodeImage := "node:18-alpine"
	if err := dockerService.PullImage(ctx, nodeImage); err != nil {
		log.Printf("Error pulling Docker image: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to pull Docker image: %v", err)})
		return
	}

	// Create container config with environment variables
	containerConfig := ContainerConfig{
		ImageName: nodeImage,
		Env: []string{
			"GIT_TERMINAL_PROMPT=0", // Disable git terminal prompts
			fmt.Sprintf("GITHUB_TOKEN=%s", requestBody.GithubToken),
		},
	}

	// Create and start container
	log.Printf("Creating Docker container...")
	containerID, err := dockerService.CreateContainer(ctx, containerConfig)
	if err != nil {
		log.Printf("Error creating container: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to create container: %v", err)})
		return
	}

	// Ensure container cleanup
	defer func() {
		log.Printf("Cleaning up container...")
		destroyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := dockerService.DestroyContainer(destroyCtx, containerID); err != nil {
			log.Printf("Warning: failed to clean up container: %v", err)
		}
	}()

	// Start the container
	if err := dockerService.StartContainer(ctx, containerID); err != nil {
		log.Printf("Error starting container: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to start container: %v", err)})
		return
	}

	// Setup Git configuration
	if err := gitService.SetupGitConfig(ctx, containerID, "gitsynth@example.com", "GitSynth Bot"); err != nil {
		log.Printf("Error setting up Git config: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to set up Git: %v", err)})
		return
	}

	// Clone the repository
	repoDir := "/repo"
	if err := gitService.CloneRepository(ctx, containerID, prDetails.CloneURL, requestBody.GithubToken, repoDir); err != nil {
		log.Printf("Error cloning repository: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to clone repository: %v", err)})
		return
	}

	// Checkout base branch
	if err := gitService.CheckoutBranch(ctx, containerID, repoDir, prDetails.BaseBranch); err != nil {
		log.Printf("Error checking out base branch: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to checkout base branch: %v", err)})
		return
	}

	// Merge the PR branch into the base branch
	if err := gitService.MergeBranch(ctx, containerID, repoDir, prDetails.HeadBranch); err != nil {
		log.Printf("Error merging branches: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to merge branches: %v", err)})
		return
	}

	// Install GitSynth npm package
	if err := gitService.InstallNpmPackage(ctx, containerID, "gitsynth"); err != nil {
		log.Printf("Error installing GitSynth: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to install GitSynth: %v", err)})
		return
	}

	// Run GitSynth
	if err := gitService.RunGitSynth(ctx, containerID, repoDir); err != nil {
		log.Printf("Error running GitSynth: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to run GitSynth: %v", err)})
		return
	}

	// Push changes back to GitHub
	if err := gitService.PushChanges(ctx, containerID, repoDir, requestBody.GithubToken); err != nil {
		log.Printf("Error pushing changes: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Failed to push changes: %v", err)})
		return
	}

	log.Printf("Workflow completed successfully for PR #%d in %s/%s", 
		requestBody.PRID, requestBody.Author, requestBody.Repo)

	// Return success response
	response := Response{
		Message: "Success!",
		Data: map[string]interface{}{
			"author": requestBody.Author,
			"repo":   requestBody.Repo,
			"pr_id":  requestBody.PRID,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}