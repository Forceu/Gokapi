//go:build gogenerate

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	output, err := buildWasmModule("github.com/forceu/gokapi/cmd/wasmdownloader", "../../internal/webserver/web/main.wasm")
	if err != nil {
		fmt.Println("ERROR: Could not compile wasmdownloader")
		fmt.Println(err)
		fmt.Println(string(output))
		os.Exit(2)
	}
	fmt.Println("Compiled Downloader WASM module")
	output, err = buildWasmModule("github.com/forceu/gokapi/cmd/wasme2e", "../../internal/webserver/web/e2e.wasm")
	if err != nil {
		fmt.Println("ERROR: Could not compile wasme2e")
		fmt.Println(err)
		fmt.Println(string(output))
		os.Exit(3)
	}
	fmt.Println("Compiled E2E WASM module")
}

func buildWasmModule(src string, dst string) ([]byte, error) {
	cmd := exec.Command("go", "build", "-o", dst, src)
	cmd.Env = append(os.Environ(),
		"GOOS=js", "GOARCH=wasm")
	return cmd.CombinedOutput()
}
