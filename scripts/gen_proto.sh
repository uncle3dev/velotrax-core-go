#!/bin/bash
set -e

echo "Generating protobuf code..."

PROTO_DIR="proto"
OUT_DIR="."
GOBIN=$(go env GOBIN)
if [ -z "$GOBIN" ]; then
  GOBIN=$(go env GOPATH)/bin
fi

mkdir -p internal/gen

# Generate Go types
protoc \
  --go_out="$OUT_DIR" \
  --go_opt=paths=source_relative \
  --go-grpc_out="$OUT_DIR" \
  --go-grpc_opt=paths=source_relative \
  --plugin=protoc-gen-go="$GOBIN/protoc-gen-go" \
  --plugin=protoc-gen-go-grpc="$GOBIN/protoc-gen-go-grpc" \
  "$PROTO_DIR/order/order.proto"

# Move to internal/gen
mkdir -p internal/gen/order
mv proto/order/order.pb.go proto/order/order_grpc.pb.go internal/gen/order/ 2>/dev/null || true
rm -rf proto/order/__pycache__ proto/__pycache__ 2>/dev/null || true

echo "✓ Protobuf code generated in internal/gen/"
