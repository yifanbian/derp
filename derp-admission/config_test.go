package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadConfigExample(t *testing.T) {
	cfg, err := LoadConfig("config-example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Tailnets) != 2 {
		t.Fatalf("tailnets = %d, want 2", len(cfg.Tailnets))
	}
	if cfg.Tailnets[0].Auth.ClientCredential == nil || cfg.Tailnets[1].Auth.IdentityFederation == nil {
		t.Fatal("unexpected auth parsing")
	}
	if got := cfg.Tailnets[1].Auth.IdentityFederation.AzureIMDS.ClientID; got == "" {
		t.Fatal("azure client id not parsed")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfig(writeConfig(t, `
tailnets:
- name: a
  auth:
    clientCredential: {id: i, secret: s}
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8080" || cfg.RefreshInterval != 5*time.Minute {
		t.Fatalf("defaults = %q, %v", cfg.Listen, cfg.RefreshInterval)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	tests := map[string]struct{ yaml, wantErr string }{
		"no tailnets": {"listen: ':1'\n", "no tailnets"},
		"no name": {`
tailnets:
- auth: {clientCredential: {id: i, secret: s}}
`, "name is required"},
		"duplicate": {`
tailnets:
- {name: a, auth: {clientCredential: {id: i, secret: s}}}
- {name: a, auth: {clientCredential: {id: i, secret: s}}}
`, "duplicate"},
		"no auth": {"tailnets:\n- name: a\n", "auth is required"},
		"both auth": {`
tailnets:
- name: a
  auth:
    clientCredential: {id: i, secret: s}
    identityFederation: {id: i, azureImds: {}}
`, "only one"},
		"missing secret": {`
tailnets:
- {name: a, auth: {clientCredential: {id: i}}}
`, "id and secret"},
		"federation without provider": {`
tailnets:
- {name: a, auth: {identityFederation: {id: i}}}
`, "azureImds"},
		"unknown field": {`
tailnets:
- {name: a, auth: {clientCredential: {id: i, secret: s}}, bogus: 1}
`, "bogus"},
		"short interval": {`
refreshInterval: 1ms
tailnets:
- {name: a, auth: {clientCredential: {id: i, secret: s}}}
`, "refreshInterval"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}
