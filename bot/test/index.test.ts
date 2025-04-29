// You can import your modules
// import index from '../src/index'

import nock from "nock";
// Requiring our app implementation
import myProbotApp from "../src/index.js";
import { Probot, ProbotOctokit } from "probot";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { describe, beforeEach, afterEach, test, expect, vi } from "vitest";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Read test fixtures
const privateKey = fs.readFileSync(
  path.join(__dirname, "fixtures/mock-cert.pem"),
  "utf-8",
);

const issuesOpenedPayload = JSON.parse(
  fs.readFileSync(path.join(__dirname, "fixtures/issues.opened.json"), "utf-8"),
);

const prOpenedPayload = JSON.parse(
  fs.readFileSync(path.join(__dirname, "fixtures/pull_request.opened.json"), "utf-8"),
);

const prWithConflictsPayload = JSON.parse(
  fs.readFileSync(path.join(__dirname, "fixtures/pull_request.with_conflicts.json"), "utf-8"),
);

describe("GitSynth Bot Tests", () => {
  let probot: any;
  
  // Mock console.log to avoid test output clutter
  vi.spyOn(console, 'log').mockImplementation(() => {});

  beforeEach(() => {
    nock.disableNetConnect();
    // Allow requests to our API endpoint
    nock.enableNetConnect('api.gitsynth.io');
    
    probot = new Probot({
      appId: 123,
      privateKey,
      // disable request throttling and retries for testing
      Octokit: ProbotOctokit.defaults({
        retry: { enabled: false },
        throttle: { enabled: false },
      }),
    });
    // Load our app into probot
    probot.load(myProbotApp);
  });

  test("checks for merge conflicts when PR is opened but has null mergeable status", async () => {
    // GitHub API mocks
    const mock = nock("https://api.github.com")
      // Return an installation token when requested
      .post("/app/installations/2/access_tokens")
      .reply(200, {
        token: "test-token",
        permissions: {
          pull_requests: "write",
          contents: "read",
        },
      })
      
      // Mock the subsequent PR fetch (to check mergeable status) - still unmergeable
      .get("/repos/octocat/Hello-World/pulls/123")
      .reply(200, {
        number: 123,
        mergeable: false,
        mergeable_state: "dirty",
      })
      
      // Mock the comment to notify about conflict resolution attempt
      .post("/repos/octocat/Hello-World/issues/123/comments", (body: any) => {
        expect(body.body).toContain("GitSynth detected merge conflicts");
        return true;
      })
      .reply(200)
      
      // Mock the final comment about conflict resolution result
      .post("/repos/octocat/Hello-World/issues/123/comments", (body: any) => {
        expect(body.body).toContain("GitSynth has attempted to resolve");
        return true;
      })
      .reply(200);
      
    // Mock the GitSynth API call
    const apiMock = nock("https://api.gitsynth.io")
      .post("/api/run", (body: any) => {
        expect(body.author).toBe("octocat");
        expect(body.repo).toBe("Hello-World");
        expect(body.pr_id).toBe(123);
        expect(body.github_token).toBe("test-token");
        return true;
      })
      .reply(200, { message: "Success!" });

    // Receive a webhook event for a PR opening
    await probot.receive({ name: "pull_request", payload: prOpenedPayload });

    // Verify all expected HTTP requests were made
    expect(mock.pendingMocks()).toStrictEqual([]);
    expect(apiMock.pendingMocks()).toStrictEqual([]);
  });

  test("detects merge conflicts in a PR and attempts resolution", async () => {
    // GitHub API mocks
    const mock = nock("https://api.github.com")
      // Return an installation token when requested
      .post("/app/installations/2/access_tokens")
      .reply(200, {
        token: "test-token",
        permissions: {
          pull_requests: "write",
          contents: "read",
        },
      })
      
      // Mock the comment to notify about conflict resolution attempt
      .post("/repos/octocat/Hello-World/issues/123/comments", (body: any) => {
        expect(body.body).toContain("GitSynth detected merge conflicts");
        return true;
      })
      .reply(200)
      
      // Mock the final comment about conflict resolution result
      .post("/repos/octocat/Hello-World/issues/123/comments", (body: any) => {
        expect(body.body).toContain("GitSynth has attempted to resolve");
        return true;
      })
      .reply(200);
      
    // Mock the GitSynth API call
    const apiMock = nock("https://api.gitsynth.io")
      .post("/api/run", (body: any) => {
        expect(body.author).toBe("octocat");
        expect(body.repo).toBe("Hello-World");
        expect(body.pr_id).toBe(123);
        expect(body.github_token).toBe("test-token");
        return true;
      })
      .reply(200, { message: "Success!" });

    // Receive a webhook event for a PR with conflicts
    await probot.receive({ name: "pull_request", payload: prWithConflictsPayload });

    // Verify all expected HTTP requests were made
    expect(mock.pendingMocks()).toStrictEqual([]);
    expect(apiMock.pendingMocks()).toStrictEqual([]);
  });

  test("handles API errors gracefully", async () => {
    // GitHub API mocks
    const mock = nock("https://api.github.com")
      // Return an installation token when requested
      .post("/app/installations/2/access_tokens")
      .reply(200, {
        token: "test-token",
        permissions: {
          pull_requests: "write",
          contents: "read",
        },
      })
      
      // Mock the comment to notify about conflict resolution attempt
      .post("/repos/octocat/Hello-World/issues/123/comments", (body: any) => {
        expect(body.body).toContain("GitSynth detected merge conflicts");
        return true;
      })
      .reply(200)
      
      // Mock the error comment
      .post("/repos/octocat/Hello-World/issues/123/comments", (body: any) => {
        expect(body.body).toContain("GitSynth encountered an error");
        return true;
      })
      .reply(200);
      
    // Mock the GitSynth API call - with an error response
    const apiMock = nock("https://api.gitsynth.io")
      .post("/api/run")
      .reply(500, { message: "Internal server error" });

    // Receive a webhook event for a PR with conflicts
    await probot.receive({ name: "pull_request", payload: prWithConflictsPayload });

    // Verify all expected HTTP requests were made
    expect(mock.pendingMocks()).toStrictEqual([]);
    expect(apiMock.pendingMocks()).toStrictEqual([]);
  });

  afterEach(() => {
    nock.cleanAll();
    nock.enableNetConnect();
  });
});
