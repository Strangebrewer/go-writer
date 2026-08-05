# go-writer — Claude Context

## What This Service Is

The writing service for the personal-enterprise project. A hierarchy of Projects → Subjects → Texts for organizing long-form writing (books, essays, research papers, blogs — the labels stay generic; the user decides what they mean per project). Backed by MongoDB Atlas. Validates JWTs issued by go-auth — does not issue tokens.

Built from `go-service-template`. Most of the scaffolding is still unadapted from that template — see **Current State** below before assuming anything is wired up.

---

## Architecture

```
cmd/
  server/main.go     ← wiring: config, DB, stores, server.New()
app/
  app.go             ← Application struct aggregating domain stores
server/
  server.go          ← chi router, global middleware
  routes.go          ← mounts domain routes with auth middleware
config/
  config.go          ← standard template config, no additions needed
db_connection/
  db.go              ← MongoDB Connect() — returns (*mongo.Client, *mongo.Database, error); creates indexes at startup
health/
  handler.go
middleware/
  auth.go
  logging.go
  requestid.go
example/
  ...                ← leftover template package; remove once a real domain is wired into app/server
project/             ← empty — domain not started
subject/             ← empty — domain not started
text/
  text_model.go      ← Text, CreateTextRequest, UpdateTextRequest
  text_store.go      ← GetAllByProject, GetAllBySubject, GetByID, Create, Update
  text_handler.go    ← stub, not yet implemented
  text_routes.go     ← stub, not yet implemented
tracer/              ← tracer client, same pattern as other Go services
```

---

## Domain Hierarchy

Fixed three levels: **Project → Subject → Text**. No arbitrary or self-referential nesting (e.g. a Subject cannot have a parent Subject). A case that seems to need a fourth level — a series of books, say — is modeled as separate Projects instead. This keeps drag-and-drop reordering bounded to two known operations (reorder-within-parent, move-between-parents) at every level, rather than needing to support recursive tree manipulation.

The frontend renders a Project as a board: Subjects are columns, Texts are cards.

---

## Ordering

Position within a parent is tracked with an **`order` field on the child document itself** — not an array of child ids on the parent.

- `Text.order` — position within its `subjectId`
- `Subject.order` — position within its `projectId` — identical mechanism, one level up

**Why not an id-array on the parent:** the child already carries its parent's id as a foreign key (`Text.subjectId`, `Subject.projectId`) for scoping and authorization queries. An id-array on the parent would record that same membership fact a second time. Moving a child to a new parent would then require writing the child's foreign key _and_ removing/inserting its id in two separate parent documents — three writes with no atomicity between them, and a real risk of the array and the foreign key disagreeing if one write fails. Keeping `order` on the child means every reorder or move is a single-document write.

**Values are plain integers, not floats.** No midpoint averaging.

- **Create**: `order` = current max `order` among siblings + `10,000`
- **Insert between two siblings** (drag-and-drop drop): `order` = the order value of the item immediately before the drop position, + `1`
- This consumes a gap linearly from one side. A `10,000`-wide gap between two siblings allows up to `9,999` sequential inserts into it before it's exhausted — far beyond what a human editing one subject's contents will ever produce.
- **Rebalance** (only if a gap is ever actually exhausted): refetch that parent's children ordered by `order`, reassign clean spaced values (`10000, 20000, 30000, ...`) in one bulk write scoped to just that parent's children.
- Order values are scoped per parent, so magnitude never grows with total user count or total documents in the collection — only with how many siblings one specific parent has.

---

## Fetching: List vs Detail

`Content` is the largest field on a `Text` — full rich-text/formatting data, and it can grow substantially per document. Opening a Project only needs `id`, `title`, `description`, `subjectId`, and `order` per text (to render subject columns and text cards) — it never needs `content` at that point. Fetching `content` for every text in a project up front would mean shipping a lot of data nothing in that view renders.

- **List view** (`GetSummariesByProject`, replacing today's `GetAllByProject`): returns `[]TextSummary` — a distinct domain type with no `Content` field (`id`, `title`, `description`, `subjectId`, `projectId`, `order`). Excludes `content` at the MongoDB query level via `options.Find().SetProjection(bson.D{{Key: "content", Value: 0}})`, not just at the response-shaping level — the field never leaves the database.
- **Detail view** (`GetByID`): returns the full `Text`, including `content`. Called when opening a Subject or Text view for one specific text.

A distinct `TextSummary` type — rather than reusing `Text` with `Content` left blank — avoids ambiguity between "this text has no content" and "content wasn't fetched."

---

## Database

MongoDB Atlas. Collections: `projects`, `subjects`, `texts`.

### Indexes (created at startup in `db_connection/db.go`)

- `projects.userId`, `projects.expiresAt` (sparse TTL)
- `subjects.userId`, `subjects.expiresAt` (sparse TTL)
- `texts.userId`, `texts.expiresAt` (sparse TTL)

### Store pattern

Same as other Go services: a private `*Doc` struct with `bson` tags, a domain struct in `_model.go` with plain Go types, and `toDomain()` to convert. IDs stored as strings (`uuid.UUID.String()`), parsed back on read.

---

## Patterns

### Partial updates

`text.Update` uses pointer fields (`*string`) on `UpdateTextRequest` and appends to the `$set` document only for fields that are non-nil — omitted fields are left untouched rather than overwritten. This is an intentional exception to the project's default full-object-replacement update pattern (see root `CLAUDE.md`): `Title`, `Description`, and `Content` can each change independently and frequently from the same editor view, and forcing a full-object PUT would mean resending the whole document (including potentially large `Content`) for a single-field change.

`SubjectID` and `ProjectID` follow the same pointer pattern on `UpdateTextRequest`, but moving a text to a new subject is expected to become its own operation once the `subject` domain exists (recomputing `order` in the destination subject at the same time) rather than going through the general update.

### Domain structure

Same four-file pattern as other services: `<domain>_model.go`, `_store.go`, `_handler.go`, `_routes.go`.

---

## Environment Variables

| Variable             | Description                                                  |
| -------------------- | ------------------------------------------------------------ |
| `PORT`               | HTTP port (defaults to 8080)                                 |
| `DATABASE_URL`       | MongoDB URI (`mongodb+srv://user:pass@cluster.mongodb.net/`) |
| `JWT_PUBLIC_KEY`     | RSA public key PEM for validating JWTs issued by go-auth     |
| `ALLOWED_ORIGINS`    | Comma-separated list of allowed CORS origins                 |
| `TRACER_SERVICE_URL` | go-tracer base URL                                           |
| `TRACER_SERVICE_KEY` | Auth key for `POST /spans` on go-tracer                      |

Copy `.env.example` to `.env.local` for local dev. Never commit `.env.local`.

---

## Current State

This service is mid-scaffold — most of it is still unadapted `go-service-template` boilerplate:

- `text` domain: `text_model.go` and `text_store.go` are implemented (`GetAllByProject`, `GetAllBySubject`, `GetByID`, `Create`, `Update`). `text_handler.go` and `text_routes.go` are empty stubs. `Text.Delete` is stubbed with a comment referencing removing the id from a subject's `textOrder` array — that array-based approach is superseded by the `order`-on-child design above and needs updating once implemented.
- `order` field does **not** exist on the `Text` model yet — the scheme above is the intended design, not yet implemented.
- `GetAllByProject` currently returns the full `Text` (including `Content`) with no projection — the `TextSummary`/projection split in **Fetching: List vs Detail** is the intended design, not yet implemented.
- `subject` and `project` domains: not started (empty directories).
- `app/app.go`, `server/routes.go`, and `cmd/server/main.go` still reference the template's `example` package and `go-service-template` module path — not yet adapted to this service's real domains.
