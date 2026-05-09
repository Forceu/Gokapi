package helper

import (
	"testing"

	"github.com/forceu/gokapi/internal/test"
)

func TestByteCountSI(t *testing.T) {
	test.IsEqualString(t, ByteCountSI(5), "5 B")
	test.IsEqualString(t, ByteCountSI(5000), "4.9 kB")
	test.IsEqualString(t, ByteCountSI(5000000), "4.8 MB")
	test.IsEqualString(t, ByteCountSI(5000000000), "4.7 GB")
	test.IsEqualString(t, ByteCountSI(5000000000000), "4.5 TB")
}

func TestCleanString(t *testing.T) {
	test.IsEqualString(t, cleanRandomString("abc-123%%___!"), "abc123")
}

func TestGenerateRandomString(t *testing.T) {
	test.IsEqualBool(t, len(GenerateRandomString(100)) == 100, true)
}

func TestSanitiseContentType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// ── Valid types — must pass through unchanged ──────────────────────
		{
			name:  "plain type",
			input: "text/plain",
			want:  "text/plain",
		},
		{
			name:  "type with charset parameter",
			input: "text/html; charset=utf-8",
			want:  "text/html charset=utf-8", // semicolon and space are stripped
		},
		{
			name:  "application/octet-stream",
			input: "application/octet-stream",
			want:  "application/octet-stream",
		},
		{
			name:  "image type with plus",
			input: "image/svg+xml",
			want:  "image/svg+xml",
		},
		{
			name:  "type with dot",
			input: "application/vnd.ms-excel",
			want:  "application/vnd.ms-excel",
		},

		// ── Injection characters stripped ──────────────────────────────────
		{
			name:  "CRLF injection attempt",
			input: "text/plain\r\nX-Evil: header",
			want:  "text/plainX-Evil header",
		},
		{
			name:  "null byte stripped",
			input: "text/plain\x00evil",
			want:  "text/plainevil",
		},
		{
			name:  "angle brackets stripped",
			input: "text/<script>",
			want:  "text/script",
		},
		{
			name:  "quotes stripped",
			input: `application/"json"`,
			want:  "application/json",
		},

		// ── Fallback to application/octet-stream ──────────────────────────
		{
			// too short (< 2 non-space chars) → must return default, NOT panic
			name:  "empty string",
			input: "",
			want:  "application/octet-stream",
		},
		{
			name:  "single character",
			input: "x",
			want:  "application/octet-stream",
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  "application/octet-stream",
		},
		{
			// exactly 101 characters → over the 100-char limit
			name:  "over 100 chars returns default not a concat",
			input: "test/octet-stream-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			want:  "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitiseContentType(tt.input)
			if got != tt.want {
				t.Errorf("SanitiseContentType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitiseFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Normal filenames — must pass through unchanged
		{
			name:  "plain ascii filename",
			input: "report.pdf",
			want:  "report.pdf",
		},
		{
			name:  "filename with spaces",
			input: "my document.docx",
			want:  "my document.docx",
		},
		{
			name:  "filename with numbers",
			input: "backup_2024-01-15.tar.gz",
			want:  "backup_2024-01-15.tar.gz",
		},
		{
			name:  "unicode filename preserved",
			input: "résumé.pdf",
			want:  "résumé.pdf",
		},
		{
			name:  "japanese filename preserved",
			input: "ファイル.txt",
			want:  "ファイル.txt",
		},

		// Path traversal
		{
			name:  "unix path traversal",
			input: "../../etc/passwd",
			want:  "_.._etc_passwd",
		},
		{
			name:  "absolute unix path",
			input: "/etc/shadow",
			want:  "_etc_shadow",
		},
		{
			name:  "deep traversal",
			input: "a/b/c/d/e/secret.key",
			want:  "a_b_c_d_e_secret.key",
		},
		{
			name:  "Windows traversal",
			input: "a\\b\\c\\d\\e\\secret.key",
			want:  "a_b_c_d_e_secret.key",
		},

		// HTTP header injection
		{
			name:  "CRLF injection",
			input: "file.txt\r\nSet-Cookie: malicious=1",
			want:  "file.txtSet-Cookie_ malicious=1",
		},
		{
			name:  "LF only injection",
			input: "file.txt\nX-Injected: header",
			want:  "file.txtX-Injected_ header",
		},
		{
			name:  "CR only",
			input: "file.txt\rEvil",
			want:  "file.txtEvil",
		},

		// Null byte injection
		{
			name:  "null byte in middle",
			input: "file\x00.txt",
			want:  "file.txt",
		},
		{
			name:  "null byte at start",
			input: "\x00evil.sh",
			want:  "evil.sh",
		},

		// Control characters
		{
			name:  "control chars stripped",
			input: "file\x01\x1F\x7Fname.txt",
			want:  "filename.txt",
		},
		{
			name:  "tab character stripped",
			input: "file\tname.txt",
			want:  "filename.txt",
		},

		// Windows-forbidden characters
		{
			name:  "colon in filename",
			input: "con:file.txt",
			want:  "con_file.txt",
		},
		{
			name:  "asterisk in filename",
			input: "file*.txt",
			want:  "file_.txt",
		},
		{
			name:  "question mark",
			input: "file?.txt",
			want:  "file_.txt",
		},
		{
			name:  "angle brackets",
			input: "file<name>.txt",
			want:  "file_name_.txt",
		},
		{
			name:  "pipe character",
			input: "file|cmd.txt",
			want:  "file_cmd.txt",
		},
		{
			name:  "double quote",
			input: `file"name.txt`,
			want:  "file_name.txt",
		},

		// Hidden file prevention
		{
			name:  "leading single dot",
			input: ".bashrc",
			want:  "bashrc",
		},
		{
			name:  "leading double dot",
			input: "..config",
			want:  "config",
		},
		{
			name:  "leading many dots",
			input: "....hidden",
			want:  "hidden",
		},
		{
			name:  "dot in middle preserved",
			input: "my.file.txt",
			want:  "my.file.txt",
		},

		// Whitespace edge cases
		{
			name:  "leading and trailing spaces",
			input: "  file.txt  ",
			want:  "file.txt",
		},

		// Empty / degenerate inputs
		{
			name:  "empty string falls back",
			input: "",
			want:  "",
		},
		{
			name:  "only dots falls back",
			input: "...",
			want:  "",
		},
		{
			name:  "only spaces falls back",
			input: "   ",
			want:  "",
		},
		{
			name:  "only forbidden chars falls back",
			input: `\/:*?"<>|`,
			want:  "_________",
		},
		{
			name:  "only control chars falls back",
			input: "\x01\x02\x7F",
			want:  "",
		},

		// Combined attacks
		{
			name:  "traversal plus CRLF",
			input: "../../etc/passwd\r\nSet-Cookie: x=1",
			want:  "_.._etc_passwdSet-Cookie_ x=1",
		},
		{
			name:  "null byte traversal bypass attempt",
			input: "../../etc/passwd\x00.jpg",
			want:  "_.._etc_passwd.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitiseFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitiseFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
