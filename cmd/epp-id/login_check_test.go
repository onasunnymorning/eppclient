package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	epp "github.com/freddymanullang/epp-id"
)

func TestLoginCheckLogFilename(t *testing.T) {
	timestamp := time.Date(2026, time.August, 8, 21, 4, 5, 0, time.FixedZone("WIB", 7*60*60))

	if got, want := loginCheckLogFilename(timestamp), "test-login-20260808-210405.log"; got != want {
		t.Fatalf("loginCheckLogFilename() = %q, want %q", got, want)
	}
}

func TestPerformLoginCheckSuccess(t *testing.T) {
	cfg := &Config{Addr: "epp.registry.example.id:700", User: "client-id", Password: "secret", TLS: true}
	startedAt := time.Now()
	attempt := func(got *Config) (epp.Result, string, error) {
		if got != cfg {
			t.Fatal("performLoginCheck() passed a different config to the login attempt")
		}
		return epp.Result{Code: 1000, Message: "Command completed successfully"}, "", nil
	}

	result := performLoginCheck("sandbox", cfg, startedAt, attempt)

	if !result.LoginSuccessful || result.Err != nil {
		t.Fatalf("performLoginCheck() result = %+v", result)
	}
	if result.Profile != "sandbox" || result.Address != cfg.Addr || result.Username != cfg.User || !result.TLSEnabled {
		t.Fatalf("performLoginCheck() metadata = %+v", result)
	}
	if result.EPPResult.Code != 1000 {
		t.Fatalf("performLoginCheck() EPP result = %+v", result.EPPResult)
	}
	if result.EndedAt.Before(startedAt) {
		t.Fatalf("performLoginCheck() ended at %v before %v", result.EndedAt, startedAt)
	}
}

func TestRunLoginCheckSavesCredentialsFailure(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	configDir, _, err := installUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Date(2026, time.August, 8, 21, 4, 5, 0, time.UTC)

	path, err := runLoginCheck("default", logDirectory(configDir), startedAt)

	if err == nil || !strings.Contains(err.Error(), "address not found") {
		t.Fatalf("runLoginCheck() error = %v, want missing address error", err)
	}
	if got, want := filepath.Dir(path), logDirectory(configDir); got != want {
		t.Fatalf("runLoginCheck() log directory = %q, want %q", got, want)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	content := string(data)
	if !strings.Contains(content, "Result: FAILED") || !strings.Contains(content, "Failure Stage: credentials") {
		t.Fatalf("credentials failure log is incomplete:\n%s", content)
	}
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if !strings.HasPrefix(lines[len(lines)-1], "Duration Process:") {
		t.Fatalf("credentials failure log does not end with duration:\n%s", content)
	}
}

func TestFormatLoginCheckLogFailureEndsWithDuration(t *testing.T) {
	location := time.FixedZone("WIB", 7*60*60)
	startedAt := time.Date(2026, time.August, 8, 21, 4, 5, 250*1e6, location)
	result := loginCheckResult{
		Profile:      "sandbox",
		Address:      "epp.registry.example.id:700",
		Username:     "client-id",
		TLSEnabled:   true,
		StartedAt:    startedAt,
		EndedAt:      startedAt.Add(1250 * time.Millisecond),
		EPPResult:    epp.Result{Code: 2200, Message: "Authentication error"},
		FailureStage: "EPP login",
		Err:          errors.New("EPP result code 2200: Authentication error"),
	}

	content := string(formatLoginCheckLog(result))
	for _, expected := range []string{
		"EPP-ID Login Test",
		"Profile: sandbox",
		"EPP Server: epp.registry.example.id:700",
		"Username: client-id",
		"Credentials TLS Enabled: true",
		"EPP Login: FAILED",
		"EPP Result Code: 2200",
		"EPP Result Message: Authentication error",
		"Result: FAILED",
		"Failure Stage: EPP login",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("login log does not contain %q:\n%s", expected, content)
		}
	}
	if strings.Contains(content, "secret") {
		t.Fatalf("login log contains a password:\n%s", content)
	}
	if !strings.HasSuffix(content, "Duration Process: 1.25s\n") {
		t.Fatalf("login log does not end with Duration Process:\n%s", content)
	}
}

func TestSaveLoginCheckLogDoesNotOverwriteSameSecond(t *testing.T) {
	directory := t.TempDir()
	startedAt := time.Date(2026, time.August, 8, 21, 4, 5, 0, time.UTC)
	result := loginCheckResult{
		Profile:         "default",
		Address:         "epp.registry.example.id:700",
		Username:        "client-id",
		StartedAt:       startedAt,
		EndedAt:         startedAt.Add(time.Second),
		LoginSuccessful: true,
		EPPResult:       epp.Result{Code: 1000, Message: "Command completed successfully"},
	}

	firstPath, err := saveLoginCheckLog(directory, result)
	if err != nil {
		t.Fatal(err)
	}
	secondPath, err := saveLoginCheckLog(directory, result)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := filepath.Base(firstPath), "test-login-20260808-210405.log"; got != want {
		t.Fatalf("first filename = %q, want %q", got, want)
	}
	if got, want := filepath.Base(secondPath), "test-login-20260808-210405-2.log"; got != want {
		t.Fatalf("second filename = %q, want %q", got, want)
	}
	info, err := os.Stat(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0600); got != want {
		t.Fatalf("saved log permissions = %o, want %o", got, want)
	}
	data, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "Duration Process: 1s\n") {
		t.Fatalf("saved login log does not end with duration: %q", data)
	}
}
