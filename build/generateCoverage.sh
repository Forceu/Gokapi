#!/bin/sh
cd ..
go test ./... -parallel 8 -coverprofile=/tmp/coverage.out --tags=test,awsmock && go tool cover -html=/tmp/coverage.out
