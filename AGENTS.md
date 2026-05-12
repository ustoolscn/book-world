# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Project overview

Book World is a two-part local web app:

- `backend/`: Go HTTP API server using the standard library router, PostgreSQL via `pgx`, embedded SQL migrations, and an OpenAI-compatible chat-completions client.
- `frontend/`: Vue 3 + Vite single-page app using hash routing and browser storage for session/local draft state.

There is no root package manager or git repository configuration in this directory; run backend and frontend commands from their respective subdirectories.

## Common commands

### Backend (`backend/`)

```bash
go run ./cmd/server
go test ./...
go test ./internal/contextbuilder
go test ./internal/contextbuilder -run TestName
go fmt ./...
go vet ./...
```

The server defaults to `:8080` and connects to `postgres://postgres:postgres@localhost:5432/book_world?sslmode=disable`. Configuration is loaded from environment variables and from `.env` in `backend/` or the repository root.

Useful backend environment variables:

- `ADDR`: HTTP listen address, default `:8080`
- `DATABASE_URL`: PostgreSQL connection string
- `DEFAULT_MODEL`: fallback model name, default `gpt-4o-mini`
- `CONTEXT_CHAR_BUDGET`: prompt construction budget, default `48000`
- `REPLY_CHAR_RESERVE`: reserved response budget, default `6000`
- `FRONTEND_ORIGIN`: CORS origin, default `http://localhost:5173`

### Frontend (`frontend/`)

```bash
npm run dev
npm run build
npm run preview
```

`npm run dev` starts Vite on all interfaces. The Vite dev server proxies `/api` to `http://localhost:8080`, so run the Go backend separately for end-to-end development. There is no frontend test script configured.

## Architecture

### Backend flow

`cmd/server/main.go` loads config, connects to PostgreSQL, runs embedded migrations from `internal/db/migrations`, then wires `internal/api.Server` with the database and LLM client.

`internal/api/server.go` owns routing, CORS, JSON helpers, and bearer-token authentication. All routes except `POST /api/identity/enter` require an auth session token. Identity entry stores the user's OpenAI-compatible `baseUrl` and `apiKey`, then returns a generated session token.

The main route groups are:

- `/api/identity/enter`: creates or updates a user identity and auth session.
- `/api/models`: lists models from the user's configured OpenAI-compatible endpoint.
- `/api/stories...`: lists stories, edits story settings, and manages saved chat sessions.
- `/api/chat/stream`: builds story context and streams Server-Sent Events back to the frontend.

`internal/contextbuilder` constructs the LLM message list. It combines fixed Chinese story-host rules, user profile, story settings, characters, triggered world-info entries, optional summary, recent messages within budget, and the current user input. World-info triggering scans the current input plus the last six recent messages for configured keywords.

`internal/llm` is provider-agnostic but OpenAI-compatible: it calls `/v1/models` and `/v1/chat/completions`, supports streaming SSE responses, falls back to non-streaming JSON when needed, and maps `thinkingEffort` to `reasoning_effort`.

`internal/model` contains shared backend DTO/domain structs. Database reads are mostly raw SQL in `internal/api/queries.go` and endpoint files.

### Database

Migrations are embedded with `go:embed` and executed lexicographically on every startup. Current migrations use `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, and seed two default stories, characters, and world-info records with `ON CONFLICT DO NOTHING`.

Main tables:

- `users` and `auth_sessions` for stored provider identity and bearer-session auth.
- `stories`, `story_characters`, and `world_info` for story configuration and prompt context.
- `chat_sessions` and `messages` for cloud-saved story records.

### Frontend flow

`src/main.ts` mounts Vue with `src/router.ts`. Routes are hash-based:

- `/app`: identity entry page.
- `/stories`: story list.
- `/stories/:slug`: chat page.
- `/stories/:slug/settings`: story settings editor.

API wrappers live in `src/api/`. `client.ts` centralizes JSON fetches and bearer-token attachment; `chat.ts` handles the SSE stream manually with `fetch()` and `ReadableStream` parsing.

The chat page is intentionally stateful. `ChatPage.vue` manages model selection, thinking effort, per-story local drafts, local saved records, cloud record upload/download, user profile prompts, stream metrics, and scroll behavior. Local browser keys use the `book-world-*` prefix. The user profile is encoded into saved-session `summary` with the `__BOOK_WORLD_USER_PROFILE__:` prefix.

Story settings edits go through `StorySettingsPage.vue` and persist to the backend `stories` table; new chats use the updated story prompts and opening message.
