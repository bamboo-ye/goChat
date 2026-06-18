# goChat

## Local configuration

The application reads non-secret defaults from `config/app.yml`. Secrets and
environment-specific values must be supplied with environment variables.

```bash
export GOCHAT_MYSQL_DSN='gochat:password@tcp(127.0.0.1:3306)/goChat?charset=utf8mb4&parseTime=true&loc=Local'
export GOCHAT_REDIS_ADDR='127.0.0.1:6379'
export GOCHAT_REDIS_PASSWORD=''
export GOCHAT_SERVER_PORT=':8081'
export GOCHAT_AI_PROVIDER='openrouter'
export GOCHAT_AI_BASEURL='https://openrouter.ai/api/v1'
export GOCHAT_AI_APIKEY='your-openrouter-api-key'
export GOCHAT_AI_MODEL='openai/gpt-oss-120b:free'
export GOCHAT_AI_APPTITLE='goChat'
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

The AI service uses OpenRouter's OpenAI-compatible `POST /chat/completions` API.
Character templates are seeded during migration.

After login, open the `AI` tab to create a role from a template, start a
conversation, and send messages. AI messages and usage are persisted in MySQL.
The MVP limits prompt history, input length, concurrent generation per
conversation, request timeout, and daily calls per user.

Before each model request, completed turns are sampled by calendar day.
Today's turns are retained at full density; turns from `N` days ago retain
approximately `1 / (N + 1)` of that day's conversations. User and assistant
messages stay paired, while the newest turns take priority within
`ai.historyLimit`.

- `GET /api/ai/templates`
- `POST|GET /api/ai/characters`
- `PUT|DELETE /api/ai/characters/:id`
- `POST|GET /api/ai/conversations`
- `DELETE /api/ai/conversations/:id`
- `GET|POST /api/ai/conversations/:id/messages`
- `POST /api/ai/conversations/:id/messages/:messageId/retry`
