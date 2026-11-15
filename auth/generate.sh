#!/usr/bin/env bash
# Regenerates pb/ from proto/.
#
# proto/ is a verbatim copy of auth/token.proto and signature/signature.proto
# from Cerebellum-Network/ddc-proto at the commit in proto/SOURCE. The proto
# packages stay `auth` and `signature`, so the messages keep their full names
# (auth.AuthToken, signature.Signature) and their wire format.
#
# Tool versions are pinned to the ones ddc-dac generates with, so the two
# outputs are interchangeable while ddc-dac moves onto this package:
#   protoc 29.3, protoc-gen-go v1.36.6, protoc-gen-go-vtproto v0.6.0
#
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
#   go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.6.0
set -euo pipefail
cd "$(dirname "$0")"

want_protoc="libprotoc 29.3"
want_go="protoc-gen-go v1.36.6"
[[ "$(protoc --version)" == "$want_protoc" ]] || { echo "need $want_protoc, have $(protoc --version)" >&2; exit 1; }
[[ "$(protoc-gen-go --version)" == "$want_go" ]] || { echo "need $want_go, have $(protoc-gen-go --version)" >&2; exit 1; }
command -v protoc-gen-go-vtproto >/dev/null || { echo "need protoc-gen-go-vtproto v0.6.0 on PATH" >&2; exit 1; }

pkg='github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb;pb'
files=(signature/signature.proto auth/token.proto)

opts=()
for f in "${files[@]}"; do
  opts+=("--go_opt=M$f=$pkg" "--go-vtproto_opt=M$f=$pkg")
done

rm -f pb/*.pb.go
protoc -I proto \
  --go_out=pb --go_opt=paths=source_relative \
  --go-vtproto_out=pb --go-vtproto_opt=paths=source_relative \
  --go-vtproto_opt=features=marshal+unmarshal+size \
  "${opts[@]}" \
  "${files[@]}"

# paths=source_relative mirrors proto/'s directories; the package is flat.
mv pb/auth/*.go pb/signature/*.go pb/
rmdir pb/auth pb/signature
