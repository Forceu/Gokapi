//  +build test

package test

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type MockT interface {
	Errorf(format string, args ...interface{})
	Helper()
}

// IsEqualString fails test if got and want are not identical
func IsEqualString(t MockT, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("Assertion failed, got: %s, want: %s.", got, want)
	}
}

// ResponseBodyContains fails test if http response does contain string
func ResponseBodyContains(t MockT, got *httptest.ResponseRecorder, want string) {
	t.Helper()
	result, _ := io.ReadAll(got.Result().Body)
	if !strings.Contains(string(result), want) {
		t.Errorf("Assertion failed, got: %s, want: %s.", got, want)
	}
}

// IsNotEqualString fails test if got and want are not identical
func IsNotEqualString(t MockT, got, want string) {
	t.Helper()
	if got == want {
		t.Errorf("Assertion failed, got: %s, want: not %s.", got, want)
	}
}

// IsEqualBool fails test if got and want are not identical
func IsEqualBool(t MockT, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("Assertion failed, got: %t, want: %t.", got, want)
	}
}

// IsEqualInt fails test if got and want are not identical
func IsEqualInt(t MockT, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("Assertion failed, got: %d, want: %d.", got, want)
	}
}

// IsNotEmpty fails test if string is empty
func IsNotEmpty(t MockT, s string) {
	t.Helper()
	if s == "" {
		t.Errorf("Assertion failed, got: %s, want: empty.", s)
	}
}

// IsEmpty fails test if string is not empty
func IsEmpty(t MockT, s string) {
	t.Helper()
	if s != "" {
		t.Errorf("Assertion failed, got: %s, want: empty.", s)
	}
}

// IsNil fails test if error not nil
func IsNil(t MockT, got error) {
	t.Helper()
	if got != nil {
		t.Errorf("Assertion failed, got: %s, want: nil.", got.(error).Error())
	}
}

// FileExists fails test a file does not exist
func FileExists(t MockT, name string) {
	t.Helper()
	if !fileExists(name) {
		t.Errorf("Assertion failed, file does not exist: %s, want: Exists.", name)
	}
}

// FileDoesNotExist fails test a file exists
func FileDoesNotExist(t MockT, name string) {
	t.Helper()
	if fileExists(name) {
		t.Errorf("Assertion failed, file exist: %s, want: Does not exist", name)
	}
}

// Copy of helper.FileExists, which cannot be used due to import cycle
func fileExists(name string) bool {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// IsNotNil fails test if error is nil
func IsNotNil(t MockT, got error) {
	t.Helper()
	if got == nil {
		t.Errorf("Assertion failed, got: nil, want: not nil.")
	}
}

// ExitCode returns a function to replace os.Exit()
func ExitCode(t MockT, want int) func(code int) {
	t.Helper()
	return func(code int) {
		IsEqualInt(t, code, want)
	}
}

// StartMockInputStdin simulates a user input on stdin. Call StopMockInputStdin afterwards!
func StartMockInputStdin(input string) *os.File {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	_, err = w.Write([]byte(input))
	if err != nil {
		panic(err)
	}
	w.Close()

	stdin := os.Stdin
	os.Stdin = r
	return stdin
}

// StopMockInputStdin needs to be called after StartMockInputStdin
func StopMockInputStdin(stdin *os.File) {
	os.Stdin = stdin
}

// HttpPageResult tests if a http server is outputting the correct result
func HttpPageResult(t MockT, config HttpTestConfig) []*http.Cookie {
	t.Helper()
	config.init(t)
	client := &http.Client{}

	data := url.Values{}
	for _, value := range config.PostValues {
		data.Add(value.Key, value.Value)
	}

	req, err := http.NewRequest(config.Method, config.Url, strings.NewReader(data.Encode()))
	IsNil(t, err)

	for _, cookie := range config.Cookies {
		req.Header.Set("Cookie", cookie.toString())
	}
	for _, header := range config.Headers {
		req.Header.Set(header.Name, header.Value)
	}
	if len(config.PostValues) > 0 {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	}
	resp, err := client.Do(req)
	IsNil(t, err)

	if resp.StatusCode != config.ResultCode {
		t.Errorf("Status %d != %d", config.ResultCode, resp.StatusCode)
	}
	content, err := ioutil.ReadAll(resp.Body)
	IsNil(t, err)
	if config.IsHtml && !bytes.Contains(content, []byte("</html>")) {
		t.Errorf(config.Url + ": Incorrect response")
	}
	for _, requiredString := range config.RequiredContent {
		if !bytes.Contains(content, []byte(requiredString)) {
			t.Errorf(config.Url + ": Incorrect response. Got:\n" + string(content))
		}
	}
	for _, excludedString := range config.ExcludedContent {
		if bytes.Contains(content, []byte(excludedString)) {
			t.Errorf(config.Url + ": Incorrect response. Got:\n" + string(content))
		}
	}
	resp.Body.Close()
	return resp.Cookies()
}

// HttpTestConfig is a struct for http test init
type HttpTestConfig struct {
	Url             string
	RequiredContent []string
	ExcludedContent []string
	IsHtml          bool
	Method          string
	PostValues      []PostBody
	Cookies         []Cookie
	Headers         []Header
	UploadFileName  string
	UploadFieldName string
	ResultCode      int
}

func (c *HttpTestConfig) init(t MockT) {
	t.Helper()
	if c.Url == "" {
		t.Errorf("No url passed!")
	}
	if c.Method == "" {
		c.Method = "GET"
	}
	if c.ResultCode == 0 {
		c.ResultCode = 200
	}
}

// Cookie is a simple struct to pass cookie values for testing
type Cookie struct {
	Name  string
	Value string
}

func (c *Cookie) toString() string {
	return c.Name + "=" + c.Value
}

// Header is a simple struct to pass header values for testing
type Header struct {
	Name  string
	Value string
}

// PostBody contains mock key/value post data
type PostBody struct {
	Key   string
	Value string
}

// HttpPostRequest sends a post request
func HttpPostRequest(t MockT, config HttpTestConfig) {
	t.Helper()
	file, err := os.Open(config.UploadFileName)
	IsNil(t, err)
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(config.UploadFieldName, filepath.Base(file.Name()))
	IsNil(t, err)

	io.Copy(part, file)
	writer.Close()
	request, err := http.NewRequest("POST", config.Url, body)
	IsNil(t, err)

	for _, cookie := range config.Cookies {
		request.Header.Set("Cookie", cookie.toString())
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}

	response, err := client.Do(request)
	IsNil(t, err)
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	IsNil(t, err)

	for _, requiredString := range config.RequiredContent {
		if !bytes.Contains(content, []byte(requiredString)) {
			t.Errorf(config.Url + ": Incorrect response. Got:\n" + string(content))
		}
	}
	for _, excludedString := range config.ExcludedContent {
		if bytes.Contains(content, []byte(excludedString)) {
			t.Errorf(config.Url + ": Incorrect response. Got:\n" + string(content))
		}
	}
}
