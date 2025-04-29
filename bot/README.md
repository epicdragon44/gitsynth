# GitHub Bot

This is the GitHub bot component of GitSynth.

## Development

```sh
# Install dependencies
make build

# Run the bot
make run
```

## Deployment

```sh
# Build container and start it
APP_ID=<app-id> PRIVATE_KEY=<pem-value> make deploy
```
