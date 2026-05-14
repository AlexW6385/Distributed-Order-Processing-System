#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export PATH="$(go env GOPATH)/bin:${PATH}"

protoc \
  --proto_path="${ROOT_DIR}/proto" \
  --go_out="${ROOT_DIR}/gen" \
  --go_opt=paths=source_relative \
  --go-grpc_out="${ROOT_DIR}/gen" \
  --go-grpc_opt=paths=source_relative \
  "${ROOT_DIR}/proto/product/v1/product.proto" \
  "${ROOT_DIR}/proto/payment/v1/payment.proto"
