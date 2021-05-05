package test

import (
	"errors"
	"fmt"
	"log"
	"net/http"
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
	IsNotNil(mockT, nil)
	mockT.Check()
}

func TestHttpConfig(t *testing.T) {
	mockT := MockTest{reference: t}
	mockT.WantFail()
	test := HttpTestConfig{}
	test.init(mockT)
	mockT.Check()
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
	HttpPostRequest(t, HttpTestConfig{
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
	os.Remove("testfile")
}

func startTestServer() {
	http.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprint(writer, "TestContent\n")
		for _, cookie := range request.Cookies() {
			fmt.Fprint(writer, "cookie name: "+cookie.Name+" cookie value: "+cookie.Value+"\n")
		}
		request.ParseForm()
		if request.Form.Get("testPostKey") != "" {
			fmt.Fprint(writer, "testPostKey: "+request.Form.Get("testPostKey")+"\n")
		}
	})
	go func() { log.Fatal(http.ListenAndServe("127.0.0.1:9999", nil)) }()
	time.Sleep(2 * time.Second)
}
