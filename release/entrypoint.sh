#!/bin/bash

set -e


targets=${@-"darwin/amd64 linux/amd64 linux/386 linux/arm linux/arm64 windows/amd64 windows/386"}

cd /usr/src/myapp

for target in $targets; do
  os="$(echo $target | cut -d '/' -f1)"
  arch="$(echo $target | cut -d '/' -f2)"
  output="release/gokapi-${os}_${arch}"
  if [ $os = "windows" ]; then
    output+='.exe'
  fi

  echo "----> Building project for: $target"
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -o $output
  zip -j $output.zip $output > /dev/null
  rm $output
done

echo "----> Build is complete. List of files at $release_path:"
cd release/
ls -l gokapi-*
