#!/bin/sh
cd ..
go test ./... -parallel 8 --tags=test,integration,noaws -coverprofile=/tmp/coverage1.out
go test ./... -parallel 8 -coverprofile=/tmp/coverage2.out --tags=test,awsmock

which gocovmerge > /dev/null
if [ $? -eq 0 ]; then
   gocovmerge /tmp/coverage1.out /tmp/coverage2.out > /tmp/coverage.out
   go tool cover -html=/tmp/coverage.out
else
   go tool cover -html=/tmp/coverage2.out
fi
