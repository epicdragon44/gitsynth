# local agent

## development

```
make
```

You can then execute the binary, which should be in `./bin/`.

## building

```
make build-all
```

## publishing

```
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
