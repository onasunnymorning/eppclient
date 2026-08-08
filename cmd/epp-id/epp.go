package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	epp "github.com/freddymanullang/epp-id"
	"github.com/wsxiaoys/terminal/color"
)

var (
	profileName string
	verbose     bool
	version     = "EPP-ID 2026080801"
	maintainer  = "freddy@pandi.id"
)

func main() {
	// Global flags
	flag.StringVar(&profileName, "profile", "default", "profile name in ~/.epp/credentials")
	flag.BoolVar(&verbose, "v", false, "enable verbose debug logging")

	// Capture logs
	var logBuf bytes.Buffer

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <command> [arguments]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  check   Check domain availability or SSL connectivity\n")
		fmt.Fprintf(os.Stderr, "  create  Create a domain\n")
		fmt.Fprintf(os.Stderr, "  renew   Renew a domain\n")
		fmt.Fprintf(os.Stderr, "  poll    Check EPP poll messages\n")
		fmt.Fprintf(os.Stderr, "  transfer Transfer a domain\n")
		fmt.Fprintf(os.Stderr, "  raw     Send raw XML from a file or stdin\n")
		fmt.Fprintf(os.Stderr, "  batch   Send EPP commands from an eppBatch XML file\n")
		fmt.Fprintf(os.Stderr, "  info    Get domain info\n")
		fmt.Fprintf(os.Stderr, "  update  Update domain, contact or host\n")
		fmt.Fprintf(os.Stderr, "  version Print version information\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	// We need to parse flags before subcommands to get -profile and -v
	// But flag.Parse() consumes args. So we need to be careful.
	// Actually, standard go flag pkg stops at non-flag args.
	// So `epp-id -profile foo check domain.com` works.
	// `epp-id check -profile foo` does NOT work for global flags if we do it this way.
	// But usually subcommands have their own flags.

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	// Manual flag parsing for global flags before subcommand
	// This is a bit hacky but allows `epp-id -v check ...`
	// A better way is to parse, then look at remaining args.
	flag.Parse()

	if verbose {
		epp.DebugLogger = io.MultiWriter(os.Stderr, &logBuf)
	} else {
		epp.DebugLogger = &logBuf
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	cmd := args[0]
	subArgs := args[1:]
	commandStartedAt := time.Now()

	// Check usage before connecting
	checkUsage(cmd, subArgs)

	configDir, configInitialized, err := installUserConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing sample configuration: %v\n", err)
		os.Exit(1)
	}
	if configInitialized {
		fmt.Fprintf(os.Stderr, "Sample configuration installed at %s\n", configDir)
		fmt.Fprintln(os.Stderr, "Edit credentials and replace the sample certificates before connecting.")
		if cmd != "version" {
			os.Exit(0)
		}
	}

	if cmd == "version" {
		runVersion()
		os.Exit(0)
	}
	if isSSLCheck(cmd, subArgs) {
		path, err := runSSLCheck(profileName, ".", commandStartedAt)
		if path != "" {
			fmt.Fprintf(os.Stderr, "SSL test log saved to %s\n", path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "SSL connection test failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "SSL connection test succeeded")
		return
	}

	automaticSession := cmd != "batch"
	cfg, err := loadConfig(profileName, automaticSession)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading credentials for profile %q: %v\n", profileName, err)
		os.Exit(1)
	}

	conn := connect(cfg, automaticSession)

	defer func() {
		logger := epp.DebugLogger
		epp.DebugLogger = nil
		if r := recover(); r != nil {
			// If it was our fatal error, we already logged it.
			// Just ensure we print the raw XML log and then exit 1.
			// We can't easily distinguish our panic from others unless we use a custom type,
			// but for this CLI it's probably fine to treat all panics as "something went wrong".
			closeConnection(conn, automaticSession)
			commandEndedAt := time.Now()
			epp.DebugLogger = logger
			printAndSaveRawXML(os.Stderr, ".", commandStartedAt, commandEndedAt, &logBuf)
			os.Exit(1)
		}
		closeConnection(conn, automaticSession)
		commandEndedAt := time.Now()
		epp.DebugLogger = logger
		printAndSaveRawXML(os.Stderr, ".", commandStartedAt, commandEndedAt, &logBuf)
	}()

	switch cmd {
	case "check":
		runCheck(conn, subArgs)
	case "info":
		runInfo(conn, subArgs)
	case "delete":
		runDelete(conn, subArgs)
	case "create":
		runCreate(conn, subArgs)
	case "renew":
		runRenew(conn, subArgs)
	case "restore":
		runRestore(conn, subArgs)
	case "poll":
		runPoll(conn, subArgs)
	case "transfer":
		runTransfer(conn, subArgs)
	case "raw":
		runRaw(conn, subArgs)
	case "batch":
		runBatch(conn, subArgs)
	case "update":
		runUpdate(conn, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}

func checkUsage(cmd string, args []string) {
	switch cmd {
	case "check":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id check ssl | epp-id check [-phase phase] <domain>...")
			os.Exit(1)
		}
		if args[0] == "ssl" && len(args) != 1 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id check ssl")
			os.Exit(1)
		}
	case "info":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id info <domain|contact> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" && sub != "contact" {
			fmt.Fprintf(os.Stderr, "Unknown info type: %s. Use 'domain' or 'contact'.\n", sub)
			os.Exit(1)
		}
	case "delete":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id delete <domain|contact|host> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" && sub != "contact" && sub != "host" {
			fmt.Fprintf(os.Stderr, "Unknown delete type: %s. Use 'domain', 'contact' or 'host'.\n", sub)
			os.Exit(1)
		}
	case "create":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id create <domain|contact|host> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" && sub != "contact" && sub != "host" {
			fmt.Fprintf(os.Stderr, "Unknown create type: %s. Use 'domain', 'contact' or 'host'.\n", sub)
			os.Exit(1)
		}
	case "renew":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id renew <domain> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" {
			fmt.Fprintf(os.Stderr, "Unknown renewal type: %s. Use 'domain'.\n", sub)
			os.Exit(1)
		}
	case "restore":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id restore <domain> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" {
			fmt.Fprintf(os.Stderr, "Unknown restore type: %s. Use 'domain'.\n", sub)
			os.Exit(1)
		}
	case "poll":
		// Usage: epp-id poll [-ack id]
		// No strict enforcement needed for req
	case "transfer":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id transfer <domain> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" {
			fmt.Fprintf(os.Stderr, "Unknown transfer type: %s. Use 'domain'.\n", sub)
			os.Exit(1)
		}
	case "raw":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id raw <file>")
			os.Exit(1)
		}
	case "batch":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id batch <delay> <batch-file.xml>")
			os.Exit(1)
		}
		if _, _, err := parseBatchArgs(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "update":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp-id update <domain|contact|host> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" && sub != "contact" && sub != "host" {
			fmt.Fprintf(os.Stderr, "Unknown update type: %s. Use 'domain', 'contact' or 'host'.\n", sub)
			os.Exit(1)
		}
	case "version":
		// No args needed
	}
}

func runVersion() {
	fmt.Println(formatBuildInfo(runtime.Version()))
}

func formatBuildInfo(goVersion string) string {
	return fmt.Sprintf("version.BuildInfo{Version:%q, Maintainer:%q, GoVersion:%q}", version, maintainer, goVersion)
}

func connect(cfg *Config, automaticLogin bool) *epp.Conn {
	// Set up TLS
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}

	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		host = cfg.Addr
	}
	tlsCfg.ServerName = host

	if cfg.CACert != "" {
		ca, err := ioutil.ReadFile(cfg.CACert)
		fatalif(err)
		tlsCfg.RootCAs = x509.NewCertPool()
		tlsCfg.RootCAs.AppendCertsFromPEM(ca)
	}

	if cfg.Cert != "" && cfg.Key != "" {
		crt, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		fatalif(err)
		tlsCfg.Certificates = append(tlsCfg.Certificates, crt)
	}

	if !cfg.TLS {
		tlsCfg = nil
	}

	var conn net.Conn
	// TODO: Proxy support if needed from config

	color.Fprintf(os.Stderr, "Connecting to %s\n", cfg.Addr)
	conn, err = net.Dial("tcp", cfg.Addr)
	fatalif(err)

	if tlsCfg != nil {
		color.Fprintf(os.Stderr, "Establishing TLS connection\n")
		tc := tls.Client(conn, tlsCfg)
		err = tc.Handshake()
		fatalif(err)
		conn = tc
	}

	color.Fprintf(os.Stderr, "Performing EPP handshake\n")
	logger := epp.DebugLogger
	epp.DebugLogger = nil
	c, err := epp.NewConn(conn)
	epp.DebugLogger = logger
	fatalif(err)

	if automaticLogin {
		color.Fprintf(os.Stderr, "Logging in as %s...\n", cfg.User)
		logger = epp.DebugLogger
		epp.DebugLogger = nil
		_, err = c.Login(cfg.User, cfg.Password, "")
		epp.DebugLogger = logger
		fatalif(err)
	}

	return c
}

func closeConnection(c *epp.Conn, automaticLogout bool) {
	if automaticLogout {
		_ = c.Close()
		return
	}
	_ = c.Conn.Close()
}

func runCheck(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	phase := fs.String("phase", "", "launch phase (e.g., sunrise, open)")
	period := fs.Int("period", 1, "registration period in years")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp-id check [-phase phase] [-period N] <domain>...")
		os.Exit(1)
	}

	start := time.Now()
	var dc *epp.DomainCheckResponse
	var err error

	extData := make(map[string]string)
	if *phase != "" {
		extData["launch:phase"] = *phase
		extData["fee:phase"] = *phase
	}

	extData["fee:period"] = fmt.Sprintf("%d", *period)

	if len(extData) > 0 {
		dc, err = c.CheckDomainExtensions(fs.Args(), extData)
	} else {
		dc, err = c.CheckDomain(fs.Args()...)
	}

	logif(err)
	printDCR(dc)
	qdur := time.Since(start)
	color.Fprintf(os.Stderr, "@{.}Query: %s\n", qdur)
}

func runInfo(c *epp.Conn, args []string) {
	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "domain":
		runInfoDomain(c, subArgs)
	case "contact":
		runInfoContact(c, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown info type: %s. Use 'domain' or 'contact'.\n", cmd)
		os.Exit(1)
	}
}

func runInfoDomain(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp-id info domain <domain>")
		os.Exit(1)
	}
	res, err := c.DomainInfo(args[0], nil)
	fatalif(err)

	fmt.Printf("Domain: %s\n", res.Domain)
	fmt.Printf("ROID: %s\n", res.ID)
	fmt.Printf("Status: %v\n", res.Status)
	fmt.Printf("Created: %s\n", res.CrDate)
	fmt.Printf("Expires: %s\n", res.ExDate)
}

func runInfoContact(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("info contact", flag.ExitOnError)
	auth := fs.String("auth", "", "auth info (required for contact info)")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp-id info contact [-auth code] <contact-id>")
		os.Exit(1)
	}

	res, err := c.ContactInfo(fs.Arg(0), *auth, nil)
	fatalif(err)

	fmt.Printf("Contact: %s\n", res.ID)
	fmt.Printf("ROID: %s\n", res.ROID)
	fmt.Printf("Status: %v\n", res.Status)
	fmt.Printf("Email: %s\n", res.Email)
	fmt.Printf("Created: %s\n", res.CrDate)
}

func runDelete(c *epp.Conn, args []string) {
	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "domain":
		runDeleteDomain(c, subArgs)
	case "contact":
		runDeleteContact(c, subArgs)
	case "host":
		runDeleteHost(c, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown delete type: %s. Use 'domain', 'contact' or 'host'.\n", cmd)
		os.Exit(1)
	}
}

func runDeleteDomain(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp-id delete domain <domain>")
		os.Exit(1)
	}
	err := c.DeleteDomain(args[0], nil)
	fatalif(err)
	color.Printf("@{g}Domain %s deleted!\n", args[0])
}

func runDeleteContact(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp-id delete contact <contact-id>")
		os.Exit(1)
	}
	err := c.DeleteContact(args[0], nil)
	fatalif(err)
	color.Printf("@{g}Contact %s deleted!\n", args[0])
}

func runDeleteHost(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp-id delete host <host>")
		os.Exit(1)
	}
	err := c.DeleteHost(args[0])
	fatalif(err)
	color.Printf("@{g}Host %s deleted!\n", args[0])
}

func runCreate(c *epp.Conn, args []string) {
	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "domain":
		runCreateDomain(c, subArgs)
	case "contact":
		runCreateContact(c, subArgs)
