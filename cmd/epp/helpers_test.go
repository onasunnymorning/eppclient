package main

import (
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestGenAuthInfo(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		a := genAuthInfo()
		if len(a) != authLength {
			t.Fatalf("len(%q) = %d", a, len(a))
		}
		for _, class := range []string{lowerChars, upperChars, digitChars, symbolChars} {
			if !strings.ContainsAny(a, class) {
				t.Fatalf("%q has no character from %q", a, class)
			}
		}
		if seen[a] {
			t.Fatalf("duplicate authInfo %q", a)
		}
		seen[a] = true
	}
}

func TestGenContactID(t *testing.T) {
	re := regexp.MustCompile(`^epp-[a-z0-9]{6}$`)
	for i := 0; i < 50; i++ {
		if id := genContactID(); !re.MatchString(id) {
			t.Fatalf("bad contact ID %q", id)
		}
	}
}

func TestNewTestContact(t *testing.T) {
	tc := newTestContact("")
	if tc.Email != "test@example.com" || tc.Postal.Name == "" || tc.Postal.City == "" || tc.Postal.CC == "" || tc.Auth == "" {
		t.Fatalf("incomplete test contact: %+v", tc)
	}
	if got := newTestContact("me@example.net").Email; got != "me@example.net" {
		t.Fatalf("email = %q", got)
	}
}

func TestNormalizeLegacyFlags(t *testing.T) {
	cases := []struct{ in, want string }{
		{"create domain x.test -period 2 -auth abc", "create domain x.test --period 2 --auth abc"},
		{"-profile ote check x.test", "--profile ote check x.test"},
		{"-v check x.test", "-v check x.test"},
		{"-vy create domain x.test", "-vy create domain x.test"},
		{"-period=2", "--period=2"},
		{"--period 2", "--period 2"},
		{"renew domain x.test --period -1", "renew domain x.test --period -1"},
		{"raw -", "raw -"},
		{"raw -- -odd", "raw -- -odd"},
	}
	for _, c := range cases {
		got := strings.Join(normalizeLegacyFlags(strings.Fields(c.in)), " ")
		if got != c.want {
			t.Errorf("normalizeLegacyFlags(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseConfig(t *testing.T) {
	ini := `
[default]
user = a
addr = epp.example:700
tls = true

[ote]
user = b
password = secret
contact = c-1
contact_roles = registrant, tech
ns = ns1.example.net , ns2.example.net
email = me@example.net
`
	cfg, err := parseConfig(strings.NewReader(ini), "ote")
	if err != nil {
		t.Fatal(err)
	}
	want := &Config{
		User:         "b",
		Password:     "secret",
		Contact:      "c-1",
		ContactRoles: []string{"registrant", "tech"},
		NS:           []string{"ns1.example.net", "ns2.example.net"},
		Email:        "me@example.net",
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("got %+v\nwant %+v", cfg, want)
	}

	cfg, err = parseConfig(strings.NewReader(ini), "default")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.ContactRoles, defaultContactRoles) {
		t.Fatalf("default roles = %v", cfg.ContactRoles)
	}

	if _, err := parseConfig(strings.NewReader(ini), "missing"); err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.json")

	st, err := readState(path)
	if err != nil || len(st) != 0 {
		t.Fatalf("missing file: %v %v", st, err)
	}
	st["ote"] = profileState{Contact: "epp-abc123"}
	if err := writeState(path, st); err != nil {
		t.Fatal(err)
	}
	got, err := readState(path)
	if err != nil || got["ote"].Contact != "epp-abc123" {
		t.Fatalf("got %v %v", got, err)
	}
}

func TestDefaultContactPrecedence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	opts.profile = "default"
	defer func() { cfg = nil }()

	cfg = &Config{}
	if id, _ := defaultContact(); id != "" {
		t.Fatalf("want no default, got %q", id)
	}

	saveContact("default", "epp-saved")
	if id, src := defaultContact(); id != "epp-saved" || src != "saved default" {
		t.Fatalf("got %q (%s)", id, src)
	}

	cfg.Contact = "c-profile"
	if id, src := defaultContact(); id != "c-profile" || src != "profile default" {
		t.Fatalf("credentials should win, got %q (%s)", id, src)
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"geoff.test":     "geoff.test",
		"ns1.a,ns2.b":    "ns1.a,ns2.b",
		"S3cret!pw":      "'S3cret!pw'",
		"it's":           `'it'\''s'`,
		"two words":      "'two words'",
		"me@example.net": "me@example.net",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}
