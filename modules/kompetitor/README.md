# Kompetitor Module

Port of `app/Http/Controllers/KompetitorController.php` (plus the services it
leaned on: `CompetitorService`, `CompetitorMappingService`,
`ProductMappingService`, `ScrapingProductImportService`, the
`ResolvesScrapingPeriods` trait, `App\Helpers\StockCalculator` and the Shopee
branch of `Harga::getMarginNilai()`) from the Laravel app
`portal_produk_dbklik`.

This module is a faithful 1:1 business-logic port — it reads and writes the
**same shared MySQL database** as the Laravel app (no data migration, no
shadow tables). Payload shapes match the old controller's
`response()->json(...)` bodies, so the panels can be pointed here without
touching the front end.

## Layers

- `domain/entities` — GORM-tagged structs mirroring the Laravel migrations
  (`kompetitors`, `kompetitor_mappings`, `scraping_*`) plus the read-only
  reference tables the panels join against (`brand`, `kategori`, `item`,
  `harga`, `hpp`, `abc_analysis`, `warehouses`, marketplace fee tables).
- `domain/repositories` — persistence interfaces and their read-model rows.
- `domain/services` — the pure business logic, all covered by unit tests
  against fakes (no database needed):
  - `period_resolver.go` — `ResolvesScrapingPeriods`: newest batch per store
    inside a period, plus the comparison period.
  - `batch_resolver.go` — `ProductMappingService::resolveBatchByKompetitor`:
    effective batch per store with the 3-day stale fallback.
  - `mapping_resolver.go` — `CompetitorMappingService`: (kategori, brand)
    cells with the universal-brand fallback, request-scoped cache.
  - `text_matcher.go` — the TF-IDF cosine matcher.
  - `margin_calculator.go` / `stock_calculator.go` — Shopee margin and total
    stock.
- `application/dto` — input/output shapes mirroring the old JSON bodies.
- `application/services` — the `KompetitorService` facade, split by concern
  across `kompetitor_service.go` (CRUD), `mapping_service.go`,
  `scraping_service.go`, `our_product_service.go`, `ingest_service.go`.
- `infrastructure/repository` — GORM implementations against the shared
  `*gorm.DB`, plus `Unavailable*` fallbacks used when no database is
  configured (the app still boots; RPCs return `UNAVAILABLE`).
- `infrastructure/provider` — DI wiring, exposed as `KompetitorProvider`.
- `presentation/grpc` — proto contract, generated `pb`, and the server that
  maps requests onto the application service.
- `presentation/http` + `routes/api.go` — the same use cases over REST, for
  the Laravel portal (PHP there has no `ext-grpc`). Responses are the raw
  legacy JSON shapes, not the framework envelope, so the portal can pass them
  through to the existing Blade/JS panels verbatim.

## RPCs

| RPC                 | Laravel source                                     |
|---------------------|----------------------------------------------------|
| `IndexMeta`         | `KompetitorController::indexNew` (view payload)    |
| `ManageData`        | `KompetitorController::manageData`                 |
| `CreateKompetitor`  | `KompetitorController::store`                      |
| `UpdateKompetitor`  | `KompetitorController::update`                     |
| `DeleteKompetitor`  | `KompetitorController::destroy`                    |
| `MappingMatrix`     | `KompetitorController::mappingMatrix`              |
| `MappingCell`       | `KompetitorController::mappingCell`                |
| `MappingCellUpdate` | `KompetitorController::mappingCellUpdate`          |
| `Stats`             | `KompetitorController::stats`                      |
| `Products`          | `KompetitorController::dataNew`                    |
| `LegacyProducts`    | `KompetitorController::data`                       |
| `BatchCodes`        | `CompetitorService::getUniqueCompetitorAndBatch`   |
| `FilterOptions`     | `KompetitorController::filterOptions`              |
| `OurProducts`       | `KompetitorController::ourProductData`             |
| `ImportProducts`    | `ScrapingProductImportService::import` (rows only) |

### What stayed in Laravel

`previewImportFiles` / `importFiles` / `importProduct` parse uploaded Excel
files, match filenames against `shops.json` and enforce the header rules.
That parsing stays in the Laravel app; it POSTs the parsed rows to
`ImportProducts`, which persists them, recomputes `scraping_summaries`,
marks the batch completed and refreshes `scraping_product_mappings` — the
Go half of `ScrapingBatch::markCompleted()`.

Blade views and the `permission:kompetitor-list` middleware also stay on the
Laravel side; this service assumes its caller is already authorized.

## Business rules (ported verbatim — confirm against the PHP before changing)

- **Mapping.** Kompetitors for an item come from the `(kategori, brand)`
  cell. A brand may also hold a *universal* cell (`kategori_id NULL`) used as
  fallback for kategoris with no cell of their own. A filled specific cell
  wins outright — the two are never unioned.
- **Periods.** Each batch only covers some stores, so a period resolves to
  the newest batch *per store* inside it; the comparison period is each
  store's newest batch before the period starts. Stores absent from the
  period are excluded from the comparison too. An empty pair set becomes
  `1 = 0` so an empty period never degrades into "all data".
- **Our Product staleness.** Stores present in the latest batch read from it;
  a missing product there is a real signal, so the cell stays blank. Stores
  not yet scraped in the latest batch fall back to their last completed batch,
  no matter how old, and are flagged `stale`.
- **IDF corpus.** Token weights come from the full corpus (every product name
  in the batches actually read + every mappable item name), cached 12 hours
  per batch set. Scoring a page against a page-local corpus would make
  borderline matches flicker depending on which items share a page.
- **Harga tayang.** Our own store (`DBKLIK_KOMPETITOR_ID`) is scraped like any
  other store, but its price is our marketplace listing price, not a
  competitor price: it is excluded from the kompetitor columns, shown as
  `harga_tayang`, and is matchable against every item regardless of mapping.
- **Shopee margin.** The Our Product panel passes the *default* free-ongkir
  type (`biasa`, 40k cap), not the row's `free_ongkir_type` column — that is
  what the PHP caller does, and it is preserved here on purpose.

## Configuration

| Env var                      | Default | Meaning                                        |
|------------------------------|---------|------------------------------------------------|
| `DBKLIK_KOMPETITOR_ID`       | `0`     | Our own store's row in `kompetitors`; 0 hides the Harga Tayang column |
| `KOMPETITOR_MATCH_THRESHOLD` | `0.6`   | TF-IDF cosine cutoff for accepting a match     |
| `KOMPETITOR_MAX_STALE_DAYS`  | `3`     | How old a fallback batch may be                |

## Regenerating the proto

```bash
protoc --proto_path=modules/kompetitor/presentation/grpc/proto \
  --go_out=modules/kompetitor/presentation/grpc/pb --go_opt=paths=source_relative \
  --go-grpc_out=modules/kompetitor/presentation/grpc/pb --go-grpc_opt=paths=source_relative \
  kompetitor.proto
```
