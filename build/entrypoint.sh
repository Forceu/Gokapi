#!/bin/bash

set -e


targets=${@-"darwin/amd64 linux/amd64 linux/386 linux/arm linux/arm64 windows/amd64 windows/386"}

cd /usr/src/myapp
go generate Gokapi/cmd/gokapi

for target in $targets; do
  os="$(echo $target | cut -d '/' -f1)"
  arch="$(echo $target | cut -d '/' -f2)"
  output="build/gokapi-${os}_${arch}"
  if [ $os = "windows" ]; then
    output+='.exe'
  fi

  echo "----> Building project for: $target"
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -ldflags="-s -w -X 'Gokapi/internal/environment.Builder=Github Release Builder' -X 'Gokapi/internal/environment.BuildTime=$(date)'" -o $output Gokapi/cmd/gokapi
  zip -j $output.zip $output > /dev/null
  rm $output
done

echo "----> Build is complete. List of files at $release_path:"
cd build/
ls -l gokapi-*
