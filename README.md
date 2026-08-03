# dbklik-kompetitor-service

Microservice yang memisahkan modul **Kompetitor** dari aplikasi Laravel
`portal_produk_dbklik`. Struktur direktori dan lapisan-lapisannya mengikuti
`DBKLIK-GO-ARCHITECTURE` (DDD + Clean/Hexagonal): kerangka framework di
`internal/` dan `pkg/`, logika bisnis di `modules/kompetitor/`.

Service ini membaca dan menulis **database MySQL yang sama** dengan aplikasi
Laravel — tidak ada migrasi data dan tidak ada tabel bayangan. Bentuk payload
mengikuti respons JSON controller lama, jadi panel yang sudah ada tinggal
diarahkan ke sini.

## Menjalankan

```bash
cp .env.example .env   # isi DB_DSN + DBKLIK_KOMPETITOR_ID
go run ./cmd/api       # HTTP di :8080 (APP_PORT), mount /api/v1
go run ./cmd/grpc      # gRPC di :9090 (GRPC_PORT)
```

Modul kompetitor disajikan lewat dua transport dengan application service
yang sama: **gRPC** untuk pemanggil service-to-service, dan **HTTP** untuk
portal Laravel yang tidak punya `ext-grpc` (portal mem-proxy endpoint
`/api/v1/kompetitor/*`). Daftar endpoint: [docs/API.md](docs/API.md).

## Struktur

```text
cmd/            entrypoint (api, grpc, worker, cli)
internal/       kerangka framework (bootstrap, container, config, router, server, middleware)
pkg/            paket reusable (errors, response, grpcerrors, cache, validator, logger, ...)
modules/
└── kompetitor/ modul bisnis — lihat modules/kompetitor/README.md
routes/         daftar modul yang dipasang ke HTTP (api.go) dan gRPC (grpc.go)
docs/           arsitektur, kontrak API, changelog
```

## Perintah

```bash
go build ./...
go vet ./...
go test ./...
```

Dokumentasi arsitektur kerangka: [ARCHITECTURE.md](ARCHITECTURE.md).
Detail modul, aturan bisnis dan daftar RPC: [modules/kompetitor/README.md](modules/kompetitor/README.md).
