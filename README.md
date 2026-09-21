# gdrive-clone backend

Go stdlib net/http (1.22 mux) + raw pgx + Neon Postgres + Cloudflare R2 + Google OAuth.

## Setup

1. `cp .env.example .env` and fill in values.
2. Neon: create a project, copy the **pooled** connection string into `DATABASE_URL`.
3. Run the schema against it:
   ```
   psql "$DATABASE_URL" -f internal/db/migrations/0001_init.sql
   ```
   (No migration tool wired up yet — single file is fine for now. Add `golang-migrate`
   once you have more than one migration to track.)
4. Google Cloud Console → OAuth consent screen + credentials → Web application →
   redirect URI `http://localhost:8080/auth/google/callback`.
5. Cloudflare R2 → create bucket → create API token (Account-scoped, not Global) →
   fill `R2_*` vars.
6. `go mod tidy`
7. `go run ./cmd/api`

## Design notes / things I skipped on purpose

- **Direct-to-R2 uploads/downloads via presigned URLs.** The Go server never touches
  file bytes. `/files/init` returns a presigned PUT; client uploads straight to R2;
  `/files/{id}/complete` does a `HeadObject` to verify it actually landed before
  marking the row `confirmed`. Never trust the client's "done" claim alone.
- **Soft deletes everywhere.** `deleted_at` on files/folders. Deleting a folder
  recursively soft-deletes its subtree via a recursive CTE, synchronously (cheap —
  it's just an UPDATE). The R2 objects themselves are *not* deleted synchronously.
- **Missing: the reaper job.** You need a background process (cron, or a scheduled
  Cloudflare Worker, or just a goroutine with a ticker) that finds files with
  `deleted_at IS NOT NULL` older than some grace period and calls `DeleteObject` on
  R2, then hard-deletes the row. I didn't build this yet — say the word and I'll add
  `cmd/reaper`.
- **No rename/move endpoints yet.** Straightforward to add — `UPDATE folders SET
  name = $1` / `UPDATE files SET folder_id = $1` with the same ownership checks used
  elsewhere.
- **No request size limit on JSON bodies** — add `http.MaxBytesReader` in production;
  irrelevant for file bytes since those go straight to R2, not through this server.
- **CORS is single-origin** (`FRONTEND_URL`). Fine for one frontend; swap for a
  proper allowlist if you'll have multiple origins.
- **Unique constraint on (folder, name)** — duplicate names in the same folder are
  rejected at the DB level (409), not silently allowed like real Drive does with
  "(1)" suffixes. Decide if you want that UX later; it's a partial unique index so
  easy to drop.

## API surface

```
GET  /auth/google/login
GET  /auth/google/callback
POST /auth/logout                    (auth required)
GET  /auth/me                        (auth required)

POST   /folders                      { name, parentId? }
GET    /folders/contents             root contents
GET    /folders/{id}/contents
DELETE /folders/{id}

POST   /files/init                   { name, mimeType, folderId? } -> { fileId, uploadUrl }
POST   /files/{id}/complete          verify + confirm
GET    /files/{id}/download          -> { downloadUrl }
DELETE /files/{id}
```

All protected routes expect an `httpOnly` `session` cookie (set automatically after
the OAuth callback).
