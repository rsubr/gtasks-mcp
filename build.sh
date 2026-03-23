#!/bin/bash

go mod tidy
go build -o gtasks-mcp -ldflags="-s -w" ./cmd/server
#strip gtasks-mcp
