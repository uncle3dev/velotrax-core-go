# velotrax-gateway-go — CLAUDE.md

> API Gateway cho hệ thống giả lập quản lý logistics.
> Nhận request từ Next.js Client Portal qua tRPC (JSON-over-HTTP), forward sang các downstream services qua gRPC.

---

## 1. Tổng quan kiến trúc

```
┌──────────────────────────────────┐
│   Next.js Client Portal          │  ← velotrax-portal  :3000
│   (App Router + tRPC client)     │
└──────────────┬───────────────────┘
               │ tRPC (JSON-over-HTTP POST)
               │ Authorization: Bearer <JWT>
               ▼
┌─────────────────────────────┐
│    velotrax-gateway-go      │  ← Project này  :8080
│    (Gin HTTP Gateway)       │
│                             │
│  /v1/auth/*       ──gRPC──► velotrax-auth-go   :50051  (Auth & User mgmt)
│  /v1/permissions/*──gRPC──► velotrax-auth-go   :50051
│  /v1/users/*      ──gRPC──► velotrax-auth-go   :50051
│  /v1/orders/*     ──gRPC──► velotrax-core-go   :50052  (Business logic)
└─────────────────────────────┘
```

**Nguyên tắc cốt lõi:**
- Gateway KHÔNG chứa business logic.
- Mọi giao tiếp downstream đều qua gRPC.
- Inbound từ portal là tRPC — gateway expose JSON-over-HTTP endpoint tương thích tRPC (`POST /v1/trpc/[procedure]`).
- Gateway chỉ làm: nhận request → xác thực JWT → transform → forward → trả response.

### tRPC Inbound Protocol
tRPC client từ Next.js gửi `POST` với body `{ "json": <input> }` và nhận `{ "result": { "data": { "json": <output> } } }`.
Gateway implement đúng envelope format này để Next.js không cần wrapper thêm.

Hai cách route song song được hỗ trợ:
| Style | Path ví dụ | Dùng cho |
|---|---|---|
| tRPC procedure | `POST /v1/trpc/auth.login` | Next.js portal gọi qua tRPC client |
| REST-style | `POST /v1/auth/login` | Tool test (curl, Postman), OpenAPI spec |

Cả hai style đều dùng chung handler — chỉ khác lớp routing ở ngoài.

---

## 2. Tech Stack

| Thành phần | Lựa chọn |
|---|---|
| Language | Go 1.25 |
| HTTP Framework | Gin |
| Downstream Protocol | gRPC |
| Inbound Protocol | tRPC JSON-over-HTTP (từ Next.js portal) |
| Auth | JWT (middleware tự xác thực, không forward sang auth-service) |
| Config | Viper + `.env` |
| Logging | Zap (structured JSON logs) |
| Validation | go-playground/validator |
| Proto codegen | protoc + protoc-gen-go + protoc-gen-go-grpc |
| Database Driver | MongoDB (mongo-driver/v2) |

---

## 3. Cấu trúc thư mục

```
velotrax-gateway-go/
├── cmd/
│   └── server/
│       └── main.go               # Entry point
├── internal/
│   ├── config/
│   │   └── config.go             # Load & validate env/config
│   ├── db/
│   │   └── mongo.go              # MongoDB connection, indexes
│   ├── middleware/
│   │   ├── auth.go               # JWT validation middleware
│   │   ├── logger.go             # Request logging
│   │   ├── recovery.go           # Panic recovery
│   │   └── cors.go               # CORS handler
│   ├── model/
│   │   ├── user.go               # User schema (portable)
│   │   └── order.go              # Order + Address schema (portable)
│   ├── handler/
│   │   ├── auth/
│   │   │   └── handler.go        # HTTP handlers cho /v1/auth/*
│   │   ├── permission/
│   │   │   └── handler.go        # HTTP handlers cho /v1/permissions/*
│   │   ├── user/
│   │   │   └── handler.go        # HTTP handlers cho /v1/users/*
│   │   └── order/
│   │       └── handler.go        # HTTP handlers cho /v1/orders/*
│   ├── grpc/
│   │   ├── client/
│   │   │   ├── auth_client.go    # gRPC client tới velotrax-auth-go
│   │   │   └── core_client.go    # gRPC client tới velotrax-core-go
│   │   └── interceptor/
│   │       └── auth.go           # Đính kèm JWT vào gRPC metadata
│   └── router/
│       └── router.go             # Đăng ký tất cả routes Gin
├── proto/
│   ├── auth/
│   │   └── auth.proto            # Contract với velotrax-auth-go
│   └── core/
│       └── core.proto            # Contract với velotrax-core-go
├── gen/                          # Code tự sinh từ proto (không chỉnh tay)
│   ├── auth/
│   └── core/
├── api/
│   └── v1/
│       └── openapi.yaml          # OpenAPI spec (source of truth cho client)
├── scripts/
│   └── gen_proto.sh              # Script chạy protoc
├── docker-compose.yml            # Docker stack (gateway + mongodb)
├── Dockerfile                    # Multi-stage build
├── .env                          # Environment vars (local dev)
├── .env.example                  # Template vars
├── go.mod / go.sum
├── Makefile
└── CLAUDE.md
```

---

## 4. API Routes — /v1

### 4.1 Auth  →  velotrax-auth-go

| Method | Path | Mô tả | Auth required |
|---|---|---|---|
| POST | `/v1/auth/register` | Đăng ký tài khoản mới | No |
| POST | `/v1/auth/login` | Đăng nhập, trả JWT | No |
| POST | `/v1/auth/logout` | Huỷ session / blacklist token | Yes |
| POST | `/v1/auth/refresh` | Làm mới access token | Yes (refresh token) |

### 4.2 Permissions  →  velotrax-auth-go

| Method | Path | Mô tả | Auth required |
|---|---|---|---|
| GET | `/v1/permissions` | Liệt kê tất cả permissions | Yes (admin) |
| POST | `/v1/permissions` | Tạo permission mới | Yes (admin) |
| PUT | `/v1/permissions/:id/assign` | Gán permission cho user | Yes (admin) |
| DELETE | `/v1/permissions/:id/revoke` | Thu hồi permission của user | Yes (admin) |

### 4.3 User Management (Active/Deactive)  →  velotrax-auth-go

| Method | Path | Mô tả | Auth required |
|---|---|---|---|
| GET | `/v1/users` | Danh sách users | Yes (admin) |
| GET | `/v1/users/:id` | Chi tiết một user | Yes |
| PUT | `/v1/users/:id/activate` | Kích hoạt user | Yes (admin) |
| PUT | `/v1/users/:id/deactivate` | Vô hiệu hoá user | Yes (admin) |

### 4.4 Orders  →  velotrax-core-go

| Method | Path | Mô tả | Auth required |
|---|---|---|---|
| GET | `/v1/orders` | Danh sách đơn hàng (có filter/paging) | Yes |
| GET | `/v1/orders/:id` | Chi tiết đơn hàng | Yes |
| GET | `/v1/orders/:id/tracking` | Lịch sử tracking đơn hàng | Yes |

---

## 5. Middleware Pipeline

Mọi request đi qua pipeline theo thứ tự:

```
Request
  → Logger        (ghi log mọi request, duration, status)
  → Recovery      (bắt panic, trả 500)
  → JWT Auth      (chỉ áp cho routes có tag "auth required")
  → Handler       (forward gRPC)
  → Response
```

### JWT Middleware (internal/middleware/auth.go)
- Đọc `Authorization: Bearer <token>` header.
- Xác thực chữ ký và hạn dùng cục bộ (không gọi gRPC để verify).
- Inject `userID`, `roles` vào Gin context (`c.Set`).
- Trả `401` nếu token không hợp lệ, `403` nếu thiếu quyền.

---

## 6. gRPC Client Strategy

### 6.1 Kết nối
- Mỗi downstream service có một dedicated gRPC client pool (singleton, khởi tạo lúc startup).
- Địa chỉ đọc từ config: `GRPC_AUTH_ADDR`, `GRPC_CORE_ADDR`.
- Dùng `grpc.WithBlock()` + timeout lúc startup để fail-fast nếu service chưa sẵn sàng.

### 6.2 Auth Interceptor
- Với các gRPC call cần xác thực, interceptor đính kèm JWT gốc từ client vào gRPC metadata:
  ```
  metadata: { "authorization": "Bearer <token>" }
  ```
- Downstream service tự xác thực token từ metadata.

### 6.3 Proto Contract
- File `.proto` đặt trong `proto/` là **source of truth**.
- Code sinh ra đặt trong `gen/` — **không chỉnh tay**.
- Chạy `make proto` để regenerate sau khi sửa `.proto`.

---

## 7. Config & Environment

### Port Map

| Service | Protocol | Port |
|---|---|---|
| velotrax-portal (Next.js) | HTTP | 3000 |
| velotrax-gateway-go (Gin) | HTTP | 8080 |
| velotrax-auth-go | gRPC | 50051 |
| velotrax-core-go | gRPC | 50052 |

### Environment Variables

```env
# .env / .env.example
APP_PORT=8080
APP_ENV=development          # development | production

# JWT
JWT_SECRET=...               # Phải >= 32 ký tự
JWT_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# gRPC downstream
GRPC_AUTH_ADDR=localhost:50051    # velotrax-auth-go  (docker: auth:50051)
GRPC_CORE_ADDR=localhost:50052    # velotrax-core-go  (docker: core:50052)

# Logging
LOG_LEVEL=info               # debug | info | warn | error

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000   # velotrax-portal

# MongoDB
MONGO_URI=mongodb://localhost:27017   # local: localhost; docker: mongodb:27017
```

Tất cả biến bắt buộc được validate tại startup — server không khởi động nếu thiếu.

---

## 8. Makefile Commands

```makefile
make run          # Chạy server dev
make build        # Build binary
make proto        # Regenerate gRPC code từ .proto files
make lint         # golangci-lint
make test         # go test ./...
make docker-build # Build Docker image
make docker-run   # docker compose up -d
make docker-down  # docker compose down
```

> Mỗi service (auth, core, portal) tự quản lý `docker-compose.yml` riêng trong repo của chúng.

---

## 9. AI Coding Rules

### 9.1 Tổng quát
- Không viết business logic trong gateway. Handler chỉ được: parse → validate → forward → respond.
- Không gọi gRPC trực tiếp từ handler. Luôn thông qua interface trong `internal/grpc/client/`.
- Mọi error từ gRPC downstream phải được map sang HTTP status code hợp lý (`codes.NotFound` → 404, v.v.).

### 9.2 Cấu trúc Handler
```go
// Pattern bắt buộc cho mọi handler
func (h *Handler) MethodName(c *gin.Context) {
    // 1. Bind & validate input
    // 2. Lấy context (userID, v.v.) từ Gin context
    // 3. Gọi gRPC client
    // 4. Map response → HTTP response
}
```

### 9.3 Error Handling
- Dùng một hàm helper `respondError(c, err)` duy nhất để map gRPC status codes → HTTP.
- Không bao giờ leak internal error message ra client ở môi trường production.
- Log full error (với trace ID) ở server, trả thông điệp generic cho client.

### 9.4 Naming
| Pattern | Convention |
|---|---|
| Files | `snake_case.go` |
| Types/Structs | `PascalCase` |
| Functions/Methods | `PascalCase` (exported), `camelCase` (internal) |
| Constants | `UPPER_SNAKE_CASE` |
| gRPC clients | `<Service>Client` (vd: `AuthClient`, `CoreClient`) |
| Handlers | `<Domain>Handler` (vd: `AuthHandler`, `OrderHandler`) |

### 9.5 Không làm
- Không cache dữ liệu trong gateway (cache thuộc về downstream).
- Không đọc/ghi database trực tiếp.
- Không import package giữa các `handler/` domain (không cross-import).
- Không bỏ qua lỗi gRPC — mọi error phải được xử lý hoặc log rõ ràng.

---

## 10. Definition of Done (per feature)

- [ ] `go build ./...` không lỗi, không warning.
- [ ] `golangci-lint` pass.
- [ ] Handler chỉ làm parse/validate/forward — không có business logic.
- [ ] gRPC error codes được map đúng sang HTTP status codes.
- [ ] JWT middleware bảo vệ đúng các route cần auth.
- [ ] Biến môi trường mới được thêm vào `.env.example`.
- [ ] Nếu thêm route mới: cập nhật `api/v1/openapi.yaml`.
- [ ] Nếu sửa `.proto`: chạy `make proto` và commit cả file generated.

---

## 11. Phase 1 — Scaffolding & Config (COMPLETED)

### Hoàn tất (2026-04-13)

#### 11.1 Config Loading & Validation
- ✅ Implement `internal/config/config.go`:
  - Load environment variables qua Viper
  - Validate required fields tại startup (JWT_SECRET >= 32 chars, GRPC_AUTH_ADDR, GRPC_CORE_ADDR)
  - Debug print config object trong validate function để xem giá trị thực

#### 11.2 Docker & Environment Setup
- ✅ Create `.env` từ `.env.example` với JWT_SECRET >= 32 ký tự
- ✅ Update `docker-compose.yml`:
  - Thêm `env_file: .env` để load từ file
  - Thêm `environment` section để explicitly pass vars vào container (Viper `AutomaticEnv()`)
- ✅ Rebuild Docker image và container tự động nhận config

#### 11.3 Main Entry Point Setup
- ✅ Implement `cmd/server/main.go`:
  - Load config → initialize logger (Zap)
  - Print config object cho debug
  - Connect gRPC clients (auth, core) tại startup
  - Setup Gin engine (debug/release mode based on APP_ENV)
  - Graceful shutdown với signal handler

#### 11.4 Dependencies Installed
- ✅ `go.mod` setup với main dependencies:
  - `github.com/gin-gonic/gin` (HTTP framework)
  - `go.uber.org/zap` (structured logging)
  - `github.com/spf13/viper` (config management)
  - `google.golang.org/grpc` (gRPC client)

**Status:** Config layer hoàn tất, server sẵn sàng chạy. Chờ gRPC services upstream (auth, core) ready hoặc mock.

---

## 12. Phase 2 — MongoDB & Data Models (COMPLETED)

### Hoàn tất (2026-04-13)

#### 12.1 MongoDB Setup
- ✅ Docker service `mongodb:8.0` trong `docker-compose.yml`
  - Health check để ensure ready state
  - Named volume `mongo_data` cho persistence
  - Gateway depends_on MongoDB trước startup
- ✅ Environment variable `MONGO_URI` trong `.env.example` và config
- ✅ `internal/config/config.go`: thêm `MongoURI` field, validate, BindEnv binding

#### 12.2 MongoDB Connection Layer
- ✅ Implement `internal/db/mongo.go`:
  - `Connect()`: kết nối với health check (ServerAPI v1, 10s timeout)
  - `Disconnect()`: graceful close
  - `EnsureIndexes()`: tạo indexes idempotent tại startup
  - Database name constant: `velotrax`

#### 12.3 Data Models (Portable)
- ✅ `internal/model/user.go`:
  - User struct: ID (ObjectID), UserName, PasswordHash, Active, Roles ([]string), OrderIDs ([]ObjectID), CreatedAt, UpdatedAt
  - Role constants: `SHIPPER`, `ADMIN`, `FREE_USER`
  - bson + json tags (json:"-" trên PasswordHash để tránh leak)
- ✅ `internal/model/order.go`:
  - Order struct: ID, UserID, Status, TrackingNumber, OriginAddress, DestinationAddress, EstimatedDelivery, WeightKg, Timestamps
  - OrderStatus enum: PENDING, CONFIRMED, IN_TRANSIT, OUT_FOR_DELIVERY, DELIVERED, CANCELLED
  - Nested Address struct: Street, City, Province, PostalCode, Country
  - Collection names: `CollectionUsers`, `CollectionOrders`

#### 12.4 Database Indexes
**Users Collection:**
- `idx_users_username_unique`: {userName: 1} — unique, để login lookup
- `idx_users_active`: {active: 1}
- `idx_users_roles`: {roles: 1}

**Orders Collection:**
- `idx_orders_user_id`: {user_id: 1}
- `idx_orders_tracking_unique`: {tracking_number: 1} — unique, sparse (null trước khi confirm)
- `idx_orders_status`: {status: 1}
- `idx_orders_user_status`: {user_id: 1, status: 1} — compound, hay dùng nhất
- `idx_orders_created_at`: {created_at: -1}

#### 12.5 Integration
- ✅ Wire `db.Connect()` trong `cmd/server/main.go` — chạy TRƯỚC gRPC clients
- ✅ Wire `db.EnsureIndexes()` lúc startup — tạo collections + indexes
- ✅ Graceful MongoDB disconnect khi shutdown (5s context timeout)

#### 12.6 Verification
```
✅ Config loads MONGO_URI từ environment
✅ MongoDB connects tới velotrax database
✅ Tất cả 9 indexes được tạo thành công
✅ internal/model/ portable: chỉ import bson + time (không internal imports)
```

**Status:** MongoDB layer sẵn sàng cho downstream services. Schemas copy-paste sang velotrax-auth-go và velotrax-core-go mà không cần sửa gì (chỉ đổi package name nếu cần).

---

## 13. Phase 3 — gRPC Clients & Middleware (NEXT)

### Objectives
- [ ] Implement gRPC client abstractions (AuthClient, CoreClient)
- [ ] Create JWT middleware xác thực local + inject context
- [ ] Create Logger middleware (structured JSON, trace IDs)
- [ ] Create Recovery middleware (panic handling)
- [ ] Setup router với middleware pipeline
- [ ] Wire all into main.go, test middleware order

**Timeline:** Phase 2 complete, Phase 3 ready to start.
