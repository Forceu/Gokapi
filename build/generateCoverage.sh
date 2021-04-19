#!/bin/sh
cd ..
go test ./... -coverprofile=/tmp/coverage.out --tags=test && go tool cover -html=/tmp/coverage.out
