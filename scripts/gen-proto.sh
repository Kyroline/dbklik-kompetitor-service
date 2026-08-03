#!/usr/bin/env bash
# Regenerate every module's gRPC code. Requires protoc plus the Go plugins:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
set -euo pipefail

cd "$(dirname "$0")/.."

for proto_dir in modules/*/presentation/grpc/proto; do
  module_dir="$(dirname "$proto_dir")"
  mkdir -p "$module_dir/pb"

  for proto in "$proto_dir"/*.proto; do
    echo "generating $proto"
    protoc \
      --proto_path="$proto_dir" \
      --go_out="$module_dir/pb" --go_opt=paths=source_relative \
      --go-grpc_out="$module_dir/pb" --go-grpc_opt=paths=source_relative \
      "$(basename "$proto")"
  done
done
