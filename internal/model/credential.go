package model

import (
	"fmt"
	"log/slog"
)

// Credential represents authentication data for a provider.
type Credential struct {
	ProviderName string // Links to ProviderConfig.Name
	AuthMethod   AuthMethod
	Token        string // API token (if token auth)
	SSHKeyPath   string // Path to SSH key (if SSH auth)
	Username     string // Username for HTTPS auth
	// Password is NEVER stored — resolved at runtime from keychain
}

// String returns a safe representation that redacts the token.
func (c Credential) String() string {
	token := "[none]"
	if c.Token != "" {
		token = "[REDACTED]"
	}
	return fmt.Sprintf("Credential{provider=%s, method=%s, token=%s}", c.ProviderName, c.AuthMethod, token)
}

// LogValue implements slog.LogValuer to prevent token leakage in structured logs.
func (c *Credential) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("provider", c.ProviderName),
		slog.String("method", string(c.AuthMethod)),
	}
	if c.Token != "" {
		attrs = append(attrs, slog.String("token", "[REDACTED]"))
	}
	if c.SSHKeyPath != "" {
		attrs = append(attrs, slog.String("ssh_key_path", c.SSHKeyPath))
	}
	if c.Username != "" {
		attrs = append(attrs, slog.String("username", c.Username))
	}
	return slog.GroupValue(attrs...)
}
