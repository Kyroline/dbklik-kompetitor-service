# API

The module is served over two transports backed by the same application
service:

- **gRPC** (`cmd/grpc`, `GRPC_PORT`, default `9090`) — the contract lives in
  [modules/kompetitor/presentation/grpc/proto/kompetitor.proto](../modules/kompetitor/presentation/grpc/proto/kompetitor.proto),
  generated code sits next to it in `pb/`. Use this for service-to-service
  callers.
- **HTTP** (`cmd/api`, `APP_PORT`, default `8080`, mounted under `/api/v1`)
  — what the Laravel portal proxies, because PHP there has no `ext-grpc`.
  Bodies are the RAW legacy shapes (no envelope) so the portal can pass them
  through verbatim to the existing Blade/JS panels.

`cmd/api` also exposes the framework's `GET /health`.

## HTTP endpoints

| Method | Path | RPC equivalent |
|--------|------|----------------|
| GET | `/api/v1/kompetitor/meta` | `IndexMeta` |
| GET | `/api/v1/kompetitor/manage/data` | `ManageData` |
| POST | `/api/v1/kompetitor/manage` | `CreateKompetitor` |
| PUT | `/api/v1/kompetitor/manage/:id` | `UpdateKompetitor` |
| DELETE | `/api/v1/kompetitor/manage/:id` | `DeleteKompetitor` |
| GET | `/api/v1/kompetitor/mapping/matrix` | `MappingMatrix` |
| GET | `/api/v1/kompetitor/mapping/cell` | `MappingCell` |
| POST | `/api/v1/kompetitor/mapping/cell` | `MappingCellUpdate` |
| GET | `/api/v1/kompetitor/stats` | `Stats` |
| GET | `/api/v1/kompetitor/new/data` | `Products` |
| GET | `/api/v1/kompetitor/data` | `LegacyProducts` |
| GET | `/api/v1/kompetitor/batches` | `BatchCodes` |
| GET | `/api/v1/kompetitor/our-product/filter-options` | `FilterOptions` |
| GET | `/api/v1/kompetitor/our-product/data` | `OurProducts` |
| POST | `/api/v1/kompetitor/import-product` | `ImportProducts` |

Array query parameters accept `key=a&key=b`, `key[]=a` (what the panels' JS
sends) and `key[0]=a&key[1]=b` (what PHP's `http_build_query` produces when
the portal forwards them).

On the HTTP side `INVALID_INPUT` maps to **422**, not 400 — these errors
stand in for Laravel's validation responses and the panels' JS branches on
422. Every error body is `{"message": "..."}`.

## Service `kompetitor.KompetitorService`

| RPC | Replaces (Laravel route) | Notes |
|-----|--------------------------|-------|
| `IndexMeta` | `GET kompetitor/new` view payload | Reference lists for the Riset Produk page |
| `ManageData` | `GET kompetitor/manage/data` | Kompetitor list + mapped-cell count |
| `CreateKompetitor` | `POST kompetitor/manage` | |
| `UpdateKompetitor` | `PUT kompetitor/manage/{kompetitor}` | `id` in the request body |
| `DeleteKompetitor` | `DELETE kompetitor/manage/{kompetitor}` | Refuses stores that already have scraping data |
| `MappingMatrix` | `GET kompetitor/mapping/matrix` | `counts` keyed `"kategoriID\|brandID"` |
| `MappingCell` | `GET kompetitor/mapping/cell` | Unset `kategori_id` = the brand's universal cell |
| `MappingCellUpdate` | `POST kompetitor/mapping/cell` | Empty `kompetitors` deletes the cell |
| `Stats` | `GET kompetitor/stats` | Absent `current`/`previous` = no data in that period |
| `Products` | `GET kompetitor/new/data` | DataTables shape (`draw`, `records_total`, ...) |
| `LegacyProducts` | `GET kompetitor/data` | Filtered by store NAME and batch CODE |
| `BatchCodes` | batch dropdown of the legacy table | |
| `FilterOptions` | `GET kompetitor/our-product/filter-options` | |
| `OurProducts` | `GET kompetitor/our-product/data` | |
| `ImportProducts` | body of `POST kompetitor/import-product` | Receives already-parsed rows |

## Error mapping

Domain errors are translated in `pkg/grpcerrors`:

| Domain code | gRPC code | HTTP status |
|-------------|-----------|-------------|
| `INVALID_INPUT` | `INVALID_ARGUMENT` | 422 (matches the old validation responses) |
| `NOT_FOUND` | `NOT_FOUND` | 404 |
| `CONFLICT` | `ALREADY_EXISTS` | 409 |
| `UNAVAILABLE` | `UNAVAILABLE` | 503 (database not configured) |
| anything else | `INTERNAL` | 500 |

## Conventions carried over from the controller

- Money fields that the old JSON rendered pre-formatted stay pre-formatted
  (`"Rp1.500.000"`); `LegacyProducts` additionally carries `harga_raw`.
- Dates are `YYYY-MM-DD` strings, both in filters and in
  `KompetitorCell.tanggal_scraping`.
- `start`/`length`/`draw` follow DataTables; `length <= 0` falls back to 25
  (10 for `OurProducts`), exactly as the PHP did.
- Empty `start_date`/`end_date` means "latest data"; `kompetitor_id = 0`
  means "every store".
