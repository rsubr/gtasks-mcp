#!/bin/bash

set -euox pipefail

go mod tidy

export CGO_ENABLED=0
export GOOS=linux

GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/gtasks-mcp-${GOOS}-amd64 ./cmd/server
GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/gtasks-mcp-${GOOS}-arm64 ./cmd/server

