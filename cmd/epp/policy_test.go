package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParsePolicySpec(t *testing.T) {
	cases := []struct {
		in   string
		want policySpec
	}{
		{"", policySpec{}},
		{"registrant:required, admin:prohibited, tech:optional", policySpec{"registrant": required, "admin": prohibited, "tech": optional}},
		{"registrant, Tech:Optional", policySpec{"registrant": required, "tech": optional}},
		{"none", policySpec{"registrant": prohibited, "admin": prohibited, "tech": prohibited, "billing": prohibited}},
		{"none, registrant", policySpec{"registrant": required, "admin": prohibited, "tech": prohibited, "billing": prohibited}},
		{"all", policySpec{"registrant": required, "admin": required, "tech": required, "billing": required}},
	}
	for _, c := range cases {
		got, err := parsePolicySpec(c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("parsePolicySpec(%q) = %v, %v; want %v", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"owner:required", "admin:forbidden"} {
		if _, err := parsePolicySpec(bad); err == nil {
			t.Errorf("parsePolicySpec(%q): expected an error", bad)
		}
	}
}

func TestParseRoleList(t *testing.T) {
	cases := map[string]map[string]bool{
		"none":            {},
		"all":             {"registrant": true, "admin": true, "tech": true, "billing": true},
		"registrant,TECH": {"registrant": true, "tech": true},
	}
	for in, want := range cases {
		got, err := parseRoleList(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("parseRoleList(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := parseRoleList("registrant,owner"); err == nil {
		t.Error("expected an error for an unknown role")
	}
}

func TestTLDCandidates(t *testing.T) {
	if got := tldCandidates("Geoff.CO.uk."); !reflect.DeepEqual(got, []string{"uk", "co.uk"}) {
		t.Fatalf("got %v", got)
	}
	if got := tldCandidates("example"); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestContactPoliciesLayers(t *testing.T) {
	tlds := map[string]policySpec{
		"uk":    {"admin": prohibited, "billing": prohibited},
		"co.uk": {"admin": optional},
	}
	profile := policySpec{"tech": optional}

	got := contactPolicies("geoff.co.uk", true, profile, tlds)
	want := policySet{
		"registrant": {required, "default"},
		"admin":      {optional, ".co.uk"},
		"tech":       {optional, "profile"},
		"billing":    {prohibited, ".uk"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}

	// No contact object in the greeting: everything prohibited, unless
	// configuration says otherwise.
	got = contactPolicies("geoff.test", false, policySpec{"registrant": required}, nil)
	if got["admin"] != (rolePolicy{prohibited, "registry greeting"}) || got["registrant"] != (rolePolicy{required, "profile"}) {
		t.Fatalf("got %v", got)
	}
}

func TestParseTLDPolicies(t *testing.T) {
	file := `
# comment
[.Example]
contacts = registrant:required, admin:prohibited   # inline comment
other = ignored

[thin]
contacts = none
`
	got, err := parseTLDPolicies(strings.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	if got["example"]["admin"] != prohibited || got["example"]["registrant"] != required || len(got["thin"]) != 4 {
		t.Fatalf("got %v", got)
	}

	_, err = parseTLDPolicies(strings.NewReader("[x]\ncontacts = admin:maybe\n"))
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("want an error naming line 2, got %v", err)
	}
}

func writeTLDs(t *testing.T, home, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, ".epp", "tlds"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDomainFollowsTLDPolicy(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	home := setupHome(t, srv, "contact = c-default")
	writeTLDs(t, home, "[example]\ncontacts = admin:prohibited, tech:optional, billing:prohibited\n")

	code, _, errOut := runCLI(t, "create", "domain", "geoff.example")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	create := srv.requestsWith("<domain:create")[0]
	mustContain(t, "domain create", create, "<domain:registrant>c-default</domain:registrant>")
	if strings.Contains(create, "<domain:contact") {
		t.Fatalf("sent contacts the TLD prohibits or makes optional:\n%s", create)
	}
	mustContain(t, "stderr", errOut, "prohibited by .example", "optional by .example, not sent", "--contacts registrant ")
}

func TestCreateDomainContactsNone(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	setupHome(t, srv, "") // no default contact, not interactive, no -y

	code, _, errOut := runCLI(t, "create", "domain", "geoff.test", "--contacts", "none")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	create := srv.requestsWith("<domain:create")[0]
	if strings.Contains(create, "<domain:registrant>") || strings.Contains(create, "<domain:contact") {
		t.Fatalf("--contacts none sent contacts:\n%s", create)
	}
	if n := len(srv.requestsWith("<contact:create")); n != 0 {
		t.Fatalf("created %d contacts", n)
	}
	mustContain(t, "stderr", errOut, "registrant contact is required, but won't be sent", "--contacts none")
}

func TestCreateDomainTLDWithoutContactsNeedsNoDefault(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	home := setupHome(t, srv, "")
	writeTLDs(t, home, "[thin]\ncontacts = none\n")

	code, _, errOut := runCLI(t, "create", "domain", "geoff.thin")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	if strings.Contains(srv.requestsWith("<domain:create")[0], "<domain:registrant>") {
		t.Fatal("sent a registrant to a TLD without contacts")
	}
	mustContain(t, "stderr", errOut, "contacts", "prohibited by .thin")
}

func TestCreateDomainContactsAllWithExplicitID(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	home := setupHome(t, srv, "contact = c-default")
	writeTLDs(t, home, "[test]\ncontacts = billing:prohibited\n")

	code, _, errOut := runCLI(t, "create", "domain", "geoff.test", "--contacts", "all", "--billing", "c-99")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	mustContain(t, "domain create", srv.requestsWith("<domain:create")[0],
		"<domain:registrant>c-default</domain:registrant>",
		`<domain:contact type="admin">c-default</domain:contact>`,
		`<domain:contact type="tech">c-default</domain:contact>`,
		`<domain:contact type="billing">c-99</domain:contact>`,
	)
	mustContain(t, "stderr", errOut, "billing contact is prohibited by .test, but will be sent", "sent anyway")
}

func TestCreateDomainBadContactsFlag(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	setupHome(t, srv, "contact = c-default")

	code, _, errOut := runCLI(t, "create", "domain", "geoff.test", "--contacts", "owner")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	mustContain(t, "stderr", errOut, `unknown contact role "owner"`)
	if n := len(srv.requestsWith("<domain:create")); n != 0 {
		t.Fatalf("sent %d creates", n)
	}
}
