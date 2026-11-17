#!/usr/bin/env bash
# Regenerates the fileapi, dagapi, cnsapi and activity packages from proto/.
#
# proto/ holds verbatim copies of the DDC storage node's gRPC API
# (api/{file,dag,cns}) and of activity/record.proto, which the file API imports;
# see proto/SOURCE. auth/token.proto and signature/signature.proto come from the
# auth module. Proto packages, file paths and descriptors are unchanged, so the
# wire format and the gRPC method names are the storage node's.
#
# Tool versions are pinned to the ones the storage node and ddc-dac generate with:
#   protoc 29.3, protoc-gen-go v1.36.6, protoc-gen-go-grpc v1.5.1,
#   protoc-gen-go-vtproto v0.6.0
set -euo pipefail
cd "$(dirname "$0")"

[[ "$(protoc --version)" == "libprotoc 29.3" ]] || { echo "need protoc 29.3" >&2; exit 1; }
[[ "$(protoc-gen-go --version)" == "protoc-gen-go v1.36.6" ]] || { echo "need protoc-gen-go v1.36.6" >&2; exit 1; }
[[ "$(protoc-gen-go-grpc --version)" == "protoc-gen-go-grpc 1.5.1" ]] || { echo "need protoc-gen-go-grpc 1.5.1" >&2; exit 1; }
command -v protoc-gen-go-vtproto >/dev/null || { echo "need protoc-gen-go-vtproto v0.6.0" >&2; exit 1; }

mod=github.com/cerebellum-network/cere-ddc-sdk-go
mappings=(
  "signature/signature.proto=$mod/auth/pb;pb"
  "auth/token.proto=$mod/auth/pb;pb"
  "activity/record.proto=$mod/storage/activity;activity"
  "api/file/file_api.proto=$mod/storage/fileapi;fileapi"
  "api/dag/dag_api.proto=$mod/storage/dagapi;dagapi"
  "api/cns/cns_api.proto=$mod/storage/cnsapi;cnsapi"
)
go_m=() grpc_m=() vt_m=()
for m in "${mappings[@]}"; do
  go_m+=("--go_opt=M$m"); grpc_m+=("--go-grpc_opt=M$m"); vt_m+=("--go-vtproto_opt=M$m")
done

rm -f activity/*.pb.go fileapi/*.pb.go dagapi/*.pb.go cnsapi/*.pb.go

# The DAC record messages, with vtproto like ddc-dac generates them.
protoc -I proto -I ../auth/proto \
  --go_out=. --go_opt=module=$mod/storage "${go_m[@]}" \
  --go-vtproto_out=. --go-vtproto_opt=module=$mod/storage "${vt_m[@]}" \
  --go-vtproto_opt=features=marshal+unmarshal+size \
  activity/record.proto

# The storage node's gRPC API.
protoc -I proto -I ../auth/proto \
  --go_out=. --go_opt=module=$mod/storage "${go_m[@]}" \
  --go-grpc_out=. --go-grpc_opt=module=$mod/storage "${grpc_m[@]}" \
  api/file/file_api.proto api/dag/dag_api.proto api/cns/cns_api.proto
