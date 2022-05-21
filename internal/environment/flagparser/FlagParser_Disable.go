//go:build test

package flagparser

// disableParsing disables parsing when running unit tests, as parsing is called in the test's init() function, which results in an error
var disableParsing = true
