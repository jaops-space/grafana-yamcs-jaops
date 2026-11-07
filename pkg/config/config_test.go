package config

import (
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestExtractConfigRejectsMalformedJSONAndNullEntries(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name:    "malformed JSON is rejected before processing entries",
			json:    `{"hosts":{"h1":null}} trailing`,
			wantErr: "could not unmarshal PluginSettings json",
		},
		{
			name:    "null host",
			json:    `{"hosts":{"h1":null}}`,
			wantErr: `host "h1" is null`,
		},
		{
			name:    "null endpoint",
			json:    `{"endpoints":{"e1":null}}`,
			wantErr: `endpoint "e1" is null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := backend.DataSourceInstanceSettings{JSONData: []byte(tt.json)}
			_, _, err := ExtractConfig(source)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
