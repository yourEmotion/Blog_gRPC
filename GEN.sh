#!/bin/bash

PROTO_DIR="api/proto"
THIRD_PARTY_DIR="third_party"
GO_OUT_DIR="api/proto"
OPENAPI_OUT="api/swagger"

echo "Creating directories..."
mkdir -p $PROTO_DIR
mkdir -p $OPENAPI_OUT

echo "Cleaning old files..."
rm -f $GO_OUT_DIR/*.pb.go
rm -f $GO_OUT_DIR/*_grpc.pb.go
rm -f $GO_OUT_DIR/*_gw.pb.go
rm -f $OPENAPI_OUT/*.swagger.json

echo "Generating Go code from proto..."
export PATH=$PATH:$(go env GOPATH)/bin

protoc -I $PROTO_DIR -I $THIRD_PARTY_DIR \
  --go_out=$GO_OUT_DIR --go-grpc_out=$GO_OUT_DIR \
  --grpc-gateway_out=$GO_OUT_DIR --openapiv2_out=$OPENAPI_OUT \
  $PROTO_DIR/blog.proto
