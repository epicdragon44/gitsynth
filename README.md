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

- [x] Local Agent: Headless, automatic resolution of merge conflicts
- [ ] Github Action: Github App with a serverless `/run` endpoint that clones the repository, gets it into the merge conflict state, runs the local agent, and pushes the changes back up.
- [ ] Site: Minimalist redesign to link to the Github Action and local agent install instructions.

### Bonus Features

- [ ] Support for .cursorrules and .rules
- [ ] Caching and other performance optimizations
- [ ] Deep thinking mode: agent should track all symbols involved across multiple files and histories before making decisions, and then suggest multiple possible candidates for how to resolve the conflict.
