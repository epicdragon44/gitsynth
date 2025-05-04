# GitSynth

Resolve all your Git merge conflicts in just one line.

```bash
npx gitsynth
```

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
