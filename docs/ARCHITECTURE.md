# Architecture

The framework skeleton (`cmd/`, `internal/`, `pkg/`, `routes/`) is the same
one documented in [../ARCHITECTURE.md](../ARCHITECTURE.md) and used by
`DBKLIK-GO-ARCHITECTURE`. This document only covers what is specific to this
service.

## Why a separate service

`KompetitorController` in `portal_produk_dbklik` had grown into a mixed bag:
competitor CRUD, the (kategori × brand) mapping panel, scraping statistics,
two product tables, the Our Product price comparison, and Excel import — all
in one 700-line controller leaning on six services. Extracting it gives the
competitor domain its own deployable unit, its own release cadence, and unit
tests over business rules (period resolution, mapping fallback, TF-IDF
scoring, margin maths) that previously required a database to exercise.

## Boundaries

```text
┌────────────────────────────┐        gRPC        ┌───────────────────────────┐
│ portal_produk_dbklik       │ ─────────────────► │ dbklik-kompetitor-service │
│ (Blade views, auth,        │                    │ (this repo)               │
│  permissions, Excel parse) │ ◄───────────────── │                           │
└────────────┬───────────────┘                    └─────────────┬─────────────┘
             │                                                  │
             └──────────────► shared MySQL database ◄────────────┘
```

- **Laravel keeps**: views, routing, `permission:kompetitor-list`, session
  auth, uploaded-file handling and Excel parsing (`shops.json` filename
  matching, header rules). This service assumes its caller is authorized.
- **This service owns**: every read/write of `kompetitors`,
  `kompetitor_mappings` and the `scraping_*` tables, plus the read-only joins
  the panels need (`brand`, `kategori`, `item`, `harga`, `hpp`,
  `abc_analysis`, `warehouses`, marketplace fee tables).
- **The database stays shared.** Both sides talk to the same MySQL instance,
  so no migration was needed and the Laravel app can be cut over one panel at
  a time. That also means schema changes must stay backwards compatible for
  both sides — the Laravel migrations remain the single source of truth for
  the schema; the Go entities mirror them.

## Layering inside the module

`presentation/grpc` → `application/services` → `domain/services` +
`domain/repositories` ← `infrastructure/repository`.

The domain layer never imports gorm, gin or grpc; every business rule is
reachable with in-memory fakes, which is what `modules/kompetitor/tests/unit`
does. Repositories are interfaces in `domain/repositories`; the GORM
implementations and the `Unavailable*` no-database fallbacks both live in
`infrastructure/repository`, and `infrastructure/provider` picks between them
at boot.

## Cutover notes

- Response shapes match the old controller's JSON bodies, so the front end
  does not change — only the Laravel controller's body does (call the gRPC
  client instead of the local services).
- The service maps domain errors to gRPC codes in `pkg/grpcerrors`;
  validation failures arrive as `INVALID_ARGUMENT` and should be rendered as
  422 on the Laravel side, matching the old validation responses.
- With `DB_DRIVER` unset every RPC returns `UNAVAILABLE` instead of panicking,
  so the service still boots in environments without a database.
