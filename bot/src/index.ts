import { Probot } from "probot";
import fetch from "node-fetch";

interface RunRequest {
  author: string; // Github Repo author or org
  repo: string; // Github Repo name
  pr_id: number; // Github PR ID (numerical)
  github_token: string; // Github token for authentication
}

export default (app: Probot) => {
  // Listen for PR opened or synchronized (new commit pushed) events
  app.on(
    ["pull_request.opened", "pull_request.synchronize"],
    async (context) => {
      const pr = context.payload.pull_request;
      const repo = context.repo();

      app.log.info(`Processing PR #${pr.number} in ${repo.owner}/${repo.repo}`);

      try {
        // Check if the PR has merge conflicts
        const hasMergeConflicts =
          !pr.mergeable && pr.mergeable_state === "dirty";

        // If GitHub hasn't computed the mergeable status yet (it's null), we need to fetch it again
        if (pr.mergeable === null) {
          app.log.info(
            `Mergeable status not yet computed for PR #${pr.number}, waiting...`,
          );

          // Wait a bit for GitHub to compute the mergeable status
          await new Promise((resolve) => setTimeout(resolve, 5000));

          // Fetch the PR again to get the updated mergeable status
          const { data: updatedPr } = await context.octokit.pulls.get({
            owner: repo.owner,
            repo: repo.repo,
            pull_number: pr.number,
          });

          // Check again with the updated information
          if (
            updatedPr.mergeable === false &&
            updatedPr.mergeable_state === "dirty"
          ) {
            app.log.info(`Detected merge conflicts in PR #${pr.number}`);
            await processWithMergeConflicts(context, updatedPr.number);
          } else {
            app.log.info(`No merge conflicts detected in PR #${pr.number}`);
          }
        }
        // If we already know there are merge conflicts
        else if (hasMergeConflicts) {
          app.log.info(`Detected merge conflicts in PR #${pr.number}`);
          await processWithMergeConflicts(context, pr.number);
        } else {
          app.log.info(`No merge conflicts detected in PR #${pr.number}`);
        }
      } catch (error) {
        app.log.error(`Error processing PR #${pr.number}: ${error}`);
      }
    },
  );

  // Function to process PRs with merge conflicts
  async function processWithMergeConflicts(context: any, prNumber: number) {
    const repo = context.repo();

    try {
      // Post a comment to inform users we're resolving conflicts
      await context.octokit.issues.createComment({
        owner: repo.owner,
        repo: repo.repo,
        issue_number: prNumber,
        body: "📢 GitSynth detected merge conflicts. Attempting to resolve them automatically...",
      });

      // Get installation token for this repository
      const installationId = context.payload.installation.id;
      const installationAccessToken = await context.octokit.auth({
        type: "installation",
        installationId,
      });

      // Prepare request payload
      const requestPayload: RunRequest = {
        author: repo.owner,
        repo: repo.repo,
        pr_id: prNumber,
        github_token: installationAccessToken.token,
      };

      // Send request to GitSynth API
      app.log.info(`Sending request to GitSynth API for PR #${prNumber}`);
      const response = await fetch("https://api.gitsynth.io/api/run", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(requestPayload),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(
          `API request failed with status ${response.status}: ${errorText}`,
        );
      }

      const result = await response.json();
      app.log.info(
        `GitSynth API response for PR #${prNumber}: ${JSON.stringify(result)}`,
      );

      // Post a follow-up comment with the result
      await context.octokit.issues.createComment({
        owner: repo.owner,
        repo: repo.repo,
        issue_number: prNumber,
        body: "✅ GitSynth has resolved the merge conflicts. Please check the PR!",
      });
    } catch (error) {
      app.log.error(`Error resolving conflicts for PR #${prNumber}: ${error}`);

      // Post a comment about the failure
      await context.octokit.issues.createComment({
        owner: repo.owner,
        repo: repo.repo,
        issue_number: prNumber,
        body: `❌ GitSynth encountered an error while trying to resolve merge conflicts: \n\n\`\`\`\n${error}\n\`\`\``,
      });
    }
  }
};
