# Changelog

## 2026-08-03 — Initial extraction

Split the kompetitor module out of `portal_produk_dbklik` into this service,
following the `DBKLIK-GO-ARCHITECTURE` layout.

Ported from Laravel:

- `KompetitorController` — CRUD, mapping matrix panel, stats, both product
  tables, filter options, Our Product.
- `CompetitorService::filterCompetitorProduct` /
  `getUniqueCompetitorAndBatch`.
- `CompetitorMappingService` — (kategori, brand) cells with universal-brand
  fallback, matrix counts, cell sync.
- `ProductMappingService` — TF-IDF matching, per-batch IDF corpus with 12h
  cache, effective-batch resolution with the 3-day stale window.
- `ResolvesScrapingPeriods` — period → batch-per-store resolution.
- `App\Helpers\StockCalculator` and the Shopee branch of
  `Harga::getMarginNilai()` / `getMarginPersen()`.
- `ScrapingProductImportService` + `ScrapingBatch::computeSummaries()` /
  `markCompleted()` — persistence half only; Excel parsing stays in Laravel
  and posts parsed rows to `ImportProducts`.

Not ported (deliberately):

- Blade views, `permission:kompetitor-list`, session auth.
- Excel/`shops.json` parsing (`previewImportFiles`, `importFiles` and the
  file handling of `importProduct`).
- `KompetitorController::index` (the older Meilisearch-era page).

## 2026-08-03 — HTTP transport + portal wired up

The Laravel portal cannot speak gRPC (no `ext-grpc`), so the module gained a
`presentation/http` layer served by `cmd/api` under `/api/v1/kompetitor/*`,
backed by the same application service as the gRPC server. Bodies are the raw
legacy JSON shapes so the portal can pass them through to the existing panels
untouched.

`portal_produk_dbklik` now proxies through `KompetitorServiceClient`; its
`ScrapingProductImportService` only parses Excel and posts the rows to
`POST /kompetitor/import-product`. See that repo's
`docs/changelog/2026-08-03-kompetitor-microservice.md`.

Also added: period validation (`start_date`/`end_date` format and ordering)
in the application service, so gRPC callers get the same 422-equivalent the
Laravel validator used to produce.

Still open:

- The portal's multi-file import (`importFiles`, `shops.json` matching) still
  writes to the shared database directly instead of going through this
  service.
- No integration test runs against a real database yet; only the pure domain
  logic is covered by unit tests.
- No authentication on either transport — the caller is assumed authorized.
