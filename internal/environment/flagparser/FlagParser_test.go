package flagparser

import (
	"bytes"
	"flag"
	"github.com/forceu/gokapi/internal/test"
	"io"
	"os"
	"testing"
)

func TestParseFlags(t *testing.T) {
	DisableParsing = true
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = append([]string{os.Args[0]}, "--version")
	flags := ParseFlags()
	test.IsEqualBool(t, flags.ShowVersion, false)

	DisableParsing = false

	tests := []struct {
		name      string
		args      []string
		assertion func(flags MainFlags)
	}{
		{
			name: "ShowVersionFlagShort",
			args: []string{"-v"},
			assertion: func(flags MainFlags) {
				if !flags.ShowVersion {
					t.Errorf("Expected ShowVersion to be true, got false")
				}
			},
		},
		{
			name: "ShowVersionFlagLong",
			args: []string{"--version"},
			assertion: func(flags MainFlags) {
				if !flags.ShowVersion {
					t.Errorf("Expected ShowVersion to be true, got false")
				}
			},
		},
		{
			name: "ReconfigureFlag",
			args: []string{"--reconfigure"},
			assertion: func(flags MainFlags) {
				if !flags.Reconfigure {
					t.Errorf("Expected Reconfigure to be true, got false")
				}
			},
		},
		{
			name: "CreateSslFlagShort",
			args: []string{"-create-ssl"},
			assertion: func(flags MainFlags) {
				if !flags.CreateSsl {
					t.Errorf("Expected CreateSsl to be true, got false")
				}
			},
		},
		{
			name: "ConfigPathFlagShort",
			args: []string{"-c", "/path/to/config"},
			assertion: func(flags MainFlags) {
				if flags.ConfigPath != "/path/to/config" {
					t.Errorf("Expected ConfigPath to be '/path/to/config', got '%s'", flags.ConfigPath)
				}
			},
		},
		{
			name: "ConfigPathFlagLong",
			args: []string{"--config", "/path/to/config"},
			assertion: func(flags MainFlags) {
				if flags.ConfigPath != "/path/to/config" {
					t.Errorf("Expected ConfigPath to be '/path/to/config', got '%s'", flags.ConfigPath)
				}
			},
		},
		{
			name: "ConfigDirFlagShort",
			args: []string{"-cd", "/path/to/config/dir"},
			assertion: func(flags MainFlags) {
				if flags.ConfigDir != "/path/to/config/dir" {
					t.Errorf("Expected ConfigDir to be '/path/to/config/dir', got '%s'", flags.ConfigDir)
				}
			},
		},
		{
			name: "ConfigDirFlagLong",
			args: []string{"--config-dir", "/path/to/config/dir"},
			assertion: func(flags MainFlags) {
				if flags.ConfigDir != "/path/to/config/dir" {
					t.Errorf("Expected ConfigDir to be '/path/to/config/dir', got '%s'", flags.ConfigDir)
				}
			},
		},
		{
			name: "DataDirFlagShort",
			args: []string{"-d", "/path/to/data"},
			assertion: func(flags MainFlags) {
				if flags.DataDir != "/path/to/data" {
					t.Errorf("Expected DataDir to be '/path/to/data', got '%s'", flags.DataDir)
				}
			},
		},
		{
			name: "DataDirFlagLong",
			args: []string{"--data", "/path/to/data"},
			assertion: func(flags MainFlags) {
				if flags.DataDir != "/path/to/data" {
					t.Errorf("Expected DataDir to be '/path/to/data', got '%s'", flags.DataDir)
				}
			},
		},
		{
			name: "PortFlagShort",
			args: []string{"-p", "8080"},
			assertion: func(flags MainFlags) {
				if flags.Port != 8080 {
					t.Errorf("Expected Port to be 8080, got %d", flags.Port)
				}
			},
		},
		{
			name: "PortFlagLong",
			args: []string{"--port", "9090"},
			assertion: func(flags MainFlags) {
				if flags.Port != 9090 {
					t.Errorf("Expected Port to be 9090, got %d", flags.Port)
				}
			},
		},
		{
			name: "DisableCorsCheckFlag",
			args: []string{"--disable-cors-check"},
			assertion: func(flags MainFlags) {
				if !flags.DisableCorsCheck {
					t.Errorf("Expected DisableCorsCheck to be true, got false")
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Reset flags and arguments for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			os.Args = append([]string{os.Args[0]}, testCase.args...)

			flags := ParseFlags()
			testCase.assertion(flags)
		})
	}
}

func TestShowUsage(t *testing.T) {
	aliases := []alias{
		{Long: "version", Short: "v"},
		{Long: "config", Short: "c"},
		{Long: "data", Short: "d"},
	}

	flagSet := flag.NewFlagSet("test", flag.ExitOnError)
	flagSet.Bool("version", false, "Show version info")
	flagSet.Bool("v", false, "alias")
	flagSet.String("config", "", "Use provided config file")
	flagSet.String("c", "", "alias")
	flagSet.String("data", "", "Sets the data directory")
	flagSet.String("d", "", "alias")

	capturedOutput := captureOutput(func() {
		showUsage(*flagSet, aliases)()
	})

	expectedOutput := "Usage:\n\n" +
		"-c, --config <string>          Use provided config file\n" +
		"-d, --data <string>            Sets the data directory\n" +
		"-v, --version                  Show version info\n"

	test.IsEqualString(t, capturedOutput, expectedOutput)
}
func TestIsAlias(t *testing.T) {
	aliases := []alias{
		{Long: "version", Short: "v"},
		{Long: "config", Short: "c"},
		{Long: "data", Short: "d"},
	}

	testCases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"IsAliasShortV", "v", true},
		{"IsAliasShortC", "c", true},
		{"IsAliasShortD", "d", true},
		{"IsAliasLong", "version", false},
		{"NotAlias", "other", false},
		{"NotAliasShort", "o", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isAlias(tc.input, aliases)
			test.IsEqualBool(t, result, tc.expect)
		})
	}
}

func TestHasAlias(t *testing.T) {
	aliases := []alias{
		{Long: "version", Short: "v"},
		{Long: "config", Short: "c"},
		{Long: "data", Short: "d"},
	}

	testCases := []struct {
		name      string
		input     string
		expect    bool
		expectVal string
	}{
		{"HasAliasShort", "v", false, ""},
		{"HasAliasLong", "version", true, "v"},
		{"HasAliasLong", "config", true, "c"},
		{"HasAliasLong", "data", true, "d"},
		{"NotAlias", "other", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, val := hasAlias(tc.input, aliases)
			test.IsEqualBool(t, result, tc.expect)
			test.IsEqualString(t, val, tc.expectVal)
		})
	}
}

// Helper function to capture output from a function
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var capturedOutput bytes.Buffer
	io.Copy(&capturedOutput, r)

	return capturedOutput.String()
}
