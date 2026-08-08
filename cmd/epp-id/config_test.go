package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallUserConfigAndLoadCustomizedCredentials(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configDir, initialized, err := installUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !initialized {
		t.Fatal("installUserConfig() did not initialize credentials")
	}
	if got, want := configDir, filepath.Join(homeDir, ".epp-id"); got != want {
		t.Fatalf("config directory = %q, want %q", got, want)
	}
	if got, want := logDirectory(configDir), filepath.Join(homeDir, ".epp-id", "log"); got != want {
		t.Fatalf("log directory = %q, want %q", got, want)
	}
	if info, err := os.Stat(logDirectory(configDir)); err != nil {
		t.Fatalf("log directory was not created: %v", err)
	} else if got, want := info.Mode().Perm(), os.FileMode(0700); got != want {
		t.Fatalf("log directory permissions = %o, want %o", got, want)
	}

	if _, err := loadConfig("default", true); err == nil || !strings.Contains(err.Error(), "address not found") {
		t.Fatalf("loadConfig() placeholder error = %v", err)
	}

	credentials := `[default]
addr = epp.registry.example.id:700
user = client-id
password = secret
tls = true
`
	if err := os.WriteFile(filepath.Join(configDir, "credentials"), []byte(credentials), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig("default", true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "epp.registry.example.id:700" || cfg.User != "client-id" || cfg.Password != "secret" {
		t.Fatalf("loadConfig() = %+v", cfg)
	}

	_, initialized, err = installUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	if initialized {
		t.Fatal("installUserConfig() reinitialized existing credentials")
	}
}
