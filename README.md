# GitSynth

Automatic Merge Conflict Resolution.

## Run Locally For Free

```bash
npm install -g gitsynth
gitsynth
```

## Run as a Github Action

*Coming Soon!*

## Roadmap

- [x] Local Agent: Tools: Read and write files, grep for merge conflicts, execute git bash commands.
- [x] Local Agent: Rate limit handling.
- [ ] Local Agent: Tool: Read from git history to inform decisions.
- [ ] Local Agent: Interactive and Headless Mode. The former with pretty outputs and explicit user approval; the latter with no user interaction.

- [ ] Github Action: Configure `bot` to run on pull requests with merge conflicts detected and hit `server` endpoint with conflict details.
- [ ] Github Action: Configure `server` to receive conflict details and respond with resolved versions of each contested file.
- [ ] Github Action: Configure `bot` to make proposed commit to pull request with resolved version.

- [ ] Site: Minimalist redesign to link to the Github Action and local agent install instructions.

- [ ] Configurable custom prompts and rules.
