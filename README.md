# Book World

Book World is a local story/chat web app with a Go backend, PostgreSQL storage, and a Vue 3 frontend.

## Local Docker Run

```bash
docker compose up --build
```

Then open:

```text
http://localhost:3000
```

The compose stack starts:

- `postgres`: PostgreSQL database
- `book-world`: one Go server that serves both `/api` and the built Vue frontend

## GitHub Container Image

The workflow at `.github/workflows/docker-image.yml` builds and publishes one image to GitHub Container Registry on pushes to `main` or `master`:

- `ghcr.io/<owner>/<repo>:latest`

Pull requests only build the images; they do not push.

## Required Runtime Environment

Backend runtime variables:

```text
ADDR=:8080
STATIC_DIR=/app/frontend
DATABASE_URL=postgres://postgres:postgres@postgres:5432/book_world?sslmode=disable
DEFAULT_MODEL=gpt-4o-mini
CONTEXT_CHAR_BUDGET=48000
REPLY_CHAR_RESERVE=6000
FRONTEND_ORIGIN=http://localhost:3000
```

Do not commit real `.env` files or API/database secrets.
