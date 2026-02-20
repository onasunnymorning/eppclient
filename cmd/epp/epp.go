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
	"strings"
	"time"

	epp "github.com/onasunnymorning/eppclient"
	"github.com/wsxiaoys/terminal/color"
)

var (
	profileName string
	verbose     bool
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
		fmt.Fprintf(os.Stderr, "  check   Check domain availability\n")
		fmt.Fprintf(os.Stderr, "  create  Create a domain\n")
		fmt.Fprintf(os.Stderr, "  renew   Renew a domain\n")
		fmt.Fprintf(os.Stderr, "  poll    Check EPP poll messages\n")
		fmt.Fprintf(os.Stderr, "  transfer Transfer a domain\n")
		fmt.Fprintf(os.Stderr, "  raw     Send raw XML from a file or stdin\n")
		fmt.Fprintf(os.Stderr, "  info    Get domain info\n")
		fmt.Fprintf(os.Stderr, "  update  Update domain, contact or host\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	// We need to parse flags before subcommands to get -profile and -v
	// But flag.Parse() consumes args. So we need to be careful.
	// Actually, standard go flag pkg stops at non-flag args.
	// So `epp -profile foo check domain.com` works.
	// `epp check -profile foo` does NOT work for global flags if we do it this way.
	// But usually subcommands have their own flags.

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	// Manual flag parsing for global flags before subcommand
	// This is a bit hacky but allows `epp -v check ...`
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

	// Check usage before connecting
	checkUsage(cmd, subArgs)

	cfg, err := loadConfig(profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading credentials for profile %q: %v\n", profileName, err)
		os.Exit(1)
	}

	conn := connect(cfg)

	defer func() {
		logger := epp.DebugLogger
		epp.DebugLogger = nil
		if r := recover(); r != nil {
			// If it was our fatal error, we already logged it.
			// Just ensure we prompt and then exit 1.
			// We can't easily distinguish our panic from others unless we use a custom type,
			// but for this CLI it's probably fine to treat all panics as "something went wrong".
			// But we DO want to run promptRawXML.
			conn.Close()
			epp.DebugLogger = logger
			promptRawXML(&logBuf)
			os.Exit(1)
		}
		conn.Close()
		epp.DebugLogger = logger
		promptRawXML(&logBuf)
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
			fmt.Fprintln(os.Stderr, "Usage: epp check <domain>...")
			os.Exit(1)
		}
	case "info":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp info <domain|contact> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" && sub != "contact" {
			fmt.Fprintf(os.Stderr, "Unknown info type: %s. Use 'domain' or 'contact'.\n", sub)
			os.Exit(1)
		}
	case "delete":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp delete <domain|contact|host> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" && sub != "contact" && sub != "host" {
			fmt.Fprintf(os.Stderr, "Unknown delete type: %s. Use 'domain', 'contact' or 'host'.\n", sub)
			os.Exit(1)
		}
	case "create":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp create <domain|contact|host> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" && sub != "contact" && sub != "host" {
			fmt.Fprintf(os.Stderr, "Unknown create type: %s. Use 'domain', 'contact' or 'host'.\n", sub)
			os.Exit(1)
		}
	case "renew":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp renew <domain> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" {
			fmt.Fprintf(os.Stderr, "Unknown renewal type: %s. Use 'domain'.\n", sub)
			os.Exit(1)
		}
	case "restore":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp restore <domain> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" {
			fmt.Fprintf(os.Stderr, "Unknown restore type: %s. Use 'domain'.\n", sub)
			os.Exit(1)
		}
	case "poll":
		// Usage: epp poll [-ack id]
		// No strict enforcement needed for req
	case "transfer":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp transfer <domain> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" {
			fmt.Fprintf(os.Stderr, "Unknown transfer type: %s. Use 'domain'.\n", sub)
			os.Exit(1)
		}
	case "raw":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp raw <file>")
			os.Exit(1)
		}
	case "update":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: epp update <domain|contact|host> [options]")
			os.Exit(1)
		}
		sub := args[0]
		if sub != "domain" && sub != "contact" && sub != "host" {
			fmt.Fprintf(os.Stderr, "Unknown update type: %s. Use 'domain', 'contact' or 'host'.\n", sub)
			os.Exit(1)
		}
	}
}

func connect(cfg *Config) *epp.Conn {
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

	color.Fprintf(os.Stderr, "Logging in as %s...\n", cfg.User)
	logger = epp.DebugLogger
	epp.DebugLogger = nil
	err = c.Login(cfg.User, cfg.Password, "")
	epp.DebugLogger = logger
	fatalif(err)

	return c
}

func runCheck(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp check <domain>...")
		os.Exit(1)
	}

	start := time.Now()
	dc, err := c.CheckDomain(args...)
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
		fmt.Fprintln(os.Stderr, "Usage: epp info domain <domain>")
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
		fmt.Fprintln(os.Stderr, "Usage: epp info contact [-auth code] <contact-id>")
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
		fmt.Fprintln(os.Stderr, "Usage: epp delete domain <domain>")
		os.Exit(1)
	}
	err := c.DeleteDomain(args[0], nil)
	fatalif(err)
	color.Printf("@{g}Domain %s deleted!\n", args[0])
}

func runDeleteContact(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp delete contact <contact-id>")
		os.Exit(1)
	}
	err := c.DeleteContact(args[0], nil)
	fatalif(err)
	color.Printf("@{g}Contact %s deleted!\n", args[0])
}

func runDeleteHost(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp delete host <host>")
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
	case "host":
		runCreateHost(c, subArgs)
	default:
		// Fallback for backward compatibility or simple "epp create domain.com"?
		// The user explicitly asked for "move to epp create domain", so enforcing subcommand is correct.
		// However, it's nice to allow "epp create domain.com" if it doesn't look like "contact"?
		// But "domain" IS the subcommand.
		// If I type `epp create example.com` -> cmd="example.com".
		// Maybe warning? Or just fail.
		// User: "lets move 'epp create' to 'epp create domain'"
		fmt.Fprintf(os.Stderr, "Unknown create type: %s. Use 'domain' or 'contact'.\n", cmd)
		os.Exit(1)
	}
}

func runCreateDomain(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("create domain", flag.ExitOnError)
	period := fs.Int("period", 1, "registration period in years")
	auth := fs.String("auth", "", "auth info")
	registrant := fs.String("registrant", "", "registrant contact ID")
	admin := fs.String("contact-admin", "", "admin contact ID")
	tech := fs.String("contact-tech", "", "tech contact ID")
	billing := fs.String("contact-billing", "", "billing contact ID")

	nsParams := fs.String("ns", "", "comma separated nameservers")
	fee := fs.String("fee", "", "fee amount (requires -currency usually)")
	currency := fs.String("currency", "", "fee currency")

	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp create domain [-period N] [-auth code] [-registrant id] ... <domain>")
		os.Exit(1)
	}

	domain := fs.Arg(0)

	contacts := make(map[string]string)
	if *admin != "" {
		contacts["admin"] = *admin
	}
	if *tech != "" {
		contacts["tech"] = *tech
	}
	if *billing != "" {
		contacts["billing"] = *billing
	}

	var ns []string
	if *nsParams != "" {
		ns = strings.Split(*nsParams, ",")
		for i, n := range ns {
			ns[i] = strings.TrimSpace(n)
		}
	}

	var extData map[string]string
	if *fee != "" {
		extData = make(map[string]string)
		extData["fee:fee"] = *fee
		if *currency != "" {
			extData["fee:currency"] = *currency
		}
	}

	res, err := c.CreateDomain(domain, *period, "y", *auth, *registrant, contacts, ns, extData)
	fatalif(err)
	color.Printf("@{g}Domain %s created!\nCreated: %s\nExpiry: %s\n", res.Domain, res.CrDate, res.ExDate)
}

func runCreateContact(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("create contact", flag.ExitOnError)
	id := fs.String("id", "", "contact ID")
	email := fs.String("email", "", "email address")
	name := fs.String("name", "", "contact name")
	org := fs.String("org", "", "organization")
	street := fs.String("street", "", "street address")
	city := fs.String("city", "", "city")
	sp := fs.String("sp", "", "state/province")
	pc := fs.String("pc", "", "postal code")
	cc := fs.String("cc", "", "country code")
	voice := fs.String("voice", "", "voice phone number")
	auth := fs.String("auth", "", "auth info")

	fs.Parse(args)

	if *id == "" || *email == "" || *name == "" || *city == "" || *cc == "" || *auth == "" {
		fmt.Fprintln(os.Stderr, "Usage: epp create contact -id <id> -email <email> -name <name> -city <city> -cc <cc> -auth <auth> [-voice number] [options]")
		fs.PrintDefaults()
		os.Exit(1)
	}

	pi := epp.PostalInfo{
		Name:   *name,
		Org:    *org,
		Street: *street,
		City:   *city,
		SP:     *sp,
		PC:     *pc,
		CC:     *cc,
	}

	res, err := c.CreateContact(*id, *email, pi, *voice, *auth, nil)
	fatalif(err)
	color.Printf("@{g}Contact %s created!\nCreated: %s\n", res.ID, res.CrDate)
}

func runCreateHost(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("create host", flag.ExitOnError)
	ips := fs.String("ips", "", "comma separated IPv4 addresses")
	v6 := fs.String("v6", "", "comma separated IPv6 addresses")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp create host [-ips v4,v4] [-v6 v6,v6] <host>")
		os.Exit(1)
	}

	host := fs.Arg(0)
	var ipList, v6List []string

	if *ips != "" {
		ipList = strings.Split(*ips, ",")
		for i, v := range ipList {
			ipList[i] = strings.TrimSpace(v)
		}
	}
	if *v6 != "" {
		v6List = strings.Split(*v6, ",")
		for i, v := range v6List {
			v6List[i] = strings.TrimSpace(v)
		}
	}

	res, err := c.CreateHost(host, ipList, v6List)
	fatalif(err)
	color.Printf("@{g}Host %s created!\nCreated: %s\n", res.Host, res.CrDate)
}

func runRenew(c *epp.Conn, args []string) {
	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "domain":
		runRenewDomain(c, subArgs)
	default:
		// Fallback or error?
		// Since "epp renew domain" is what we want, we should enforce it?
		// But existing "epp renew" was convenient.
		// "epp renew domain.com" was the old way.
		// If cmd doesn't look like a subcommand (no "domain" keyword), maybe assume old behavior?
		// But cleaner to be strict if we are standardizing.
		// "unknown command %s".
		fmt.Fprintf(os.Stderr, "Unknown renewal type: %s. Use 'domain'.\n", cmd)
		os.Exit(1)
	}
}

func runRenewDomain(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("renew domain", flag.ExitOnError)
	period := fs.Int("period", 1, "renewal period in years")
	curExp := fs.String("exp", "", "current expiry date (YYYY-MM-DD) - optional, will be fetched if not provided")
	fee := fs.String("fee", "", "fee amount")
	currency := fs.String("currency", "", "fee currency")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp renew domain [-exp YYYY-MM-DD] [-period N] [-fee amount] [-currency code] <domain>")
		os.Exit(1)
	}

	domain := fs.Arg(0)
	var date time.Time
	var err error

	if *curExp != "" {
		date, err = time.Parse("2006-01-02", *curExp)
		fatalif(err)
	} else {
		// Auto-fetch expiry if not provided
		fmt.Printf("Fetching info for %s to determine current expiry date...\n", domain)
		infoRes, err := c.DomainInfo(domain, nil)
		fatalif(err)
		date = infoRes.ExDate
		fmt.Printf("Current expiry: %s\n", date.Format("2006-01-02"))
	}

	var extData map[string]string
	if *fee != "" {
		extData = make(map[string]string)
		extData["fee:fee"] = *fee
		if *currency != "" {
			extData["fee:currency"] = *currency
		}
	}

	res, err := c.RenewDomain(domain, date, *period, "y", extData)
	fatalif(err)
	color.Printf("@{g}Domain %s renewed!\nNew Expiry: %s\n", res.Domain, res.ExDate)
}

func runRestore(c *epp.Conn, args []string) {
	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "domain":
		runRestoreDomain(c, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown restore type: %s. Use 'domain'.\n", cmd)
		os.Exit(1)
	}
}

func runRestoreDomain(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp restore domain <domain>")
		os.Exit(1)
	}
	// Restore often requires RGP extension
	// TODO: Add support for reporting data if required by registry?
	// For now, simple RGP restore request.
	_, err := c.RestoreDomain(args[0], nil)
	fatalif(err)
	color.Printf("@{g}Domain %s restored!\n", args[0])
}

func runTransfer(c *epp.Conn, args []string) {
	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "domain":
		runTransferDomain(c, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown transfer type: %s. Use 'domain'.\n", cmd)
		os.Exit(1)
	}
}

func runPoll(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("poll", flag.ExitOnError)
	ack := fs.String("ack", "", "acknowledge message ID")
	fs.Parse(args)

	if *ack != "" {
		res, err := c.PollAck(*ack)
		fatalif(err)
		color.Printf("@{g}Message %s acknowledged.\n", *ack)
		if res.Count > 0 {
			color.Printf("@{.}Messages remaining: %d\n", res.Count)
		}
		return
	}

	res, err := c.PollReq()
	fatalif(err)

	if res.ID == "" {
		color.Println("@{y}No messages in queue.")
		return
	}

	color.Printf("@{g}Message ID: %s\n", res.ID)
	color.Printf("Count: %d\n", res.Count)
	color.Printf("Date: %s\n", res.Date.Format(time.RFC3339))
	color.Printf("Message: %s\n", res.Message)
}

func runTransferDomain(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("transfer domain", flag.ExitOnError)
	op := fs.String("op", "query", "transfer operation (query, request, approve, reject, cancel)")
	auth := fs.String("auth", "", "auth info")
	period := fs.Int("period", 1, "registration period in years (optional for request)")
	fee := fs.String("fee", "", "fee amount")
	currency := fs.String("currency", "", "fee currency")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp transfer domain [-op op] [-auth code] [-period N] [-fee amount] [-currency code] <domain>")
		os.Exit(1)
	}

	domain := fs.Arg(0)

	var extData map[string]string
	if *fee != "" {
		extData = make(map[string]string)
		extData["fee:fee"] = *fee
		if *currency != "" {
			extData["fee:currency"] = *currency
		}
	}

	res, err := c.TransferDomain(*op, domain, *period, "y", *auth, extData)
	fatalif(err)

	color.Printf("@{g}Domain %s %s operation successful!\n", domain, *op)
	if res != nil {
		fmt.Printf("Status: %s\n", res.Status)
		if !res.REDate.IsZero() {
			fmt.Printf("Requested: %s by %s\n", res.REDate.Format(time.RFC3339), res.REID)
		}
		if !res.ACDate.IsZero() {
			fmt.Printf("Acted: %s by %s\n", res.ACDate.Format(time.RFC3339), res.ACID)
		}
		if !res.ExDate.IsZero() {
			fmt.Printf("Expiry: %s\n", res.ExDate.Format(time.RFC3339))
		}
	}
}

func runRaw(c *epp.Conn, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp raw <file>")
		os.Exit(1)
	}

	var data []byte
	var err error

	filename := args[0]
	if filename == "-" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(filename)
	}
	fatalif(err)

	res, err := c.Raw(data)
	fatalif(err)

	fmt.Printf("%s\n", string(res))
}

func runUpdate(c *epp.Conn, args []string) {
	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "domain":
		runUpdateDomain(c, subArgs)
	case "contact":
		runUpdateContact(c, subArgs)
	case "host":
		runUpdateHost(c, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown update type: %s. Use 'domain', 'contact' or 'host'.\n", cmd)
		os.Exit(1)
	}
}

func runUpdateDomain(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("update domain", flag.ExitOnError)
	addNS := fs.String("add-ns", "", "comma separated nameservers to add")
	remNS := fs.String("rem-ns", "", "comma separated nameservers to remove")
	addStatus := fs.String("add-status", "", "comma separated status codes to add (key=value or just key)")
	remStatus := fs.String("rem-status", "", "comma separated status codes to remove")
	registrant := fs.String("chg-registrant", "", "new registrant ID")
	auth := fs.String("chg-auth", "", "new auth info")

	// Contacts
	addAdmin := fs.String("add-admin", "", "admin contact to add")
	addTech := fs.String("add-tech", "", "tech contact to add")
	addBilling := fs.String("add-billing", "", "billing contact to add")
	remAdmin := fs.String("rem-admin", "", "admin contact to remove")
	remTech := fs.String("rem-tech", "", "tech contact to remove")
	remBilling := fs.String("rem-billing", "", "billing contact to remove")

	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp update domain [options] <domain>")
		fs.PrintDefaults()
		os.Exit(1)
	}

	domain := fs.Arg(0)

	add := make(map[string]interface{})
	rem := make(map[string]interface{})
	chg := make(map[string]string)

	// NS
	if *addNS != "" {
		add["ns"] = parseList(*addNS)
	}
	if *remNS != "" {
		rem["ns"] = parseList(*remNS)
	}

	// Status
	if *addStatus != "" {
		add["status"] = parseMap(*addStatus)
	}
	if *remStatus != "" {
		rem["status"] = parseMap(*remStatus)
	}

	// Contacts
	addContacts := make(map[string]string)
	if *addAdmin != "" {
		addContacts["admin"] = *addAdmin
	}
	if *addTech != "" {
		addContacts["tech"] = *addTech
	}
	if *addBilling != "" {
		addContacts["billing"] = *addBilling
	}
	if len(addContacts) > 0 {
		add["contacts"] = addContacts
	}

	remContacts := make(map[string]string)
	if *remAdmin != "" {
		remContacts["admin"] = *remAdmin
	}
	if *remTech != "" {
		remContacts["tech"] = *remTech
	}
	if *remBilling != "" {
		remContacts["billing"] = *remBilling
	}
	if len(remContacts) > 0 {
		rem["contacts"] = remContacts
	}

	// Chg
	if *registrant != "" {
		chg["registrant"] = *registrant
	}
	if *auth != "" {
		chg["auth"] = *auth
	}

	err := c.UpdateDomain(domain, add, rem, chg)
	fatalif(err)
	color.Printf("@{g}Domain %s updated!\n", domain)
}

func runUpdateContact(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("update contact", flag.ExitOnError)
	addStatus := fs.String("add-status", "", "comma separated status codes to add")
	remStatus := fs.String("rem-status", "", "comma separated status codes to remove")

	name := fs.String("chg-name", "", "new name")
	org := fs.String("chg-org", "", "new organization")
	street := fs.String("chg-street", "", "new street")
	city := fs.String("chg-city", "", "new city")
	sp := fs.String("chg-sp", "", "new state/province")
	pc := fs.String("chg-pc", "", "new postal code")
	cc := fs.String("chg-cc", "", "new country code")

	email := fs.String("chg-email", "", "new email")
	voice := fs.String("chg-voice", "", "new voice")
	fax := fs.String("chg-fax", "", "new fax")
	auth := fs.String("chg-auth", "", "new auth info")

	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp update contact [options] <contact-id>")
		fs.PrintDefaults()
		os.Exit(1)
	}

	id := fs.Arg(0)

	add := make(map[string]interface{})
	rem := make(map[string]interface{})
	chg := make(map[string]interface{})

	if *addStatus != "" {
		add["status"] = parseMap(*addStatus)
	}
	if *remStatus != "" {
		rem["status"] = parseMap(*remStatus)
	}

	// Postal change
	if *name != "" || *org != "" || *street != "" || *city != "" || *sp != "" || *pc != "" || *cc != "" {
		// This is tricky because we don't know if we need to set existing values or if partial update is allowed by EPP lib helper
		// Our helper takes PostalInfo.
		// EPP requires full postal info replacement usually if changing any checking.
		// But let's assume user provides what they want to change, and we might strictly need all if RFC says so.
		// RFC 5733 says: <contact:chg> contains <contact:postalInfo>.
		// Ideally we should fetch existing info first, but that's expensive.
		// For now let's construct what we have. API consumer should know better.
		pi := epp.PostalInfo{
			Name:   *name,
			Org:    *org,
			Street: *street,
			City:   *city,
			SP:     *sp,
			PC:     *pc,
			CC:     *cc,
		}
		chg["postal"] = pi
	}

	if *email != "" {
		chg["email"] = *email
	}
	if *voice != "" {
		chg["voice"] = *voice
	}
	if *fax != "" {
		chg["fax"] = *fax
	}
	if *auth != "" {
		chg["auth"] = *auth
	}

	err := c.UpdateContact(id, add, rem, chg)
	fatalif(err)
	color.Printf("@{g}Contact %s updated!\n", id)
}

func runUpdateHost(c *epp.Conn, args []string) {
	fs := flag.NewFlagSet("update host", flag.ExitOnError)
	addIPs := fs.String("add-ips", "", "comma separated IPv4 to add")
	addV6 := fs.String("add-v6", "", "comma separated IPv6 to add")
	remIPs := fs.String("rem-ips", "", "comma separated IPv4 to remove")
	remV6 := fs.String("rem-v6", "", "comma separated IPv6 to remove")
	addStatus := fs.String("add-status", "", "comma separated status to add")
	remStatus := fs.String("rem-status", "", "comma separated status to remove")
	newName := fs.String("chg-name", "", "new host name")

	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: epp update host [options] <host>")
		fs.PrintDefaults()
		os.Exit(1)
	}

	host := fs.Arg(0)

	add := make(map[string]interface{})
	rem := make(map[string]interface{})
	chg := make(map[string]interface{})

	if *addIPs != "" {
		add["ips"] = parseList(*addIPs)
	}
	if *addV6 != "" {
		add["v6"] = parseList(*addV6)
	}
	if *addStatus != "" {
		add["status"] = parseMap(*addStatus)
	}

	if *remIPs != "" {
		rem["ips"] = parseList(*remIPs)
	}
	if *remV6 != "" {
		rem["v6"] = parseList(*remV6)
	}
	if *remStatus != "" {
		rem["status"] = parseMap(*remStatus)
	}

	if *newName != "" {
		chg["name"] = *newName
	}

	err := c.UpdateHost(host, add, rem, chg)
	fatalif(err)
	color.Printf("@{g}Host %s updated!\n", host)
}

func parseList(s string) []string {
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func parseMap(s string) map[string]string {
	m := make(map[string]string)
	parts := strings.Split(s, ",")
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		k := strings.TrimSpace(kv[0])
		if len(kv) == 2 {
			m[k] = strings.TrimSpace(kv[1])
		} else {
			m[k] = "" // No value, e.g. status without message
		}
	}
	return m
}

func logif(err error) bool {
	if err != nil {
		color.Fprintf(os.Stderr, "@{r}%s\n", err)
		return true
	}
	return false
}

func fatalif(err error) {
	if logif(err) {
		// Panic ensuring we can recover in main to show logs if needed
		panic(err)
	}
}

func printDCR(dcr *epp.DomainCheckResponse) {
	if dcr == nil {
		return
	}

	// Map domain availability for quick lookup when printing fees
	avData := make(map[string]struct {
		Available bool
		Reason    string
	})
	for _, c := range dcr.Checks {
		avData[c.Domain] = struct {
			Available bool
			Reason    string
		}{c.Available, c.Reason}

		if c.Available {
			color.Printf("%-30s @{g}available", c.Domain)
		} else {
			color.Printf("%-30s @{y}unavailable", c.Domain)
		}

		if c.Reason != "" {
			color.Printf(" @{.}reason=%q", c.Reason)
		}
		color.Println()
	}

	// Print Fee details if present
	if len(dcr.Charges) > 0 {
		color.Println("\n@{.}Fees & Pricing:")
		for _, c := range dcr.Charges {
			if len(c.Fees) == 0 && c.Category == "" {
				continue
			}

			if avData[c.Domain].Available {
				color.Printf("  @{g}%s", c.Domain)
			} else {
				color.Printf("  @{y}%s", c.Domain)
			}

			if c.Category != "" {
				color.Printf(" @{.}category=%s", c.Category)
			}
			if c.Currency != "" {
				color.Printf(" @{.}currency=%s", c.Currency)
			}
			color.Println()

			for _, f := range c.Fees {
				color.Printf("    %-10s @{w}%8s", f.Name, f.Amount)

				attrs := []string{}
				if f.Standard {
					attrs = append(attrs, "standard")
				}
				if f.Refundable {
					attrs = append(attrs, "refundable")
				}
				if f.GracePeriod != "" {
					attrs = append(attrs, fmt.Sprintf("grace=%s", f.GracePeriod))
				}
				if f.Description != "" {
					attrs = append(attrs, fmt.Sprintf("desc=%q", f.Description))
				}

				if len(attrs) > 0 {
					color.Printf("  @{.}%s", strings.Join(attrs, " "))
				}
				color.Println()
			}
		}
	}
}
