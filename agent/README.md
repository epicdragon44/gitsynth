# GitSynth

Automatic Merge Conflict Resolution.

## Run Locally For Free

```bash
export ANTHROPIC_API_KEY=your-key # OPTIONAL if ANTHROPIC_API_KEY already set in your .env or environment
npx gitsynth
```

## Run as a Github Action

*Coming Soon!*

## Contributing

### Developing

```bash
make # Build a binary for UNIX
```

You can then move the binary, which should be in `./bin/`, to any location, and execute it to try it out.

### Releasing

```
make build-all
make version-patch|minor|major
make publish
```
