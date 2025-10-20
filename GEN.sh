#!/bin/bash

echo "Creating directories..."
mkdir -p api/go
mkdir -p api/swagger

echo "Cleaning old files..."
rm -f api/go/*.pb.go
rm -f api/go/*_grpc.pb.go
rm -f api/go/*_gw.pb.go
rm -f api/swagger/*.swagger.json

echo "Generating Go code from proto..."
export PATH=$PATH:$(go env GOPATH)/bin

protoc -I api/proto -I third_party \
  --go_out=api/go --go-grpc_out=api/go \
  --grpc-gateway_out=api/go --openapiv2_out=api/swagger \
  api/proto/blog.proto
