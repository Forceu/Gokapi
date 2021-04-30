package test

import (
	"errors"
	"testing"
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
