# go-writer — Claude Context

## What This Service Is

The writing service for the personal-enterprise project. A hierarchy of Projects → Subjects → Texts for organizing long-form writing (books, essays, research papers, blogs — the labels stay generic; the user decides what they mean per project). Backed by MongoDB Atlas. Validates JWTs issued by go-auth — does not issue tokens.

Built from `go-service-template`. All three domains are implemented and wired; the cross-cutting concerns other services have (tracing, demo seeding, Pub/Sub) are not. See **Current State** below.

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
  ...                ← leftover template package; still present, still unused — safe to delete
utils/
  dumbwaiter/        ← NewID() — uuid generation
  extraction/        ← UserIDFromRequest() — pulls the authed user id off the request context
writer/              ← all three domains, one flat package (see below)
  project_model.go   project_store.go   project_handler.go   project_routes.go
  subject_model.go   subject_store.go   subject_handler.go   subject_routes.go
  text_model.go      text_store.go      text_handler.go      text_routes.go
tracer/              ← tracer client, same pattern as other Go services (not yet called)
```

---

## Why one package

Project, Subject, and Text live in a single flat `writer` package, not three. This deliberately breaks symmetry with the other Go services in this project, which use a package per domain.

The three-package layout doesn't compile. `ProjectResponse` embeds `[]Subject` and `SubjectResponse` embeds `[]Text`, and the subject read wanted its parent `Project` for a deep-linked subject page — parent-to-child and child-to-parent references between the same two packages is an import cycle, which Go forbids outright. The deeper reason is that these three aren't independent domains: they're one aggregate under one root, with a shared lifecycle (the project delete cascade) and no read that's meaningful outside the hierarchy. A package boundary with no dependency boundary behind it is just ceremony, and here the compiler said so.

Consequences of the flattening, all of which are load-bearing:

- **Directory is flat.** A subdirectory in Go is a separate package; nesting `writer/subject/` would recreate the exact cycle. There is nothing between `writer/` and its twelve files.
- **Filename prefixes are the only grouping.** `subject_store.go`, not `store.go` — the prefix does the work the directory used to.
- **Identifiers carry the domain.** `ProjectStore` / `SubjectStore` / `TextStore`, `NewSubjectHandler`, `ProjectRoutes`. Bare `Store` and `Handler` would collide three ways.
- **Sentinels are per domain**: `ErrProjectNotFound`, `ErrSubjectNotFound`, `ErrTextNotFound`. Watch these — they're all the same type in one namespace now, so a store returning the wrong domain's sentinel compiles cleanly and silently turns a 404 into a 500. This has already happened twice.
- **Encapsulation is convention, not compiler-enforced.** `projectDoc`, `subjectDoc`, and `textDoc` are visible to every file in the package. Keep each doc struct and its `toDomain()` used only by its own store.

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

> **Status: designed, not implemented — and currently non-functional end to end.** `sortOrder` exists on the `Project` domain type and `projectDoc`. It does **not** exist on `subjectDoc` or `textDoc`, so it is never persisted or read back for subjects and texts even though `Subject`, `Text`, and their create/update requests all declare it. `TextStore.Update` has no branch for it. No list query sorts on it. Everything below is intended design.

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

> **Status: design, not implemented.** There is no `TextSummary` type and no `GetSummariesByProject`. `TextStore` exposes `GetAllBySubject` only, returning full `Text` documents including `Content`. Project detail currently returns project + subjects with no texts at all — text cards can't render until this is built.

`Content` is the largest field on a `Text` — full rich-text/formatting data, and it can grow substantially per document. Opening a Project only needs `id`, `title`, `description`, `subjectId`, and `sortOrder` per text (to render subject columns and text cards) — it never needs `content` at that point. Fetching `content` for every text in a project up front would mean shipping a lot of data nothing in that view renders.

- **List view** (`GetSummariesByProject`): returns `[]TextSummary` — a distinct domain type with no `Content` field (`id`, `title`, `description`, `subjectId`, `projectId`, `sortOrder`). Excludes `content` at the MongoDB query level via `options.Find().SetProjection(bson.D{{Key: "content", Value: 0}})`, not just at the response-shaping level — the field never leaves the database.
- **Detail view** (`GetByID`): returns the full `Text`, including `content`. Called when opening a Subject or Text view for one specific text.

A distinct `TextSummary` type — rather than reusing `Text` with `Content` left blank — avoids ambiguity between "this text has no content" and "content wasn't fetched."

### Composing parent + children

> **Status: partially implemented.** Both composed reads exist and both compose in the handler. Project detail returns project + subjects (no texts yet — see the status note above). Subject detail returns subject + full texts, as designed.

Every "get one" returns that entity plus its children. There is no standalone list endpoint for subjects or texts, because a list of either is only ever meaningful inside its parent — the parent's read already carries it.

- **Projects list** (dashboard cards) — bare `Project` documents scoped by `userId`. Deliberately does **not** compose: one subject query per project is the actual N+1 in this service.
- **Project detail** (board page) — project + its subjects + a `TextSummary` per text. Two levels below the project. Text cards render title and description only; `content` never enters this response.
- **Subject detail** (reading/editing page) — subject + the **full** `Text` including `content` for each of its texts. One level below the subject. This is where prose is loaded, which is viable because a subject is chapter-sized by construction — a handful of texts, not a whole book.

Texts are leaves and compose nothing.

**Both reads compose in the handler.** `ProjectHandler.GetOne` calls `ProjectStore.GetByID` then `SubjectStore.GetAllByProject`; `SubjectHandler.GetOne` calls `SubjectStore.GetByID` then `TextStore.GetAllBySubject`. Each store decodes its own documents into its own domain types; the handler assembles the response struct.

This is the cheaper option on the merits and stays the default. The round-trip saving from a `$lookup` pipeline is unmeasurable at this scale — flat queries with no fan-out, since `Text` carries `projectId` as well as `subjectId`, so a single query can fetch every text summary in a project and Go groups them by `subjectId` in memory.

An earlier version of this document specified a `$lookup` pipeline for the project read, justified by keeping each collection's document shape private to the package that owned it. That argument died with the three-package layout — `projectDoc`, `subjectDoc`, and `textDoc` are now all visible inside `writer`, so a pipeline would no longer need shadow decode structs and no longer costs anything in encapsulation. Nothing currently requires one. If the project detail read ever grows deep enough to want it, these still apply:

- **`$lookup` does not preserve order.** Every sub-pipeline needs an explicit `$sort` on `sortOrder`. It will appear to work without one whenever insertion order happens to match intended order.
- **Scope every `$match` on `userId`, not just the foreign key.** A sub-pipeline matching only `projectId`/`subjectId` returns other users' rows. Worth an integration test pinned to a second user's data specifically.

**Cross-collection access is concentrated in the stores.** `ProjectStore` holds `subjects` and `texts` handles for its delete cascade; `SubjectStore` holds `texts` (cascade) and `projects` (create-time FK check); `TextStore` holds `subjects` (create-time FK check). Handlers never touch a collection they don't own.

---

## Database

MongoDB Atlas. Collections: `projects`, `subjects`, `texts`.

### Indexes (created at startup in `db_connection/db.go`)

- `projects.userId`, `projects.expiresAt` (sparse TTL)
- `subjects.userId`, `subjects.expiresAt` (sparse TTL)
- `texts.userId`, `texts.expiresAt` (sparse TTL)

### Store pattern

Same as other Go services: a private `*Doc` struct with `bson` tags, a domain struct in `_model.go` with plain Go types, and `toDomain()` to convert. IDs stored as strings (`uuid.UUID.String()`), parsed back on read.

Each store's constructor takes the whole `*mongo.Database`, so a store that needs a sibling collection just takes another handle — no signature changes anywhere upstream. See the cross-collection note under **Composing parent + children** for who holds what.

### Delete cascades and referential integrity

No DB-level enforcement; both directions are application-layer.

- **Delete project** — `DeleteMany` on texts and subjects matching `{projectId, userId}`, then the project itself. Safe only because `projectId` is immutable (see **Moves are text-level only**).
- **Delete subject** — `DeleteMany` on texts matching `{subjectId, userId}`, then the subject.
- **Create subject / create text** — the store `CountDocuments` on `{_id, userId}` of the named parent before inserting, returning `ErrProjectNotFound` / `ErrSubjectNotFound` if it's absent or owned by someone else. The handler maps that to a `400`. Without this the parent id is an unchecked free-form string in the request body, and orphans are unreachable by any cascade. Note this applies to every caller, so anything seeding data has to create parents before children.

---

## Patterns

### Partial updates

All three domains use pointer fields on their update requests and append to the `$set` document only for fields that are non-nil — omitted fields are left untouched rather than overwritten. This is an intentional exception to the project's default full-object-replacement update pattern (see root `CLAUDE.md`): `Title`, `Description`, and `Content` can each change independently and frequently from the same editor view, and forcing a full-object PUT would mean resending the whole document (including potentially large `Content`) for a single-field change.

**The update document must be wrapped in `$set`.** `FindOneAndUpdate(ctx, filter, bson.D{{Key: "$set", Value: update}}, ...)` — passing the bare accumulated `bson.D` makes MongoDB reject the whole operation for having no atomic operator, and every update on that domain returns a 500. This has already been shipped broken twice.

`UpdateTextRequest.SubjectID` moves a text between subjects. That is expected to become its own operation eventually (recomputing `sortOrder` in the destination subject at the same time) rather than going through the general update. There is no `ProjectID` on any update request — `projectId` is immutable by design.

### Domain structure

Four files per domain — `<domain>_model.go`, `_store.go`, `_handler.go`, `_routes.go` — same as other services, but all twelve sit in the one `writer` package rather than in a directory per domain. See **Why one package**.

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

### Working

- All three domains have model, store, handler, and routes. Mounted at `/projects`, `/subjects`, `/texts` in `server/routes.go`, every route behind `authMiddleware`.
- Full CRUD on projects and texts. Subjects have everything but a list endpoint, by design — subjects are only ever read through their project.
- Every store query is scoped on `userId` alongside `_id`.
- Delete cascades and create-time FK validation (see **Delete cascades and referential integrity**).
- Demo handling: per-domain create limits (5 projects, 5 subjects, 20 texts) gated on `middleware.IsDemoFromContext`, and `expiresAt` populated from `middleware.ExpiresAtFromContext` on all three creates so the sparse TTL indexes collect demo records. Real users get `nil` and their records never expire.
- `app/app.go`, `server/routes.go`, and `cmd/server/main.go` are fully adapted — no template references left in the wiring.

### Not done

- **Tracing.** A `*tracer.Client` is threaded through all three `Routes` functions and never called. No spans are emitted. The `TRACER_SERVICE_*` env vars are declared but unused.
- **Demo seeding / Pub/Sub.** No subscriber, no `demo-registered` handler. Unlike the other services, a new demo account gets an empty writer.
- **`sortOrder`** — see the status note under **Ordering**. Nothing sorts; the field isn't persisted for subjects or texts.
- **`TextSummary` / projection** — see the status note under **Fetching: List vs Detail**. Project detail returns no texts, so the board's text cards have no data source yet.
- **Tests.** None, of either kind.
- **Deployment.** Not deployed to dev or prod. No Cloud Run service, no WIF binding.
- **`example/`** — the template package is still on disk and referenced by nothing but its own test file.

### Watch for

The three `Err*NotFound` sentinels share one namespace and one type. Returning the wrong domain's sentinel compiles silently and converts a 404 into a 500 — it has happened in `SubjectStore.Update` (returned `ErrTextNotFound`) and `TextStore.Delete` (returned `ErrSubjectNotFound`), both since fixed. Check the sentinel matches the store whenever you touch one.
