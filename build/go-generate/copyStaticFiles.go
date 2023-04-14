//go:build gogenerate

package main

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
)

func main() {
	copyFile(build.Default.GOROOT+"/misc/wasm/wasm_exec.js", "../../internal/webserver/web/static/js/wasm_exec.js")
	copyFile("../../go.mod", "../../build/go.mod")
	copyFile("../../openapi.json", "../../internal/webserver/web/static/apidocumentation/openapi.json")
}

// copyFile should only be used for small files
func copyFile(src string, dst string) {
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Println("ERROR: Cannot read " + src)
		fmt.Println(err)
		os.Exit(1)
	}
	err = os.WriteFile(dst, data, 0644)
	if err != nil {
		fmt.Println("ERROR: Cannot write " + dst)
		fmt.Println(err)
		os.Exit(2)
	}
	filename := filepath.Base(src)
	fmt.Println("Copied " + filename)
}
