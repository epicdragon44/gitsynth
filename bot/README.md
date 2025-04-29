# github bot

## development

```sh
# Install dependencies
make build

# Run the bot
make run
```

## deployment

```sh
# Build container and start it
APP_ID=<app-id> PRIVATE_KEY=<pem-value> make deploy
```
