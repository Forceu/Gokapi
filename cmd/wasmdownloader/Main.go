//go:build js && wasm

package main

// includes code from Alessandro Segala (ItalyPaleAle)
// https://withblue.ink/2020/10/03/go-webassembly-http-requests-and-promises.html

import (
	"encoding/base64"
	"errors"
	"github.com/forceu/gokapi/internal/encryption"
	"io"
	"net/http"
	"syscall/js"
)

// Main routine that is called on startup
func main() {
	js.Global().Set("GokapiEncrypt", js.FuncOf(Encrypt))
	js.Global().Set("GokapiDecrypt", js.FuncOf(Decrypt))
	println("WASM Downloader module loaded")
	// Prevent the function from returning, which is required in a wasm module
	select {}
}

func Decrypt(this js.Value, args []js.Value) interface{} {
	key, url, err := getParams(args)
	if err != nil {
		return jsError(err.Error())
	}
	return encryptDecrypt(key, url, false)
}

func Encrypt(this js.Value, args []js.Value) interface{} {
	key, url, err := getParams(args)
	if err != nil {
		return jsError(err.Error())
	}
	return encryptDecrypt(key, url, true)
}

func encryptDecrypt(key []byte, url string, doEncrypt bool) interface{} {

	// Handler for the Promise
	// We need to return a Promise because HTTP requests are blocking in Go
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]

		// Run this code asynchronously
		go func() {
			// Make the HTTP request
			res, err := http.DefaultClient.Get(url)
			if err != nil {
				// Handle errors: reject the Promise if we have an error
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(err.Error())
				reject.Invoke(errorObject)
				return
			}
			// We're not calling res.Body.Close() here, because we are reading it asynchronously

			// Create the "underlyingSource" object for the ReadableStream constructor
			// See: https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream/ReadableStream
			underlyingSource := map[string]interface{}{
				// start method
				"start": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					// The first and only arg is the controller object
					controller := args[0]

					// Process the stream in yet another background goroutine,
					// because we can't block on a goroutine invoked by JS in Wasm
					// that is dealing with HTTP requests
					go func() {
						// Close the response body at the end of this method
						defer res.Body.Close()
						var reader io.Reader
						if doEncrypt {
							reader, err = encryption.GetEncryptReader(key, res.Body)
						} else {
							reader, err = encryption.GetDecryptReader(key, res.Body)
						}
						if err != nil {
							// Tell the controller we have an error
							errorConstructor := js.Global().Get("Error")
							errorObject := errorConstructor.New(err.Error())
							controller.Call("error", errorObject)
							return
						}
						// Read the entire stream and pass it to JavaScript
						for {
							// Read up to 1MB at a time
							buf := make([]byte, 1048576)
							n, err := reader.Read(buf)
							if err != nil && err != io.EOF {
								// Tell the controller we have an error
								// We're ignoring "EOF" however, which means the stream was done
								errorConstructor := js.Global().Get("Error")
								errorObject := errorConstructor.New(err.Error())
								controller.Call("error", errorObject)
								return
							}
							if n > 0 {
								// If we read anything, send it to JavaScript using the "enqueue" method on the controller
								// We need to convert it to a Uint8Array first
								arrayConstructor := js.Global().Get("Uint8Array")
								dataJS := arrayConstructor.New(n)
								js.CopyBytesToJS(dataJS, buf[0:n])
								controller.Call("enqueue", dataJS)
							}
							if err == io.EOF {
								// Stream is done, so call the "close" method on the controller
								controller.Call("close")
								return
							}
						}
					}()

					return nil
				}),
				// cancel method
				"cancel": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					// If the request is canceled, just close the body
					res.Body.Close()

					return nil
				}),
			}

			// Create a ReadableStream object from the underlyingSource object
			readableStreamConstructor := js.Global().Get("ReadableStream")
			readableStream := readableStreamConstructor.New(underlyingSource)

			// Create the init argument for the Response constructor
			// This allows us to pass a custom status code (and optionally headers and more)
			// See: https://developer.mozilla.org/en-US/docs/Web/API/Response/Response
			responseInitObj := map[string]interface{}{
				"status":     http.StatusOK,
				"statusText": http.StatusText(http.StatusOK),
			}

			// Create a Response object with the stream inside
			responseConstructor := js.Global().Get("Response")
			response := responseConstructor.New(readableStream, responseInitObj)

			// Resolve the Promise
			resolve.Invoke(response)
		}()

		// The handler of a Promise doesn't return any value
		return nil
	})

	// Create and return the Promise object
	// The Promise will resolve with a Response object
	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

func getParams(args []js.Value) ([]byte, string, error) {
	keyBase64 := args[0].String()
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, "", errors.New("invalid base64 provided")
	}
	if len(key) != 32 {
		return nil, "", errors.New("invalid cipher provided")
	}
	url := args[1].String()
	return key, url, nil
}

// Wraps a message into a JavaScript object of type error
func jsError(message string) js.Value {
	errConstructor := js.Global().Get("Error")
	errVal := errConstructor.New(message)
	return errVal
}
