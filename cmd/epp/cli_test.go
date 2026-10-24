package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// fakeServer is a minimal EPP server that answers the commands the CLI's
// create flows send, and records every request.
type fakeServer struct {
	ln       net.Listener
	objects  []string
	taken    map[string]bool
	fees     []string // create fee items quoted by domain:check
	mu       sync.Mutex
	requests []string
}

func newFakeServer(t *testing.T, objects ...string) *fakeServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeServer{ln: ln, objects: objects, taken: map[string]bool{}, fees: []string{"12.00"}}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(c)
		}
	}()
	return s
}

func (s *fakeServer) addr() string { return s.ln.Addr().String() }

func (s *fakeServer) serve(c net.Conn) {
	defer c.Close()
	writeFrame(c, s.greeting())
	for {
		req, err := readFrame(c)
		if err != nil {
			return
		}
		s.mu.Lock()
		s.requests = append(s.requests, req)
		s.mu.Unlock()
		writeFrame(c, s.respond(req))
		if strings.Contains(req, "<logout") {
			return
		}
	}
}

func (s *fakeServer) greeting() string {
	var objs strings.Builder
	for _, o := range s.objects {
		objs.WriteString("<objURI>" + o + "</objURI>")
	}
	return `<?xml version="1.0" encoding="UTF-8"?><epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><greeting>` +
		`<svID>fake</svID><svDate>2026-10-24T00:00:00.0Z</svDate><svcMenu><version>1.0</version><lang>en</lang>` +
		objs.String() +
		`<svcExtension><extURI>urn:ietf:params:xml:ns:epp:fee-1.0</extURI></svcExtension>` +
		`</svcMenu></greeting></epp>`
}

var (
	reDomainName = regexp.MustCompile(`<domain:name>([^<]+)</domain:name>`)
	reContactID  = regexp.MustCompile(`<contact:id>([^<]+)</contact:id>`)
)

func (s *fakeServer) respond(req string) string {
	switch {
	case strings.Contains(req, "<login>"):
		return eppResponse(1000, "")
	case strings.Contains(req, "<logout"):
		return eppResponse(1500, "")
	case strings.Contains(req, "<domain:check"):
		var chk, fee strings.Builder
		for _, m := range reDomainName.FindAllStringSubmatch(req, -1) {
			avail := "1"
			if s.taken[m[1]] {
				avail = "0"
			}
			fmt.Fprintf(&chk, `<domain:cd><domain:name avail="%s">%s</domain:name></domain:cd>`, avail, m[1])
			var items strings.Builder
			for _, f := range s.fees {
				items.WriteString("<fee:fee>" + f + "</fee:fee>")
			}
			fmt.Fprintf(&fee, `<fee:cd avail="1"><fee:objID>%s</fee:objID><fee:command name="create"><fee:period unit="y">1</fee:period>%s</fee:command></fee:cd>`, m[1], items.String())
		}
		return eppResponse(1000,
			`<resData><domain:chkData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">`+chk.String()+`</domain:chkData></resData>`+
				`<extension><fee:chkData xmlns:fee="urn:ietf:params:xml:ns:epp:fee-1.0"><fee:currency>USD</fee:currency>`+fee.String()+`</fee:chkData></extension>`)
	case strings.Contains(req, "<contact:create"):
		id := reContactID.FindStringSubmatch(req)[1]
		return eppResponse(1000, `<resData><contact:creData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0">`+
			`<contact:id>`+id+`</contact:id><contact:crDate>2026-10-24T00:00:00.0Z</contact:crDate></contact:creData></resData>`)
	case strings.Contains(req, "<domain:create"):
		name := reDomainName.FindStringSubmatch(req)[1]
		if s.taken[name] {
			return eppResponse(2302, "")
		}
		return eppResponse(1000, `<resData><domain:creData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">`+
			`<domain:name>`+name+`</domain:name><domain:crDate>2026-10-24T00:00:00.0Z</domain:crDate>`+
			`<domain:exDate>2027-10-24T00:00:00.0Z</domain:exDate></domain:creData></resData>`)
	}
	return eppResponse(2000, "")
}

func eppResponse(code int, body string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><response>`+
		`<result code="%d"><msg>result %d</msg></result>%s<trID><svTRID>1</svTRID></trID></response></epp>`, code, code, body)
}

func writeFrame(w io.Writer, s string) {
	binary.Write(w, binary.BigEndian, uint32(4+len(s)))
	io.WriteString(w, s)
}

func readFrame(r io.Reader) (string, error) {
	var n uint32
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return "", err
	}
	buf := make([]byte, n-4)
	_, err := io.ReadFull(r, buf)
	return string(buf), err
}

// requests returns the recorded requests that contain substr.
func (s *fakeServer) requestsWith(substr string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, r := range s.requests {
		if strings.Contains(r, substr) {
			out = append(out, r)
		}
	}
	return out
}

const (
	objDomain  = "urn:ietf:params:xml:ns:domain-1.0"
	objContact = "urn:ietf:params:xml:ns:contact-1.0"
)

// setupHome points HOME at a temp dir with a credentials file for srv.
func setupHome(t *testing.T, srv *fakeServer, extra string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	creds := fmt.Sprintf("[default]\naddr = %s\nuser = tester\npassword = pw\ntls = false\n%s\n", srv.addr(), extra)
	if err := os.MkdirAll(filepath.Join(home, ".epp"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".epp", "credentials"), []byte(creds), 0600); err != nil {
		t.Fatal(err)
	}
	return home
}

// runCLI runs the CLI with args, capturing stdout and stderr.
func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	outF, _ := os.CreateTemp(t.TempDir(), "out")
	errF, _ := os.CreateTemp(t.TempDir(), "err")
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outF, errF
	code = run(args)
	os.Stdout, os.Stderr = oldOut, oldErr
	o, _ := os.ReadFile(outF.Name())
	e, _ := os.ReadFile(errF.Name())
	return code, string(o), string(e)
}

func mustContain(t *testing.T, what, s string, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			t.Errorf("%s does not contain %q:\n%s", what, sub, s)
		}
	}
}

func TestCreateDomainCreatesAndReusesTestContact(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	setupHome(t, srv, "email = me@example.net")

	code, out, errOut := runCLI(t, "create", "domain", "geoff.test", "-y")
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}

	contacts := srv.requestsWith("<contact:create")
	if len(contacts) != 1 {
		t.Fatalf("want 1 contact create, got %d", len(contacts))
	}
	id := reContactID.FindStringSubmatch(contacts[0])[1]
	mustContain(t, "contact create", contacts[0], "<contact:email>me@example.net</contact:email>")

	creates := srv.requestsWith("<domain:create")
	if len(creates) != 1 {
		t.Fatalf("want 1 domain create, got %d", len(creates))
	}
	mustContain(t, "domain create", creates[0],
		"<domain:registrant>"+id+"</domain:registrant>",
		`<domain:contact type="admin">`+id+`</domain:contact>`,
		`<domain:contact type="tech">`+id+`</domain:contact>`,
		`<domain:period unit="y">1</domain:period>`,
		"<fee:fee>12.00</fee:fee>",
		"<fee:currency>USD</fee:currency>",
		"<domain:pw>",
	)
	mustContain(t, "stdout", out, "Domain geoff.test created!", "AuthInfo: ")
	mustContain(t, "stderr", errOut, "Created test contact "+id, "To repeat without prompts:", "--registrant "+id)

	if got := savedContact("default"); got != id {
		t.Fatalf("saved contact = %q, want %q", got, id)
	}

	// The second run reuses the saved contact.
	code, _, errOut = runCLI(t, "create", "domain", "other.test", "-y")
	if code != 0 {
		t.Fatalf("second run exit %d\n%s", code, errOut)
	}
	if n := len(srv.requestsWith("<contact:create")); n != 1 {
		t.Fatalf("second run created another contact (total %d)", n)
	}
	mustContain(t, "second domain create", srv.requestsWith("<domain:create")[1], "<domain:registrant>"+id+"</domain:registrant>")
	mustContain(t, "stderr", errOut, "saved default")
}

func TestCreateDomainWithoutContactFailsWhenNotInteractive(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	setupHome(t, srv, "")

	code, _, errOut := runCLI(t, "create", "domain", "geoff.test")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	mustContain(t, "stderr", errOut, "no default contact", "-y")
	if n := len(srv.requestsWith("<domain:create")) + len(srv.requestsWith("<contact:create")); n != 0 {
		t.Fatalf("sent %d create requests, want none", n)
	}
}

func TestCreateDomainThinRegistrySkipsContacts(t *testing.T) {
	srv := newFakeServer(t, objDomain)
	setupHome(t, srv, "")

	code, _, errOut := runCLI(t, "create", "domain", "geoff.test")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	if n := len(srv.requestsWith("<contact:create")); n != 0 {
		t.Fatalf("created %d contacts on a thin registry", n)
	}
	create := srv.requestsWith("<domain:create")[0]
	if strings.Contains(create, "<domain:registrant>") || strings.Contains(create, "<domain:contact") {
		t.Fatalf("thin registry create has contacts:\n%s", create)
	}
}

func TestCreateDomainFlagsAndProfileDefaults(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	setupHome(t, srv, "contact = c-default\nns = ns1.example.net, ns2.example.net")

	// Old single-dash flag style, including a legacy flag name.
	code, _, errOut := runCLI(t, "create", "domain", "geoff.test",
		"-registrant", "c-reg", "-contact-admin", "c-admin", "-auth", "S3cret!pw", "-period", "2")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	create := srv.requestsWith("<domain:create")[0]
	mustContain(t, "domain create", create,
		"<domain:registrant>c-reg</domain:registrant>",
		`<domain:contact type="admin">c-admin</domain:contact>`,
		`<domain:contact type="tech">c-default</domain:contact>`,
		"<domain:hostObj>ns1.example.net</domain:hostObj>",
		"<domain:hostObj>ns2.example.net</domain:hostObj>",
		"<domain:pw>S3cret!pw</domain:pw>",
		`<domain:period unit="y">2</domain:period>`,
	)
	// The fee check prices one year, so it isn't sent for a 2-year create.
	if strings.Contains(create, "<fee:fee>") {
		t.Errorf("fee sent for a 2-year create:\n%s", create)
	}
	mustContain(t, "stderr", errOut, "fee check, not sent")
	if n := len(srv.requestsWith("<contact:create")); n != 0 {
		t.Fatalf("created %d contacts despite profile default", n)
	}
}

func TestCreateDomainTakenStillTriesWhenNotInteractive(t *testing.T) {
	srv := newFakeServer(t, objDomain)
	srv.taken["taken.test"] = true
	setupHome(t, srv, "")

	code, _, errOut := runCLI(t, "create", "domain", "taken.test")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	mustContain(t, "stderr", errOut, "taken.test is not available")
	if n := len(srv.requestsWith("<domain:create")); n != 1 {
		t.Fatalf("want the create to be sent once, got %d", n)
	}
}

func TestCreateContactDefault(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	setupHome(t, srv, "")

	code, out, errOut := runCLI(t, "create", "contact", "--default", "--name", "Geoff")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	req := srv.requestsWith("<contact:create")[0]
	id := reContactID.FindStringSubmatch(req)[1]
	mustContain(t, "contact create", req, "<contact:name>Geoff</contact:name>", "<contact:email>test@example.com</contact:email>")
	mustContain(t, "stdout", out, "Contact "+id+" created!")
	if got := savedContact("default"); got != id {
		t.Fatalf("saved contact = %q, want %q", got, id)
	}
}

func TestOfflineCommandsDontConnect(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no credentials file

	for _, args := range [][]string{{"--help"}, {"create", "domain", "--help"}, {"version"}} {
		if code, _, errOut := runCLI(t, args...); code != 0 {
			t.Errorf("%v: exit %d\n%s", args, code, errOut)
		}
	}
	if code, _, _ := runCLI(t, "create", "domain"); code != 1 {
		t.Errorf("missing domain name: exit %d, want 1", code)
	}
	if code, _, _ := runCLI(t, "info", "foo"); code != 1 {
		t.Errorf("unknown noun: exit %d, want 1", code)
	}
}

func TestCreateDomainEmptyFlagsOverrideDefaults(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	setupHome(t, srv, "contact = c-default\nns = ns1.example.net")

	code, _, errOut := runCLI(t, "create", "domain", "geoff.test", "--ns", "", "--tech", "")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	create := srv.requestsWith("<domain:create")[0]
	if strings.Contains(create, "<domain:hostObj>") || strings.Contains(create, `type="tech"`) {
		t.Fatalf("empty --ns/--tech were replaced by defaults:\n%s", create)
	}
	mustContain(t, "domain create", create, "<domain:registrant>c-default</domain:registrant>")
	// The equivalent command keeps the empty overrides, so a replay matches.
	mustContain(t, "stderr", errOut, "--tech ''", "--ns ''", "--registrant c-default")
}

func TestCreateContactEmptyFlagsOverridePlaceholders(t *testing.T) {
	srv := newFakeServer(t, objDomain, objContact)
	setupHome(t, srv, "")

	code, _, errOut := runCLI(t, "create", "contact", "--voice", "", "--sp", "")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	req := srv.requestsWith("<contact:create")[0]
	if strings.Contains(req, "<contact:voice>") || strings.Contains(req, "<contact:sp>") {
		t.Fatalf("empty --voice/--sp were replaced by placeholders:\n%s", req)
	}
	mustContain(t, "stderr", errOut, "--voice ''", "--sp ''", "--street ")
}

func TestCreateDomainLegacyAuthValueKept(t *testing.T) {
	srv := newFakeServer(t, objDomain)
	setupHome(t, srv, "")

	if code, _, errOut := runCLI(t, "create", "domain", "geoff.test", "-auth", "-Secret123!"); code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	mustContain(t, "domain create", srv.requestsWith("<domain:create")[0], "<domain:pw>-Secret123!</domain:pw>")
}

func TestCreateDomainMultipleFeeItemsNotAutoAccepted(t *testing.T) {
	srv := newFakeServer(t, objDomain)
	srv.fees = []string{"100.00", "12.00"}
	setupHome(t, srv, "")

	code, _, errOut := runCLI(t, "create", "domain", "geoff.test")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errOut)
	}
	if create := srv.requestsWith("<domain:create")[0]; strings.Contains(create, "<fee:fee>") {
		t.Fatalf("fee sent for a multi-item quote:\n%s", create)
	}
	mustContain(t, "stderr", errOut, "quoted 2 create fee items", "112.00 USD (100.00 + 12.00)", "fee check, not sent")
}
