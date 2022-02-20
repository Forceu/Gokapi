package main

import (
	"errors"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/models"
	"net/http"
	"os"
	"strconv"
	"syscall/js"
)

const version = "1.0"

// Main routine that is called on startup
func main() {
	// Export a "Gokapi" global object that contains our functions
	js.Global().Set("GokapiTest", js.FuncOf(Test))
	// Prevent the function from returning, which is required in a wasm module
	select {}
}

func Test(this js.Value, args []js.Value) interface{} {
	message := args[0].String() // get the parameters
	println(message)
	return nil
}

func Decrypt(this js.Value, args []js.Value) interface{} {

	key := bytesFromJs(args[0])
	if len(key) != 32 {
		return jsError("Invalid cipher provided")
	}
	encryption.InitWithCipher(key)

	req := args[1]

	// Ensure req is a Request object
	requestConstructor := js.Global().Get("Request")
	if req.Type() != js.TypeObject || !req.InstanceOf(requestConstructor) {
		return jsError("Invalid type for req argument")
	}
	// URL for the request
	reqUrlVal := req.Get("url")
	if reqUrlVal.Type() != js.TypeString {
		return jsError("Empty or invalid URL from the request")
	}
	var reqUrlStr = reqUrlVal.String()

	// Return a Promise
	// This is because HTTP request needs to be made in a separate goroutine: https://github.com/golang/go/issues/41310

	return nil
}

func Encrypt(this js.Value, args []js.Value) interface{} {
	resp, err := http.Get("test")
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	// Check server response
	if resp.StatusCode != http.StatusOK {
		return errors.New("bad status: " + strconv.Itoa(resp.StatusCode))
	}

	out, err := os.Create("output.txt")
	encryption.DecryptReader(models.EncryptionInfo{}, resp.Body, out)
	return nil
}

// Wraps a message into a JavaScript object of type error
func jsError(message string) js.Value {
	errConstructor := js.Global().Get("Error")
	errVal := errConstructor.New(message)
	return errVal
}

// Returns a byte slice from a js.Value
func bytesFromJs(arg js.Value) []byte {
	out := make([]byte, arg.Length())
	js.CopyBytesToGo(out, arg)
	return out
}

// Returns a js.Value from a byte slice
func jsFromBytes(data []byte) js.Value {
	arrayConstructor := js.Global().Get("Uint8Array")
	result := arrayConstructor.New(len(data))
	js.CopyBytesToJS(result, data)
	return result
}
