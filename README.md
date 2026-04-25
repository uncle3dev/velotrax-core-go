# velotrax-core-go

`velotrax-core-go` là service lõi cho phần đơn hàng của hệ thống Velotrax. Repo này hiện chạy một gRPC server `OrderService` trên cổng `50052`, khởi tạo kết nối MongoDB, và tự tạo index khi khởi động.

Hiện tại các RPC trong service vẫn là dữ liệu demo/stub:
- `ListOrders`
- `GetOrder`
- `GetOrderTracking`

## Tech Stack

- Go 1.25
- gRPC + protobuf
- MongoDB Go Driver v2
- Viper để đọc biến môi trường
- Zap để log
- JWT để xác thực token từ metadata gRPC
- Gin middleware/router có sẵn trong `internal/router` và `internal/middleware`, nhưng entrypoint hiện chưa wire HTTP server vào

## Cấu Trúc Chính

- `cmd/server/main.go`: entrypoint, load config, connect MongoDB, ensure indexes, start gRPC server
- `internal/config/config.go`: đọc và validate environment variables
- `internal/db/mongo.go`: kết nối MongoDB và tạo index
- `internal/service/order/service.go`: implement `OrderService`
- `internal/model/`: schema MongoDB dùng chung
- `internal/gen/order/`: code sinh từ protobuf, không sửa tay
- `proto/order/order.proto`: source of truth cho contract gRPC
- `scripts/gen_proto.sh`: script generate lại code protobuf
- `internal/router/router.go`: scaffold HTTP router, hiện chưa được gọi từ `main.go`

## Chạy Local

1. Tạo hoặc cập nhật biến môi trường theo `.env.example`.
2. Đảm bảo MongoDB đang chạy và `MONGO_URI` trỏ đúng tới database.
3. Chạy service:
```bash
make run
```

Hoặc build binary:
```bash
make build
```

## Biến Môi Trường

Các biến mà code hiện đọc:

- `GRPC_PORT`: mặc định `50052`
- `APP_ENV`: mặc định `development`
- `JWT_SECRET`: tối thiểu 32 ký tự; code có default placeholder nhưng nên set giá trị thật
- `LOG_LEVEL`: mặc định `info`
- `CORS_ALLOWED_ORIGINS`: có trong config, mặc định `http://localhost:3000`
- `MONGO_URI`: bắt buộc

Lưu ý:
- `config.Load()` đọc từ environment variables, không tự parse file `.env`
- `.env.example` chỉ là mẫu để dev set môi trường nhanh hơn

## Scripts và Lệnh Hỗ Trợ

Từ `Makefile`:

- `make run`: chạy server dev
- `make build`: build binary vào `./bin/server`
- `make proto`: regenerate protobuf code
- `make lint`: chạy `golangci-lint`
- `make test`: chạy `go test -v ./...`
- `make docker-build`: build Docker image
- `make docker-run`: `docker compose up -d`
- `make docker-down`: `docker compose down`

## Proto và Codegen

Contract gRPC nằm ở `proto/order/order.proto`. Sau khi sửa file này, chạy:

```bash
make proto
```

Code sinh ra sẽ được đặt trong `internal/gen/order/`. Không chỉnh tay các file generated.

## Docker

`Dockerfile` build binary Go và expose cổng `50052`.

`docker-compose.yml` hiện chỉ khai báo service `core` và dùng external network `velotrax-net`. Nếu chạy bằng Docker Compose, cần đảm bảo network này đã tồn tại.

## Lưu Ý Quan Trọng

- Service hiện xác thực JWT ngay trong `internal/service/order/service.go` bằng metadata `authorization: Bearer <token>`.
- `ListOrders` hiện trả dữ liệu hard-code, không query DB.
- `GetOrder` và `GetOrderTracking` cũng đang là dữ liệu mẫu.
- HTTP `/health` có scaffold trong `internal/router/router.go`, nhưng chưa được gắn vào entrypoint hiện tại.
