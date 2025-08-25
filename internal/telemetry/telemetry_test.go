package telemetry

import (
	"testing"
)

func TestNormalizeOTLPEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hostport string
		urlPath  string
		insecure bool
		resolved string
		wantErr  bool
	}{
		{"default localhost", "http://localhost:4318", "localhost:4318", "/v1/traces", true, "http://localhost:4318/v1/traces", false},
		{"trailing slash base", "http://collector:4318/", "collector:4318", "/v1/traces", true, "http://collector:4318/v1/traces", false},
		{"already traces path", "http://collector:4318/v1/traces", "collector:4318", "/v1/traces", true, "http://collector:4318/v1/traces", false},
		{"custom base path", "https://otlp.example.com:4318/otlp", "otlp.example.com:4318", "/otlp/v1/traces", false, "https://otlp.example.com:4318/otlp/v1/traces", false},
		{"invalid no scheme", "collector:4318", "", "", true, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hp, path, insecure, resolved, err := normalizeOTLPEndpoint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeOTLPEndpoint() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if hp != tt.hostport {
				t.Errorf("hostport = %q, want %q", hp, tt.hostport)
			}
			if path != tt.urlPath {
				t.Errorf("urlPath = %q, want %q", path, tt.urlPath)
			}
			if insecure != tt.insecure {
				t.Errorf("insecure = %v, want %v", insecure, tt.insecure)
			}
			if resolved != tt.resolved {
				t.Errorf("resolved = %q, want %q", resolved, tt.resolved)
			}
		})
	}
}
