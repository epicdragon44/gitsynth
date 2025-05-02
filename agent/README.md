# local agent

## development

```
make
```

You can then move the binary, which should be in `./bin/`, to any location, and execute it.

## release

```
make build-all
make version-patch|minor|major
make publish
```

## env

When running the binary, make sure ANTHROPIC_API_KEY is set in your environment or `.env` file.

## usage

```
npm install -g gitsynth
gitsynth
```
