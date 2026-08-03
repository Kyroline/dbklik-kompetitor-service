# Go Framework Architecture

## Overview

This document defines the standard directory structure for building a modular Go framework inspired by Domain-Driven Design (DDD), Clean Architecture, and Hexagonal Architecture.

The primary goals are:

* Clear separation of responsibilities
* High testability
* Framework-independent business logic
* Modular architecture
* Easily extendable through extension points
* Support for REST API, CLI, Workers, and future interfaces (gRPC, GraphQL, etc.)

---

# Root Structure

```text
my-framework/
├── cmd/
├── internal/
├── pkg/
├── modules/
├── configs/
├── database/
├── resources/
├── routes/
├── docs/
├── scripts/
├── storage/
├── tests/
├── go.mod
├── go.sum
└── README.md
```

---

# Directory Explanation

## cmd/

Application entrypoints.

Example:

```text
cmd/
├── api/
│   └── main.go
├── worker/
│   └── main.go
└── cli/
    └── main.go
```

Each executable has its own startup configuration.

---

## internal/

Contains framework internals.

Example:

```text
internal/
├── app/
├── bootstrap/
├── config/
├── container/
├── database/
├── http/
├── logger/
├── middleware/
├── router/
└── server/
```

Packages inside `internal` cannot be imported by external projects.

---

## pkg/

Reusable packages.

Example:

```text
pkg/
├── cache/
├── config/
├── crypto/
├── errors/
├── filesystem/
├── helper/
├── httpclient/
├── logger/
├── queue/
├── response/
├── validator/
└── utils/
```

Anything inside `pkg` is intended to be imported by applications or other modules.

---

# Module Architecture

Every business feature lives inside the `modules` directory.

Example:

```text
modules/
├── inventory/
├── employee/
├── payroll/
├── attendance/
└── recruitment/
```

Each module is completely isolated.

---

# Module Structure

Example:

```text
modules/
└── inventory/
    ├── domain/
    ├── application/
    ├── infrastructure/
    ├── presentation/
    ├── database/
    ├── resources/
    ├── routes/
    ├── config/
    ├── tests/
    ├── docs/
    ├── module.json
    └── README.md
```

---

# Domain Layer

Contains pure business logic.

```text
domain/
├── entities/
├── valueobjects/
├── repositories/
├── services/
├── events/
├── specifications/
└── contracts/
```

The Domain layer:

* never imports Gin
* never imports Fiber
* never imports GORM
* never imports SQL drivers
* never depends on HTTP

Only business rules belong here.

---

## entities/

Business entities.

Example:

```text
entities/
├── Product.go
├── Warehouse.go
├── Stock.go
└── Category.go
```

Entities contain business behavior, not database logic.

---

## valueobjects/

Immutable objects.

Examples:

```text
SKU
Money
Quantity
Email
PhoneNumber
Percentage
```

---

## repositories/

Repository interfaces.

Example:

```go
type ProductRepository interface {
    FindByID(id string) (*Product, error)
    Save(product *Product) error
}
```

Only interfaces live here.

---

## services/

Pure domain services.

Examples:

```text
StockCalculator.go
AverageCostCalculator.go
PricingService.go
```

---

## events/

Domain events.

Example:

```text
StockAdjusted.go
GoodsReceived.go
LowStockReached.go
ProductArchived.go
```

---

## specifications/

Business rules.

Example:

```text
CanReceiveGoods.go
CanTransferStock.go
CanArchiveProduct.go
```

---

## contracts/

Business extension points.

Examples:

```go
type StockValuationMethod interface

type ReorderPolicy interface

type PricingCalculator interface
```

---

# Application Layer

Coordinates use cases.

```text
application/
├── actions/
├── commands/
├── queries/
├── handlers/
├── services/
├── dto/
└── mapper/
```

This layer orchestrates the Domain.

---

## actions/

Single-purpose actions.

Examples:

```text
ReceiveGoods.go
TransferStock.go
AdjustStock.go
ArchiveProduct.go
```

---

## commands/

Write operations.

Examples:

```text
CreateProductCommand.go
ReceiveGoodsCommand.go
AdjustStockCommand.go
```

---

## queries/

Read operations.

Examples:

```text
GetProductQuery.go
GetInventoryQuery.go
```

---

## handlers/

CQRS handlers.

```text
ReceiveGoodsHandler.go
CreateProductHandler.go
```

---

## services/

Application services.

Examples:

```text
InventoryService.go
WarehouseService.go
```

---

## dto/

Input and output DTOs.

```text
CreateProductDTO.go
ReceiveGoodsDTO.go
```

---

## mapper/

Converts DTO ↔ Domain Entity.

---

# Infrastructure Layer

Framework implementations.

```text
infrastructure/
├── persistence/
├── repository/
├── cache/
├── queue/
├── notification/
├── provider/
├── scheduler/
└── external/
```

---

## persistence/

Database implementation.

Examples:

```text
gorm/
sqlx/
mongo/
redis/
```

---

## repository/

Repository implementations.

Example:

```text
ProductRepository.go
WarehouseRepository.go
```

Implements domain repository interfaces.

---

## cache/

Redis or in-memory cache.

---

## queue/

Background jobs.

---

## notification/

Email, SMS, WhatsApp, Push.

---

## provider/

Dependency Injection registration.

---

## scheduler/

Cron jobs.

---

## external/

Third-party integrations.

Examples:

```text
SAP/
Midtrans/
Xendit/
Stripe/
Slack/
```

---

# Presentation Layer

Public interfaces.

```text
presentation/
├── http/
├── grpc/
├── graphql/
├── cli/
└── websocket/
```

---

## HTTP

```text
http/
├── controllers/
├── middleware/
├── requests/
├── responses/
└── resources/
```

---

## controllers/

Receive HTTP requests.

Controllers should remain thin.

Responsibilities:

* validate request
* call application layer
* return response

---

## requests/

Validation structs.

---

## responses/

Response builders.

---

## resources/

JSON transformers.

---

# Database

```text
database/
├── migrations/
├── seeders/
└── factories/
```

---

# Resources

```text
resources/
├── views/
├── lang/
└── templates/
```

---

# Routes

```text
routes/
├── api.go
├── web.go
└── cli.go
```

---

# Config

```text
config/
└── inventory.go
```

All module configuration lives here.

---

# Tests

```text
tests/
├── unit/
├── integration/
├── feature/
└── benchmark/
```

---

# Documentation

```text
docs/
├── README.md
├── EVENTS.md
├── EXTENSION_POINTS.md
├── CHANGELOG.md
├── API.md
└── ARCHITECTURE.md
```

---

# Module Manifest

Example:

```json
{
  "name": "inventory",
  "version": "1.0.0",
  "description": "Inventory Management Module",
  "author": "Framework Team",
  "dependencies": [],
  "providers": [
    "InventoryProvider"
  ]
}
```

---

# Dependency Flow

```
Presentation
      │
      ▼
Application
      │
      ▼
Domain
      ▲
      │
Infrastructure
```

Rules:

* Domain knows nothing.
* Application depends on Domain.
* Infrastructure implements Domain interfaces.
* Presentation calls Application only.

---

# Extension Points

The framework is designed to be extensible through contracts (interfaces).

Examples include:

* Repository implementations
* Authentication providers
* Cache drivers
* Queue backends
* Notification channels
* Storage providers
* Payment gateways
* Event dispatchers
* Logging adapters
* Validation rules
* Stock valuation methods
* Pricing strategies
* Reorder policies

Each extension point should expose a well-defined interface within the Domain or Application layer, while concrete implementations reside in the Infrastructure layer.

This allows developers to replace implementations without modifying business logic, following the Dependency Inversion Principle.

---

# Design Principles

* Domain-Driven Design (DDD)
* Clean Architecture
* SOLID Principles
* Dependency Injection
* Dependency Inversion Principle
* Interface-Driven Development
* CQRS-ready
* Event-Driven Architecture
* Test-First Friendly
* Modular Monolith Ready
* Microservice Migration Friendly
* Framework Agnostic Business Logic

---

# Benefits

* Highly maintainable
* Easily testable
* Clear separation of concerns
* Independent business logic
* Replaceable infrastructure
* Scalable module architecture
* Easier onboarding for new developers
* Ready for enterprise-scale applications
