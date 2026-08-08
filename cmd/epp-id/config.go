package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sampleconfig "github.com/freddymanullang/epp-id/config"
)

type Config struct {
	User     string
	Password string
	Addr     string
	TLS      bool
	Cert     string
	Key      string
	CACert   string
}

const (
	userConfigDirectoryName = ".epp-id"
	credentialsFilename     = "credentials"
	userLogDirectoryName    = "log"
)

func loadConfig(profile string, requireUser bool) (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(homeDir, userConfigDirectoryName, credentialsFilename)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open credentials file: %w", err)
	}
	defer f.Close()

	cfg := &Config{}
	scanner := bufio.NewScanner(f)
	currentSection := ""

	// Basic INI parser
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			continue
		}

		if currentSection != profile {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "user":
			cfg.User = val
		case "password":
			cfg.Password = val
		case "addr":
			cfg.Addr = val
		case "tls":
			cfg.TLS = (val == "true")
		case "cert":
			cfg.Cert = val
		case "key":
			cfg.Key = val
		case "ca":
			cfg.CACert = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if cfg.Addr == "" || strings.Contains(cfg.Addr, "CHANGE_ME") {
		return nil, fmt.Errorf("address not found in profile %s", profile)
	}

	if requireUser && (cfg.User == "" || strings.Contains(cfg.User, "CHANGE_ME")) {
		return nil, fmt.Errorf("user not found in profile %s", profile)
	}
	if requireUser && (cfg.Password == "" || strings.Contains(cfg.Password, "CHANGE_ME")) {
		return nil, fmt.Errorf("password not found in profile %s", profile)
	}

	return cfg, nil
}

func installUserConfig() (string, bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}

	created, err := sampleconfig.Install(homeDir)
	if err != nil {
		return "", false, err
	}

	configDir := filepath.Join(homeDir, userConfigDirectoryName)
	credentialsCreated := sampleconfig.Created(created, filepath.ToSlash(filepath.Join(userConfigDirectoryName, credentialsFilename)))
	return configDir, credentialsCreated, nil
}

func logDirectory(configDir string) string {
	return filepath.Join(configDir, userLogDirectoryName)
}
