package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateBaseURL(t *testing.T) { //nolint:funlen // table-driven test
	tests := []struct {
		name     string
		url      string
		wantErrs int
		contains string // substring expected in first error (if any)
	}{
		{
			name:     "valid HTTPS URL",
			url:      "https://github.com",
			wantErrs: 0,
		},
		{
			name:     "valid HTTPS with path",
			url:      "https://git.example.com/repos",
			wantErrs: 0,
		},
		{
			name:     "HTTP warns but allowed",
			url:      "http://git.internal.com",
			wantErrs: 1,
			contains: "should use HTTPS",
		},
		{
			name:     "file scheme blocked",
			url:      "file:///etc/passwd",
			wantErrs: 1,
			contains: "unsupported scheme",
		},
		{
			name:     "ftp scheme blocked",
			url:      "ftp://files.example.com",
			wantErrs: 1,
			contains: "unsupported scheme",
		},
		{
			name:     "loopback IP blocked",
			url:      "https://127.0.0.1",
			wantErrs: 1,
			contains: "private/internal IP",
		},
		{
			name:     "localhost IP blocked",
			url:      "https://127.0.0.2",
			wantErrs: 1,
			contains: "private/internal IP",
		},
		{
			name:     "private 10.x blocked",
			url:      "https://10.0.0.1",
			wantErrs: 1,
			contains: "private/internal IP",
		},
		{
			name:     "private 192.168.x blocked",
			url:      "https://192.168.1.1",
			wantErrs: 1,
			contains: "private/internal IP",
		},
		{
			name:     "private 172.16.x blocked",
			url:      "https://172.16.0.1",
			wantErrs: 1,
			contains: "private/internal IP",
		},
		{
			name:     "link-local 169.254.x blocked (AWS metadata)",
			url:      "https://169.254.169.254",
			wantErrs: 1,
			contains: "private/internal IP",
		},
		{
			name:     "GCP metadata endpoint blocked",
			url:      "https://metadata.google.internal",
			wantErrs: 1,
			contains: "cloud metadata",
		},
		{
			name:     "public IP allowed",
			url:      "https://140.82.121.3",
			wantErrs: 0,
		},
		{
			name:     "HTTP + private IP: two errors",
			url:      "http://192.168.1.1",
			wantErrs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateBaseURL(tt.url)
			assert.Len(t, errs, tt.wantErrs, "URL: %s, errors: %v", tt.url, errs)
			if tt.contains != "" && len(errs) > 0 {
				assert.Contains(t, errs[0], tt.contains)
			}
		})
	}
}

func TestValidateBaseURL_InvalidURL(t *testing.T) {
	errs := validateBaseURL("://not-a-url")
	assert.NotEmpty(t, errs)
}
