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
- `backend`: Go API server
- `frontend`: Nginx-served Vue app, proxying `/api` to `backend`

## GitHub Container Images

The workflow at `.github/workflows/docker-images.yml` builds and publishes two images to GitHub Container Registry on pushes to `main` or `master`:

- `ghcr.io/<owner>/<repo>/backend`
- `ghcr.io/<owner>/<repo>/frontend`

Pull requests only build the images; they do not push.

## Required Runtime Environment

Backend runtime variables:

```text
ADDR=:8080
DATABASE_URL=postgres://postgres:postgres@postgres:5432/book_world?sslmode=disable
DEFAULT_MODEL=gpt-4o-mini
CONTEXT_CHAR_BUDGET=48000
REPLY_CHAR_RESERVE=6000
FRONTEND_ORIGIN=http://localhost:3000
```

Do not commit real `.env` files or API/database secrets.
