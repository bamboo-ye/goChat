# goChat

## Local configuration

The application reads non-secret defaults from `config/app.yml`. Secrets and
environment-specific values must be supplied with environment variables.

```bash
export GOCHAT_MYSQL_DSN='gochat:password@tcp(127.0.0.1:3306)/goChat?charset=utf8mb4&parseTime=true&loc=Local'
export GOCHAT_REDIS_ADDR='127.0.0.1:6379'
export GOCHAT_REDIS_PASSWORD=''
export GOCHAT_SERVER_PORT=':8081'
export GOCHAT_AI_BASEURL='https://api.openai.com/v1'
export GOCHAT_AI_APIKEY='your-api-key'
export GOCHAT_AI_MODEL='your-model'
go run .
```

See `.env.example` for all supported settings. The app checks MySQL and Redis,
runs GORM migrations, and then starts the HTTP server. UDP message broadcast is
disabled by default.

## Verification

```bash
go test ./...
go vet ./...
```

## Docker image

Build the Linux binary and image:

```bash
mkdir -p .docker-build
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o .docker-build/gochat .
docker build --pull=false -t gochat-docker:latest .
```

All chat APIs except registration and login require an `Authorization: Bearer
<token>` header. The browser client adds this header automatically after login.

## AI service API

The AI service uses an OpenAI-compatible `POST /chat/completions` provider.
Character templates are seeded during migration.

After login, open the `AI` tab to create a role from a template, start a
conversation, and send messages. AI messages and usage are persisted in MySQL.
The MVP limits prompt history, input length, concurrent generation per
conversation, request timeout, and daily calls per user.

- `GET /api/ai/templates`
- `POST|GET /api/ai/characters`
- `PUT|DELETE /api/ai/characters/:id`
- `POST|GET /api/ai/conversations`
- `DELETE /api/ai/conversations/:id`
- `GET|POST /api/ai/conversations/:id/messages`
- `POST /api/ai/conversations/:id/messages/:messageId/retry`
