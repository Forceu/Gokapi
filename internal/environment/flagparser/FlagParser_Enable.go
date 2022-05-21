//go:build !test

package flagparser

// DisableParsing disables parsing when running unit tests, as parsing is called in the test's init() function, which results in an error
var DisableParsing = false
