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

Fixed three levels: **Project → Subject → Text**. No arbitrary or self-referential nesting (e.g. a Subject cannot have a parent Subject). A case that seems to need a fourth level — a series of books, say — is modeled as separate Projects instead. This keeps drag-and-drop reordering bounded to known operations rather than needing to support recursive tree manipulation.

The frontend renders a Project as a board: Subjects are columns, Texts are cards.

### Moves are text-level only

A Text can move to a different Subject **within the same Project**. That is the only move operation in the domain:

- A Subject never moves to a different Project — Subjects only reorder within their own Project.
- A Text never moves to a different Project.

So `projectId` is immutable on both `Subject` and `Text` once set. The project delete cascade depends on this: it matches subjects and texts on `projectId` directly, which is only safe because that field can never drift from the document's actual parent.

---

## Ordering

Position within a parent is tracked with a **`sortOrder` field on the child document itself** — not an array of child ids on the parent.

- `Text.sortOrder` — position within its `subjectId`
- `Subject.sortOrder` — position within its `projectId` — identical mechanism, one level up

**Why `sortOrder` and not `order` / `position` / `index`:** the value is a sparse sort key, not a place count — `position` and `index` both imply a dense `0, 1, 2, 3` sequence, which these values are not. Bare `order` reads as a noun in most other domains ("an order") and is SQL-reserved, which would be an annoyance in any future analytics export. `sortOrder` says exactly what the field is for and reads correctly at every call site.

**Why not an id-array on the parent:** the child already carries its parent's id as a foreign key (`Text.subjectId`, `Subject.projectId`) for scoping and authorization queries. An id-array on the parent would record that same membership fact a second time. Moving a text to a new subject would then require writing the text's `subjectId` _and_ removing/inserting its id in two separate subject documents — three writes with no atomicity between them, and a real risk of the array and the foreign key disagreeing if one write fails. Keeping `sortOrder` on the child means every reorder or move is a single-document write.

**Values are plain integers, not floats.** No midpoint averaging.

- **Create**: `sortOrder` = current max `sortOrder` among siblings + `10,000`
- **Insert between two siblings** (drag-and-drop drop): `sortOrder` = the `sortOrder` value of the item immediately before the drop position, + `1`
- This consumes a gap linearly from one side. A `10,000`-wide gap between two siblings allows up to `9,999` sequential inserts into it before it's exhausted — far beyond what a human editing one subject's contents will ever produce.
- **Rebalance** (only if a gap is ever actually exhausted): refetch that parent's children sorted by `sortOrder`, reassign clean spaced values (`10000, 20000, 30000, ...`) in one bulk write scoped to just that parent's children.
- `sortOrder` values are scoped per parent, so magnitude never grows with total user count or total documents in the collection — only with how many siblings one specific parent has.

---

## Fetching: List vs Detail

`Content` is the largest field on a `Text` — full rich-text/formatting data, and it can grow substantially per document. Opening a Project only needs `id`, `title`, `description`, `subjectId`, and `sortOrder` per text (to render subject columns and text cards) — it never needs `content` at that point. Fetching `content` for every text in a project up front would mean shipping a lot of data nothing in that view renders.

- **List view** (`GetSummariesByProject`, replacing today's `GetAllByProject`): returns `[]TextSummary` — a distinct domain type with no `Content` field (`id`, `title`, `description`, `subjectId`, `projectId`, `sortOrder`). Excludes `content` at the MongoDB query level via `options.Find().SetProjection(bson.D{{Key: "content", Value: 0}})`, not just at the response-shaping level — the field never leaves the database.
- **Detail view** (`GetByID`): returns the full `Text`, including `content`. Called when opening a Subject or Text view for one specific text.

A distinct `TextSummary` type — rather than reusing `Text` with `Content` left blank — avoids ambiguity between "this text has no content" and "content wasn't fetched."

### Composing parent + children

> **Status: design, not yet implemented.** Revisit this subsection once the pipeline and the subject read actually exist — stage details and field lists below are intended shape, not observed behavior.

Every "get one" returns that entity plus its children. There is no standalone list endpoint for subjects or texts, because a list of either is only ever meaningful inside its parent — the parent's read already carries it.

- **Projects list** (dashboard cards) — bare `Project` documents scoped by `userId`. Deliberately does **not** compose: one subject query per project is the actual N+1 in this service.
- **Project detail** (board page) — project + its subjects + a `TextSummary` per text. Two levels below the project. Text cards render title and description only; `content` never enters this response.
- **Subject detail** (reading/editing page) — subject + the **full** `Text` including `content` for each of its texts. One level below the subject. This is where prose is loaded, which is viable because a subject is chapter-sized by construction — a handful of texts, not a whole book.

Texts are leaves and compose nothing.

**The rule: aggregate when composing more than one level down; compose in the handler when it's one.**

- **`project` uses a `$lookup` pipeline.** `project.Store` returns the entire detail read in one query, decoding into `projectDetailDoc` → `detailSubjectDoc` → `detailTextDoc`. The embedded `projectDoc` needs `bson:",inline"` — the BSON codec nests embedded structs by default and would otherwise silently decode a zero-valued project. The domain-side `ProjectResponse` needs no equivalent tag, since `encoding/json` promotes embedded fields automatically.
- **`subject` composes in the handler.** `subject.Store.GetByID` plus `text.Store.GetAllBySubject`, assembled in `subject.Handler`. No new decode structs — each store already decodes its own documents into its own domain types.

**Why two approaches instead of one.** Handler composition is the cheaper option on the merits: the round-trip saving from a pipeline is unmeasurable at this scale (three flat queries, no fan-out — `Text` carries `projectId` as well as `subjectId`, so one query gets every text summary in a project and Go groups them by `subjectId`), and it keeps each collection's document shape private to the package that owns it. The pipeline's real cost is that `project` must carry decode structs shadowing shapes `subject` and `text` already own — schema knowledge crossing a package boundary with nothing to enforce it, unlike a method call the compiler checks.

That cost is accepted here deliberately, on the deeper of the two reads, so the service demonstrates both techniques and the tradeoff between them. It is not an oversight, and it should not be treated as the pattern to copy into a new domain without the same weighing.

**Pipeline gotchas:**

- **`$lookup` does not preserve order.** Both sub-pipelines need an explicit `$sort` on `sortOrder`. This will appear to work without it in early testing, whenever insertion order happens to match intended order.
- **Scope every `$match` on `userId`, not just the foreign key.** A sub-pipeline matching only `projectId`/`subjectId` returns other users' rows. Worth an integration test pinned to a second user's data specifically.
- Cross-collection **writes** stay in the owning package regardless. The only exception is `project.Store.Delete`, whose cascade issues raw `DeleteMany` calls against `subjects` and `texts` — filter-only, no decoding, so its entire coupling is a collection name and two field names.

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

`SubjectID` and `ProjectID` follow the same pointer pattern on `UpdateTextRequest`, but moving a text to a new subject is expected to become its own operation once the `subject` domain exists (recomputing `sortOrder` in the destination subject at the same time) rather than going through the general update.

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

- `text` domain: `text_model.go` and `text_store.go` are implemented (`GetAllByProject`, `GetAllBySubject`, `GetByID`, `Create`, `Update`). `text_handler.go` and `text_routes.go` are empty stubs. `Text.Delete` is stubbed with a comment referencing removing the id from a subject's `textOrder` array — that array-based approach is superseded by the `sortOrder`-on-child design above and needs updating once implemented.
- `sortOrder` field does **not** exist on the `Text` model yet — the scheme above is the intended design, not yet implemented.
- `GetAllByProject` currently returns the full `Text` (including `Content`) with no projection — the `TextSummary`/projection split in **Fetching: List vs Detail** is the intended design, not yet implemented.
- `subject` and `project` domains: not started (empty directories).
- `app/app.go`, `server/routes.go`, and `cmd/server/main.go` still reference the template's `example` package and `go-service-template` module path — not yet adapted to this service's real domains.
