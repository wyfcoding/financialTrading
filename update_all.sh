#!/bin/bash
set -e

# 获取 go.mod 所有 module 名字
modules=$(go list -m -f '{{if not .Main}}{{.Path}}{{end}}' all)

for m in $modules; do
  echo "Updating $m to latest..."
  go get $m@latest
done

echo "Running go mod tidy..."
go mod tidy
