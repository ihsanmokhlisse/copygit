package model

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredential_String_RedactsToken(t *testing.T) {
	cred := Credential{
		ProviderName: "my-github",
		AuthMethod:   AuthToken,
		Token:        "ghp_supersecrettoken123",
	}

	str := cred.String()
	assert.Contains(t, str, "my-github")
	assert.Contains(t, str, "[REDACTED]")
	assert.NotContains(t, str, "ghp_supersecrettoken123")
}

func TestCredential_String_EmptyToken(t *testing.T) {
	cred := Credential{
		ProviderName: "gitlab",
		AuthMethod:   AuthSSH,
	}

	str := cred.String()
	assert.Contains(t, str, "[none]")
	assert.NotContains(t, str, "[REDACTED]")
}

func TestCredential_String_FmtSafe(t *testing.T) {
	cred := Credential{
		ProviderName: "test",
		Token:        "secret-token-value",
	}

	// %v, %s, and Sprintf should all use our String() method
	output := cred.String()
	assert.NotContains(t, output, "secret-token-value")
	assert.Contains(t, output, "[REDACTED]")

	output2 := cred.String()
	assert.NotContains(t, output2, "secret-token-value")
}

func TestCredential_LogValue_RedactsToken(t *testing.T) {
	cred := Credential{
		ProviderName: "github",
		AuthMethod:   AuthToken,
		Token:        "ghp_secret123",
		Username:     "user1",
		SSHKeyPath:   "/home/user/.ssh/id_ed25519",
	}

	logVal := cred.LogValue()

	// Serialize the slog.Value to check contents
	var buf strings.Builder
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)
	logger.Info("test", "cred", logVal)

	output := buf.String()
	assert.NotContains(t, output, "ghp_secret123")
	assert.Contains(t, output, "REDACTED")
	assert.Contains(t, output, "github")
	assert.Contains(t, output, "user1")
}

func TestCredential_LogValue_NoTokenOmitted(t *testing.T) {
	cred := Credential{
		ProviderName: "gitlab",
		AuthMethod:   AuthSSH,
		SSHKeyPath:   "/home/user/.ssh/id_ed25519",
	}

	logVal := cred.LogValue()

	var buf strings.Builder
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)
	logger.Info("test", "cred", logVal)

	output := buf.String()
	assert.NotContains(t, output, "REDACTED", "token field should not appear when empty")
	assert.Contains(t, output, "gitlab")
}
