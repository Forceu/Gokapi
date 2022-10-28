package models

import "testing"

func TestE2EInfoEncrypted_HasBeenSetUp(t *testing.T) {
	type fields struct {
		Version        int
		Nonce          []byte
		Content        []byte
		AvailableFiles []string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"empty", fields{}, false},
		{"version 0", fields{Version: 0}, false},
		{"version 1, empty", fields{Version: 1}, false},
		{"version 0, not empty", fields{Version: 0, Content: []byte("content")}, false},
		{"version 1, not empty", fields{Version: 1, Content: []byte("content")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &E2EInfoEncrypted{
				Version:        tt.fields.Version,
				Nonce:          tt.fields.Nonce,
				Content:        tt.fields.Content,
				AvailableFiles: tt.fields.AvailableFiles,
			}
			if got := e.HasBeenSetUp(); got != tt.want {
				t.Errorf("HasBeenSetUp() = %v, want %v", got, tt.want)
			}
		})
	}
}
