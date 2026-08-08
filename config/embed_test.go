package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCreatesCompleteSampleTree(t *testing.T) {
	homeDir := t.TempDir()

	created, err := Install(homeDir)
	if err != nil {
		t.Fatal(err)
	}

	wantFiles := []string{
		".epp-id/credentials",
		".epp-id/ssl/certificate.key",
		".epp-id/ssl/certificate.pem",
		".epp-id/xml/example-batch.xml",
		".epp-id/xml/example.xml",
	}
	for _, path := range wantFiles {
		if !Created(created, path) {
			t.Errorf("Install() did not report %q as created", path)
		}
		if _, err := os.Stat(filepath.Join(homeDir, filepath.FromSlash(path))); err != nil {
			t.Errorf("installed file %q: %v", path, err)
		}
	}

	credentialsPath := filepath.Join(homeDir, ".epp-id", "credentials")
	credentials, err := os.ReadFile(credentialsPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(credentials), homePlaceholder) {
		t.Fatalf("credentials still contain home placeholder: %q", credentials)
	}
	if !strings.Contains(string(credentials), filepath.Join(homeDir, ".epp-id", "ssl", "certificate.pem")) {
		t.Fatalf("credentials do not contain installed certificate path: %q", credentials)
	}

	assertMode(t, credentialsPath, 0600)
	assertMode(t, filepath.Join(homeDir, ".epp-id", "ssl", "certificate.key"), 0600)
	assertMode(t, filepath.Join(homeDir, ".epp-id", "xml", "example.xml"), 0644)
	assertMode(t, filepath.Join(homeDir, ".epp-id", "log"), 0700)
	if _, err := os.Stat(filepath.Join(homeDir, ".epp-id", "log", ".gitkeep")); !os.IsNotExist(err) {
		t.Fatalf("repository placeholder was installed in the user log directory: %v", err)
	}
}

func TestInstallDoesNotOverwriteExistingFiles(t *testing.T) {
	homeDir := t.TempDir()
	if _, err := Install(homeDir); err != nil {
		t.Fatal(err)
	}

	credentialsPath := filepath.Join(homeDir, ".epp-id", "credentials")
	const customCredentials = "[default]\nuser = preserved\n"
	if err := os.WriteFile(credentialsPath, []byte(customCredentials), 0600); err != nil {
		t.Fatal(err)
	}

	created, err := Install(homeDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Fatalf("second Install() created files: %v", created)
	}

	credentials, err := os.ReadFile(credentialsPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(credentials); got != customCredentials {
		t.Fatalf("Install() overwrote credentials: %q", got)
	}
}

func TestInstallRecreatesMissingLogDirectory(t *testing.T) {
	homeDir := t.TempDir()
	if _, err := Install(homeDir); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(homeDir, ".epp-id", "log")
	if err := os.RemoveAll(logPath); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(homeDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(logPath, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(homeDir); err != nil {
		t.Fatal(err)
	}
	assertMode(t, logPath, 0700)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s permissions = %o, want %o", path, got, want)
	}
}
