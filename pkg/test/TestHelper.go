package test

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// IsEqualString fails test if got and want are not identical
func IsEqualString(t *testing.T, got, want string) {
	if got != want {
		t.Errorf("Assertion failed, got: %s, want: %s.", got, want)
	}
}

// IsEqualBool fails test if got and want are not identical
func IsEqualBool(t *testing.T, got, want bool) {
	if got != want {
		t.Errorf("Assertion failed, got: %t, want: %t.", got, want)
	}
}

// IsEqualInt fails test if got and want are not identical
func IsEqualInt(t *testing.T, got, want int) {
	if got != want {
		t.Errorf("Assertion failed, got: %d, want: %d.", got, want)
	}
}

// HttpPageResult tests if a http server is outputting the correct result
func HttpPageResult(t *testing.T, configuration HttpTestConfig) []*http.Cookie {
	configuration.init()
	client := &http.Client{}

	data := url.Values{}
	for _, value := range configuration.PostValues {
		data.Add(value.Key, value.Value)
	}

	req, err := http.NewRequest(configuration.Method, configuration.Url, strings.NewReader(data.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	for _, cookie := range configuration.Cookies {
		req.Header.Set("Cookie", cookie.toString())
	}
	if len(configuration.PostValues) > 0 {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	}
	resp, err := client.Do(req)

	if resp.StatusCode != 200 {
		t.Errorf("Status %d != 200", resp.StatusCode)
	}
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.IsHtml && !bytes.Contains(bs, []byte("</html>")) {
		t.Error(configuration.Url + ": Incorrect response")
	}
	if configuration.RequiredContent != "" && !bytes.Contains(bs, []byte(configuration.RequiredContent)) {
		t.Error(configuration.Url + ": Incorrect response. Got:\n" + string(bs))
	}
	resp.Body.Close()
	return resp.Cookies()
}

// HttpTestConfig is a struct for http test init
type HttpTestConfig struct {
	Url             string
	RequiredContent string
	IsHtml          bool
	Method          string
	PostValues      []PostBody
	Cookies         []Cookie
}

func (c *HttpTestConfig) init() {
	if c.Url == "" {
		log.Fatalln("No url passed!")
	}
	if c.Method == "" {
		c.Method = "GET"
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

type PostBody struct {
	Key   string
	Value string
}

func HttpPostRequest(t *testing.T, url, filename, fieldName, requiredText string, cookies []Cookie) {
	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filepath.Base(file.Name()))
	if err != nil {
		t.Fatal(err)
	}

	io.Copy(part, file)
	writer.Close()
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		t.Fatal(err)
	}

	for _, cookie := range cookies {
		request.Header.Set("Cookie", cookie.toString())
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if requiredText != "" && !bytes.Contains(content, []byte(requiredText)) {
		t.Error(url + ": Incorrect response. Got:\n" + string(content))
	}
}
