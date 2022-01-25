#!/bin/bash

set -e

targets=${@-"darwin/amd64 linux/amd64 linux/386 linux/arm linux/arm64 windows/amd64 windows/386"}

cd /usr/src/myapp
go generate ./...

for target in $targets; do
	for tag in "full" "noaws"; do
		os="$(echo $target | cut -d '/' -f1)"
		arch="$(echo $target | cut -d '/' -f2)"
		output="build/gokapi_${tag}-${os}_${arch}"
		if [ $os = "windows" ]; then
			output+='.exe'
		fi

		echo "----> Building Gokapi ($tag) for $target"
		GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -tags $tag -ldflags="-s -w -X 'github.com/forceu/gokapi/internal/environment.Builder=Github Release Builder' -X 'github.com/forceu/gokapi/internal/environment.BuildTime=$(date)'" -o $output github.com/forceu/gokapi/cmd/gokapi
		zip -j $output.zip $output >/dev/null
		rm $output
	done
done

echo "----> Build is complete. List of files at $release_path:"
cd build/
ls -l gokapi_*
