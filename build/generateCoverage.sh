#!/bin/sh
cd ..
go test ./... -parallel 8 --tags=test,integration,noaws -coverprofile=/tmp/coverage1.out
GOKAPI_AWS_BUCKET="gokapi" GOKAPI_AWS_REGION="eu-central-1" GOKAPI_AWS_KEY="keyid" GOKAPI_AWS_KEY_SECRET="secret" go test ./... -parallel 8 -coverprofile=/tmp/coverage2.out --tags=test,awstest

which gocovmerge > /dev/null
if [ $? -eq 0 ]; then
   gocovmerge /tmp/coverage1.out /tmp/coverage2.out > /tmp/coverage.out
   go tool cover -html=/tmp/coverage.out
else
   go tool cover -html=/tmp/coverage2.out
fi
