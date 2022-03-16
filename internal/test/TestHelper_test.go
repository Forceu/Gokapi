//go:build test

package test

import (
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var (
	wantFail bool
	isFailed = false
)

type MockTest struct {
	reference *testing.T
}

func (t MockTest) Errorf(format string, args ...interface{}) {
	isFailed = true
}
func (t MockTest) Helper() {
}

func (t *MockTest) WantFail() {
	t.Check()
	isFailed = false
	wantFail = true
}
func (t *MockTest) WantNoFail() {
	t.Check()
	isFailed = false
	wantFail = false
}

func (t *MockTest) Check() {
	if wantFail != isFailed {
		t.reference.Helper()
		t.reference.Error("Test failed")
	}
}

func TestFunctions(t *testing.T) {
	mockT := MockTest{reference: t}
	mockT.WantNoFail()
	IsEqualString(mockT, "test", "test")
	mockT.WantNoFail()
	IsNotEqualString(mockT, "test", "test2")
	mockT.WantNoFail()
	IsEqualBool(mockT, true, true)
	mockT.WantNoFail()
	IsEqualInt(mockT, 1, 1)
	mockT.WantNoFail()
	IsNotEmpty(mockT, "notEmpty")
	mockT.WantNoFail()
	IsEmpty(mockT, "")
	mockT.WantNoFail()
	IsNil(mockT, nil)
	mockT.WantNoFail()
	FileDoesNotExist(mockT, "testfile")
	os.WriteFile("testfile", []byte("content"), 0777)
	mockT.WantNoFail()
	FileExists(mockT, "testfile")

	mockT.WantNoFail()
	IsNotNil(mockT, errors.New("hello"))
	mockT.WantFail()
	IsEqualString(mockT, "test", "test2")
	mockT.WantFail()
	IsNotEqualString(mockT, "test", "test")
	mockT.WantFail()
	IsEqualBool(mockT, true, false)
	mockT.WantFail()
	IsEqualInt(mockT, 1, 2)
	mockT.WantFail()
	IsNotEmpty(mockT, "")
	mockT.WantFail()
	IsEmpty(mockT, "notEmpty")
	mockT.WantFail()
	IsNil(mockT, errors.New("hello"))
	mockT.WantFail()
	IsNilWithMessage(mockT, errors.New("hello"), "there")
	mockT.WantFail()
	IsNotNil(mockT, nil)
	mockT.WantFail()
	IsNotNilWithMessage(mockT, nil, "hello")
	mockT.WantFail()
	FileDoesNotExist(mockT, "testfile")
	os.Remove("testfile")
	mockT.WantFail()
	FileExists(mockT, "testfile")
	mockT.Check()
}

func TestHttpConfig(t *testing.T) {
	mockT := MockTest{reference: t}
	mockT.WantFail()
	test := HttpTestConfig{}
	test.init(mockT)
	mockT.Check()
}

func TestMockInputStdin(t *testing.T) {
	original := StartMockInputStdin("test input")
	result := helper.ReadLine()
	StopMockInputStdin(original)
	IsEqualString(t, result, "test input")
}

func TestHttpPageResult(t *testing.T) {
	startTestServer()
	HttpPageResult(t, HttpTestConfig{
		Url:             "http://127.0.0.1:9999/test",
		RequiredContent: []string{"TestContent", "testName", "testValue", "testPostKey", "testPostValue"},
		ExcludedContent: []string{"invalid"},
		PostValues: []PostBody{{
			Key:   "testPostKey",
			Value: "testPostValue",
		}},
		Cookies: []Cookie{{
			Name:  "testName",
			Value: "testValue",
		}},
		Headers: []Header{{
			Name:  "testHeader",
			Value: "value",
		}},
		Method: "POST",
	})
	mockT := MockTest{reference: t}
	mockT.WantFail()
	HttpPageResult(mockT, HttpTestConfig{
		Url: "http://127.0.0.1:9999/invalid",
	})
	mockT.WantFail()
	HttpPageResult(mockT, HttpTestConfig{
		Url:             "http://127.0.0.1:9999/test",
		RequiredContent: []string{"invalid"},
	})
	mockT.WantFail()
	HttpPageResult(mockT, HttpTestConfig{
		Url:             "http://127.0.0.1:9999/test",
		ExcludedContent: []string{"TestContent"},
	})
	mockT.WantFail()
	HttpPageResult(mockT, HttpTestConfig{
		Url:    "http://127.0.0.1:9999/test",
		IsHtml: true,
	})
	mockT.Check()
}

func TestHttpPostRequest(t *testing.T) {
	os.WriteFile("testfile", []byte("Testbytes"), 0777)
	HttpPostUploadRequest(t, HttpTestConfig{
		Url:             "http://127.0.0.1:9999/test",
		UploadFileName:  "testfile",
		UploadFieldName: "file",
		RequiredContent: []string{"TestContent", "testName", "testValue"},
		ExcludedContent: []string{"invalid"},
		Cookies: []Cookie{{
			Name:  "testName",
			Value: "testValue",
		}},
	})
	mockT := MockTest{reference: t}
	mockT.WantFail()
	HttpPostUploadRequest(mockT, HttpTestConfig{
		Url:             "http://127.0.0.1:9999/test",
		UploadFileName:  "testfile",
		UploadFieldName: "file",
		ExcludedContent: []string{"TestContent"}},
	)
	mockT.WantFail()
	HttpPostUploadRequest(mockT, HttpTestConfig{
		Url:             "http://127.0.0.1:9999/test",
		UploadFileName:  "testfile",
		UploadFieldName: "file",
		RequiredContent: []string{"invalid"}},
	)
	mockT.Check()
	os.Remove("testfile")
}

func TestResponseBodyContains(t *testing.T) {
	mockT := MockTest{reference: t}
	mockT.WantNoFail()
	w := httptest.NewRecorder()
	_, _ = io.WriteString(w, "TestContentWrite")
	ResponseBodyContains(mockT, w, "TestContentWrite")
	mockT.WantFail()
	w = httptest.NewRecorder()
	_, _ = io.WriteString(w, "TestContentWrite")
	ResponseBodyContains(mockT, w, "invalid")
	mockT.Check()
}

func startTestServer() {
	http.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		io.WriteString(writer, "TestContent\n")
		for _, cookie := range request.Cookies() {
			io.WriteString(writer, "cookie name: "+cookie.Name+" cookie value: "+cookie.Value+"\n")
		}
		request.ParseForm()
		if request.Form.Get("testPostKey") != "" {
			io.WriteString(writer, "testPostKey: "+request.Form.Get("testPostKey")+"\n")
		}
	})
	go func() { log.Fatal(http.ListenAndServe("127.0.0.1:9999", nil)) }()
	time.Sleep(2 * time.Second)
}

func TestOsExit(t *testing.T) {
	mockT := MockTest{reference: t}
	mockT.WantNoFail()
	osExit := ExitCode(mockT, 0)
	osExit(0)
	mockT.WantFail()
	osExit(1)
	mockT.Check()
}

func TestExpectPanic(t *testing.T) {
	mockT := MockTest{reference: t}
	mockT.WantFail()
	ExpectPanic(mockT)
	mockT.Check()
	err := errors.New("test")
	defer ExpectPanic(mockT)
	panic(err)
}

func TestGetRecorder(t *testing.T) {
	_, r := GetRecorder("GET", "/", []Cookie{{
		Name:  "test",
		Value: "testvalue",
	}}, []Header{{
		Name:  "testheader",
		Value: "testheadervalue",
	}}, nil)

	cookie, err := r.Cookie("test")
	IsNil(t, err)
	IsEqualString(t, cookie.Value, "testvalue")
	IsEqualString(t, r.Header.Get("testheader"), "testheadervalue")
}

func TestCompletesWithinTime(t *testing.T) {
	mockT := MockTest{reference: t}
	mockT.WantFail()
	CompletesWithinTime(mockT, delay100ms, 50*time.Millisecond)
	mockT.WantNoFail()
	CompletesWithinTime(mockT, delay100ms, 150*time.Millisecond)
	mockT.Check()
}

func delay100ms() {
	time.Sleep(100 * time.Millisecond)
}
