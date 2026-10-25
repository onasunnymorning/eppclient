package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Contact policy per role, as TLD policies usually state it.
type policy string

const (
	required   policy = "required"   // sent with the default contact unless given
	optional   policy = "optional"   // sent only when asked for
	prohibited policy = "prohibited" // never sent unless asked for, with a warning
)

var contactRoles = []string{"registrant", "admin", "tech", "billing"}

// rolePolicy is a role's policy and where it came from: "default",
// "profile", "registry greeting" or a TLD such as ".example".
type rolePolicy struct {
	policy policy
	source string
}

type policySet map[string]rolePolicy

func (p rolePolicy) String() string {
	if p.source == "default" {
		return string(p.policy)
	}
	return fmt.Sprintf("%s by %s", p.policy, p.source)
}

// policySpec maps roles to policies; roles it doesn't mention keep the
// policy of the layer below.
type policySpec map[string]policy

func builtinPolicies() policySet {
	return policySet{
		"registrant": {required, "default"},
		"admin":      {optional, "default"},
		"tech":       {optional, "default"},
		"billing":    {optional, "default"},
	}
}

func (ps policySet) apply(spec policySpec, source string) {
	for role, p := range spec {
		ps[role] = rolePolicy{p, source}
	}
}

// parsePolicySpec parses "registrant:required, admin:prohibited, tech".
// A bare role name means required; "none" prohibits every role and "all"
// requires every role.
func parsePolicySpec(s string) (policySpec, error) {
	spec := policySpec{}
	for _, item := range parseList(s) {
		switch strings.ToLower(item) {
		case "none":
			for _, r := range contactRoles {
				spec[r] = prohibited
			}
			continue
		case "all":
			for _, r := range contactRoles {
				spec[r] = required
			}
			continue
		}

		role, pol, hasPolicy := strings.Cut(item, ":")
		role = strings.ToLower(strings.TrimSpace(role))
		if !isContactRole(role) {
			return nil, fmt.Errorf("unknown contact role %q (roles: %s)", role, strings.Join(contactRoles, ", "))
		}
		p := required
		if hasPolicy {
			p = policy(strings.ToLower(strings.TrimSpace(pol)))
			if p != required && p != optional && p != prohibited {
				return nil, fmt.Errorf("unknown policy %q for %s (use required, optional or prohibited)", pol, role)
			}
		}
		spec[role] = p
	}
	return spec, nil
}

// parseRoleList parses the --contacts value: the roles to send, "all" or "none".
func parseRoleList(s string) (map[string]bool, error) {
	roles := map[string]bool{}
	for _, item := range parseList(s) {
		item = strings.ToLower(item)
		switch {
		case item == "none":
		case item == "all":
			for _, r := range contactRoles {
				roles[r] = true
			}
		case isContactRole(item):
			roles[item] = true
		default:
			return nil, fmt.Errorf("unknown contact role %q in --contacts (use %s, all or none)", item, strings.Join(contactRoles, ", "))
		}
	}
	return roles, nil
}

func isContactRole(s string) bool {
	for _, r := range contactRoles {
		if s == r {
			return true
		}
	}
	return false
}

// tldCandidates returns the suffixes of domain that could have a TLD
// policy, shortest first: "geoff.co.uk" gives "uk", "co.uk".
func tldCandidates(domain string) []string {
	labels := strings.Split(strings.Trim(strings.ToLower(domain), "."), ".")
	var out []string
	for i := len(labels) - 1; i >= 1; i-- {
		out = append(out, strings.Join(labels[i:], "."))
	}
	return out
}

// contactPolicies works out each role's policy for domain. Layers, each
// overriding the one before: built-in default, the registry's greeting (no
// contact object means none are allowed), the profile, then TLD policies
// from ~/.epp/tlds with the longest matching suffix last.
func contactPolicies(domain string, registryHasContacts bool, profile policySpec, tlds map[string]policySpec) policySet {
	ps := builtinPolicies()
	if !registryHasContacts {
		for _, r := range contactRoles {
			ps[r] = rolePolicy{prohibited, "registry greeting"}
		}
	}
	ps.apply(profile, "profile")
	for _, tld := range tldCandidates(domain) {
		if spec, ok := tlds[tld]; ok {
			ps.apply(spec, "."+tld)
		}
	}
	return ps
}

func tldsPath() (string, error) {
	dir, err := eppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tlds"), nil
}

// loadTLDPolicies reads ~/.epp/tlds; a missing file means no TLD policies.
func loadTLDPolicies() (map[string]policySpec, error) {
	path, err := tldsPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tlds, err := parseTLDPolicies(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return tlds, nil
}

// parseTLDPolicies reads an INI file with a [tld] section per TLD and a
// "contacts = ..." policy in each.
func parseTLDPolicies(r io.Reader) (map[string]policySpec, error) {
	tlds := map[string]policySpec{}
	section := ""
	scanner := bufio.NewScanner(r)
	for n := 1; scanner.Scan(); n++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(strings.ToLower(strings.TrimSpace(line[1:len(line)-1])), ".")
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || section == "" || strings.TrimSpace(key) != "contacts" {
			continue
		}
		spec, err := parsePolicySpec(stripComment(strings.TrimSpace(val)))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		tlds[section] = spec
	}
	return tlds, scanner.Err()
}
