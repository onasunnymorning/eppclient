package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	User     string
	Password string
	Addr     string
	TLS      bool
	Cert     string
	Key      string
	CACert   string

	// Optional defaults used when the matching flag is omitted.
	Contact      string   // contact ID for every role in ContactRoles
	ContactRoles []string // roles to fill on domain create
	NS           []string // nameservers for domain create
	Email        string   // email for generated test contacts
}

var defaultContactRoles = []string{"registrant", "admin", "tech"}

func credentialsPath() (string, error) {
	dir, err := eppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials"), nil
}

func loadConfig(profile string) (*Config, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, initConfig(path)
		}
		return nil, fmt.Errorf("could not open credentials file: %w", err)
	}
	defer f.Close()

	return parseConfig(f, profile)
}

// parseConfig reads one profile section from a basic INI file.
func parseConfig(r io.Reader, profile string) (*Config, error) {
	cfg := &Config{ContactRoles: defaultContactRoles}
	found := false
	currentSection := ""

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			found = found || currentSection == profile
			continue
		}

		if currentSection != profile {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)

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
		case "contact":
			cfg.Contact = stripComment(val)
		case "contact_roles":
			if roles := parseList(stripComment(val)); len(roles) > 0 {
				cfg.ContactRoles = roles
			}
		case "ns":
			cfg.NS = parseList(stripComment(val))
		case "email":
			cfg.Email = stripComment(val)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("profile %s not found (profiles: %s)", profile, strings.Join(listProfiles(), ", "))
	}
	if cfg.User == "" {
		return nil, fmt.Errorf("user not found in profile %s", profile)
	}

	return cfg, nil
}

// stripComment removes a trailing "# comment" (or "; comment") preceded by
// whitespace. Only used for the default keys: contact IDs, roles, host names
// and email addresses can't contain such a sequence, while passwords can.
func stripComment(val string) string {
	for i := 0; i < len(val); i++ {
		if (val[i] == '#' || val[i] == ';') && (i == 0 || val[i-1] == ' ' || val[i-1] == '\t') {
			return strings.TrimSpace(val[:i])
		}
	}
	return val
}

// listProfiles returns the section names in the credentials file.
func listProfiles() []string {
	path, err := credentialsPath()
	if err != nil {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var profiles []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			profiles = append(profiles, line[1:len(line)-1])
		}
	}
	return profiles
}

const configTemplate = `[default]
# Your EPP server address (e.g., epp.example.com:700)
addr =
user =
password =
tls = true

# Optional: TLS client certificate and key
# cert =
# key =

# Optional: Custom CA certificate for server validation
# ca =

# Optional defaults, used when the matching flag is left out:
# contact = my-contact-id          # contact for registrant/admin/tech
# contact_roles = registrant,admin,tech
# ns = ns1.example.net,ns2.example.net
# email = me@example.net           # email for generated test contacts

[ote]
addr = epp.ote.registry.example:700
user =
password =
tls = true
`

func initConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("could not create configuration directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(configTemplate), 0600); err != nil {
		return fmt.Errorf("could not create credentials file %s: %w", path, err)
	}

	fmt.Fprintf(os.Stderr, "Configuration file created at %s\n", path)
	fmt.Fprintf(os.Stderr, "Please edit this file with your EPP credentials and run the command again.\n")
	os.Exit(0)
	return nil
}
