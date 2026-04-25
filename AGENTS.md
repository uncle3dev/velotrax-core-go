# AGENTS.md

Tài liệu ngắn cho người làm tiếp theo trong repo `velotrax-core-go`.

## Quy ước chung

- Ưu tiên bám theo code hiện tại, không suy diễn thêm tính năng.
- Không sửa file generated trừ khi task yêu cầu rõ ràng.
- Sau khi đổi code Go, chạy `gofmt` nếu cần.
- Nếu đổi protobuf, phải regenerate lại code và kiểm tra diff.

## Nguồn Sự Thật

- `proto/order/order.proto`: contract gRPC
- `internal/service/order/service.go`: logic `OrderService`
- `internal/config/config.go`: biến môi trường và validate config
- `internal/db/mongo.go`: connect MongoDB và index
- `cmd/server/main.go`: entrypoint thực tế

## Nên Sửa Khi Làm Tính Năng

- `internal/service/order/service.go`: đổi hành vi RPC
- `proto/order/order.proto`: đổi request/response contract
- `internal/model/*.go`: đổi schema dùng chung
- `internal/config/config.go` và `.env.example`: thêm/sửa biến môi trường
- `internal/db/mongo.go`: đổi collection/index khi liên quan data layer

## Không Nên Đụng Tay

- `internal/gen/order/*`: file sinh từ protobuf
- `bin/`: output build
- `.env` nếu là file local của máy dev

## Luồng Auth / Data / Route

- Auth hiện nằm trong `internal/service/order/service.go` qua JWT ở metadata gRPC `authorization: Bearer <token>`.
- `ListOrders`, `GetOrder`, `GetOrderTracking` hiện query MongoDB; `ADMIN` xem được mọi order, user thường chỉ xem order của chính mình.
- Mock data hiện được nạp qua startup seeder trong `internal/db/seed.go`, theo version marker để skip nếu đã chạy.
- HTTP router trong `internal/router/router.go` chỉ là scaffold; `main.go` hiện chỉ start gRPC server.

## Khi Đổi Proto

1. Sửa `proto/order/order.proto`
2. Chạy `make proto`
3. Kiểm tra lại `internal/gen/order/`
4. Đồng bộ README nếu contract public thay đổi

## Khi Đổi Seed/Mock Data

- `internal/db/seed.go` là nơi seed mock data startup hiện tại
- Seed là idempotent theo version marker, nên tăng version khi thay bộ dữ liệu mẫu
- Đồng bộ README nếu số lượng/mẫu dữ liệu public thay đổi

## Khi Đổi Config

- Update `internal/config/config.go`
- Update `.env.example`
- Update `README.md` nếu biến đó ảnh hưởng cách chạy
- Nhớ giữ `JWT_SECRET` dài tối thiểu 32 ký tự, và `MONGO_URI` phải có thật
