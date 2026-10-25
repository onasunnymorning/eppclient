package main

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	epp "github.com/onasunnymorning/eppclient"
	"github.com/spf13/cobra"
	"github.com/wsxiaoys/terminal/color"
)

func newCreateCmd() *cobra.Command {
	cmd := newGroupCmd("create", "Create a domain, contact or host")
	cmd.AddCommand(newCreateDomainCmd(), newCreateContactCmd(), newCreateHostCmd())
	return cmd
}

// ---------------------------------------------------------------------------
// create domain

type domainCreateFlags struct {
	period                           int
	auth, ns, phase, fee, currency   string
	registrant, admin, tech, billing string
	contacts                         string // --contacts: roles to send

	// given reports whether a flag was passed, even with an empty value:
	// --ns '' means "no nameservers", not "use the profile default".
	given func(name string) bool
}

// domainCreate is what will be sent, plus notes on where values came from.
type domainCreate struct {
	name     string
	period   int
	auth     string
	contacts map[string]string // role -> contact ID, for every role that is sent
	policies policySet         // contact policy per role
	ns       []string
	phase    string
	fee      string // sent with the create
	currency string
	price    string            // from the fee check, shown only
	notes    map[string]string // field -> "generated", "profile default", ...

	savedContact string // default contact taken from ~/.epp/state.json, if any
	defaultNS    bool   // the profile has default nameservers
}

var contactTypes = []string{"admin", "tech", "billing"}

func newCreateDomainCmd() *cobra.Command {
	var fl domainCreateFlags
	cmd := &cobra.Command{
		Use:   "domain <name>",
		Short: "Register a domain, filling in defaults for anything left out",
		Long: `Register a domain. Anything you leave out gets a default:

  period       1 year
  authInfo     a random password
  contacts     roles the contact policy marks required are sent with the
               profile's default contact (contact = ... in ~/.epp/credentials).
               Without one, you're asked once to create a test contact or give
               an existing ID, and the answer is remembered for the profile.
  nameservers  the profile's ns = ... setting, or none
  fee          from a fee check, when the registry supports fee-1.0

Contact policy says per role (registrant, admin, tech, billing) whether it
is required, optional or prohibited. It comes from, in increasing priority:
the built-in default (registrant required, the rest optional), the
registry's greeting (no contact object: all prohibited), contacts = ... in the
profile, and [tld] sections in ~/.epp/tlds. Optional and prohibited roles are
left out unless you ask for them with --contacts or a role flag.

You then get a summary to confirm or edit before anything is sent.
With -y the defaults are used without questions.`,
		Example: `  epp create domain geoff.test
  epp create domain geoff.test --period 2 --ns ns1.example.net,ns2.example.net
  epp create domain geoff.test --registrant c-123 --auth 'S3cret!pw' -y
  epp create domain geoff.test --phase sunrise --fee 100.00 --currency USD
  epp create domain geoff.test --contacts none
  epp create domain geoff.test --contacts registrant,tech
  epp create domain geoff.test --contacts all --billing c-99`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fl.given = func(name string) bool {
				return cmd.Flags().Changed(name) || cmd.Flags().Changed("contact-"+name)
			}
			runCreateDomain(args[0], &fl)
		},
	}

	f := cmd.Flags()
	f.IntVar(&fl.period, "period", 1, "registration period in years")
	f.StringVar(&fl.auth, "auth", "", "authInfo (default: generated)")
	f.StringVar(&fl.contacts, "contacts", "", "contact roles to send: a list of registrant,admin,tech,billing, or all, or none (default: from contact policy)")
	f.StringVar(&fl.registrant, "registrant", "", "registrant contact ID; sends the role ('' leaves it out)")
	f.StringVar(&fl.admin, "admin", "", "admin contact ID; sends the role ('' leaves it out)")
	f.StringVar(&fl.tech, "tech", "", "tech contact ID; sends the role ('' leaves it out)")
	f.StringVar(&fl.billing, "billing", "", "billing contact ID; sends the role ('' leaves it out)")
	_ = cmd.RegisterFlagCompletionFunc("contacts", cobra.FixedCompletions(
		append([]string{"none", "all"}, contactRoles...), cobra.ShellCompDirectiveNoFileComp))
	f.StringVar(&fl.ns, "ns", "", "comma separated nameservers (default: profile ns)")
	f.StringVar(&fl.phase, "phase", "", "launch phase (e.g., sunrise)")
	f.StringVar(&fl.fee, "fee", "", "fee to accept (default: from fee check)")
	f.StringVar(&fl.currency, "currency", "", "fee currency")

	// Flag names from the previous CLI.
	f.StringVar(&fl.admin, "contact-admin", "", "admin contact ID")
	f.StringVar(&fl.tech, "contact-tech", "", "tech contact ID")
	f.StringVar(&fl.billing, "contact-billing", "", "billing contact ID")
	for _, name := range []string{"contact-admin", "contact-tech", "contact-billing"} {
		_ = f.MarkHidden(name)
	}
	return cmd
}

func runCreateDomain(name string, fl *domainCreateFlags) {
	c := session()

	d := &domainCreate{
		name:     name,
		period:   fl.period,
		auth:     fl.auth,
		phase:    fl.phase,
		fee:      fl.fee,
		currency: fl.currency,
		contacts: map[string]string{},
		notes:    map[string]string{},
	}

	if d.auth == "" {
		d.auth = genAuthInfo()
		d.notes["auth"] = "generated"
	}
	d.defaultNS = len(cfg.NS) > 0
	if fl.given("ns") {
		d.ns = parseList(fl.ns)
	} else if d.defaultNS {
		d.ns = cfg.NS
		d.notes["ns"] = "profile default"
	}
	if d.fee != "" {
		d.notes["fee"] = "--fee"
	}

	d.setContacts(c, map[string]string{
		"registrant": fl.registrant,
		"admin":      fl.admin,
		"tech":       fl.tech,
		"billing":    fl.billing,
	}, fl)

	d.check(c, true)

	title := fmt.Sprintf("Create domain %s  (profile: %s)", d.name, opts.profile)
	for {
		printReview(title, d.rows())
		if !interactive() || reviewChoice("Yes, create it") == reviewProceed {
			break
		}
		if d.edit() {
			// Period changed: the price we had may no longer apply.
			if d.notes["fee"] == "fee check" {
				d.fee, d.currency = "", ""
				delete(d.notes, "fee")
			}
			d.price = ""
			d.check(c, false)
		}
	}

	var ext map[string]string
	if d.fee != "" || d.phase != "" {
		ext = map[string]string{}
		if d.fee != "" {
			ext["fee:fee"] = d.fee
			if d.currency != "" {
				ext["fee:currency"] = d.currency
			}
		}
		if d.phase != "" {
			ext["launch:phase"] = d.phase
		}
	}

	others := map[string]string{}
	for _, t := range contactTypes {
		if id := d.contacts[t]; id != "" {
			others[t] = id
		}
	}
	res, err := c.CreateDomain(d.name, d.period, "y", d.auth, d.contacts["registrant"], others, d.ns, ext)
	if err != nil {
		logif(err)
		if d.savedContact != "" {
			warnf("If contact %s no longer exists, run `epp create contact --default` to make a new default.", d.savedContact)
		}
		panic(fatalError{err})
	}

	color.Printf("@{g}Domain %s created!\n", res.Domain)
	fmt.Printf("Created:  %s\n", res.CrDate)
	fmt.Printf("Expiry:   %s\n", res.ExDate)
	fmt.Printf("AuthInfo: %s\n", d.auth)
	printEquivalent(d.args()...)
}

// setContacts decides which contact roles to send, and with which IDs. A
// role given as a flag is sent with that ID (an empty flag leaves it out).
// Otherwise --contacts picks the roles, or else the contact policy does:
// required roles are sent. Roles sent without an ID get the default contact.
func (d *domainCreate) setContacts(c *epp.Conn, ids map[string]string, fl *domainCreateFlags) {
	tlds, err := loadTLDPolicies()
	fatalif(err)
	d.policies = contactPolicies(d.name, usesContacts(c), cfg.Contacts, tlds)

	var include map[string]bool
	if fl.given("contacts") {
		include, err = parseRoleList(fl.contacts)
		fatalif(err)
	}

	var needDefault []string
	for _, role := range contactRoles {
		var send bool
		switch {
		case fl.given(role):
			send = ids[role] != ""
		case include != nil:
			send = include[role]
		default:
			send = d.policies[role].policy == required
		}
		switch {
		case !send:
		case ids[role] == "":
			needDefault = append(needDefault, role)
		default:
			d.contacts[role] = ids[role]
		}
	}

	if len(needDefault) > 0 {
		id, source := defaultContact()
		if id == "" {
			id, source = pickDefaultContact(c, needDefault), "new default"
		}
		if source == "saved default" {
			d.savedContact = id
		}
		for _, role := range needDefault {
			d.contacts[role] = id
			d.notes[role] = source
		}
	}

	for _, role := range contactRoles {
		p := d.policies[role]
		sent := d.contacts[role] != ""
		switch {
		case sent && p.policy == prohibited:
			warnf("The %s contact is %s, but will be sent.", role, p)
		case !sent && p.policy == required:
			warnf("The %s contact is %s, but won't be sent.", role, p)
		}
	}
}

// pickDefaultContact is used when the profile has no default contact yet.
// The result is saved so later runs don't ask again.
func pickDefaultContact(c *epp.Conn, roles []string) string {
	if !interactive() {
		if !opts.yes {
			fatalf("profile %q has no default contact (needed for %s).\n"+
				"Pass contact IDs (--registrant, --admin, ...), leave roles out with --contacts,\n"+
				"set contact = <id> in ~/.epp/credentials, or add -y to create a test contact",
				opts.profile, strings.Join(roles, ", "))
		}
		return createTestContact(c)
	}

	choice := choose(
		fmt.Sprintf("Profile %q has no default contact (needed for %s)", opts.profile, strings.Join(roles, ", ")),
		huh.NewOption("Create a test contact and use it from now on", "create"),
		huh.NewOption("Use an existing contact ID from now on", "existing"),
	)
	if choice == "create" {
		return createTestContact(c)
	}

	id := ask("Contact ID", notEmpty)
	if _, err := c.ContactInfo(id, "", nil); err != nil {
		warnf("Could not look up contact %s: %v", id, err)
		if !confirm("Use "+id+" anyway?", false) {
			abort()
		}
	}
	saveContact(opts.profile, id)
	recordChange("saved %s as the default contact for profile %q", id, opts.profile)
	fmt.Fprintf(os.Stderr, "Saved %s as the default contact for profile %q.\n", id, opts.profile)
	return id
}

// createTestContact creates a placeholder contact and saves it as the
// profile's default.
func createTestContact(c *epp.Conn) string {
	tc := newTestContact(cfg.Email)
	_, err := c.CreateContact(tc.ID, tc.Email, tc.Postal, tc.Voice, tc.Auth, nil)
	fatalif(err)
	recordChange("created test contact %s", tc.ID)
	saveContact(opts.profile, tc.ID)
	recordChange("saved %s as the default contact for profile %q", tc.ID, opts.profile)
	color.Fprintf(os.Stderr, "@{g}Created test contact %s@{|} (now the default for profile %q)\n", tc.ID, opts.profile)
	return tc.ID
}

// check warns if the domain is taken and picks up the create price from a
// fee check. confirmTaken asks whether to go on when it's taken.
func (d *domainCreate) check(c *epp.Conn, confirmTaken bool) {
	ext := map[string]string{"fee:period": strconv.Itoa(d.period)}
	if d.phase != "" {
		ext["launch:phase"] = d.phase
		ext["fee:phase"] = d.phase
	}
	dc, err := c.CheckDomainExtensions([]string{d.name}, ext)
	if err != nil {
		warnf("Could not check availability: %v", err)
		return
	}

	for _, chk := range dc.Checks {
		if !strings.EqualFold(chk.Domain, d.name) || chk.Available {
			continue
		}
		msg := d.name + " is not available"
		if chk.Reason != "" {
			msg += " (" + chk.Reason + ")"
		}
		warnf("%s", msg)
		if confirmTaken && interactive() && !confirm("Try to create it anyway?", false) {
			abort()
		}
	}

	amounts, currency := createFees(dc, d.name)
	if len(amounts) == 0 {
		return
	}
	d.price = strings.TrimSpace(sumAmounts(amounts) + " " + currency)
	if len(amounts) > 1 {
		d.price += " (" + strings.Join(amounts, " + ") + ")"
	}

	// A fee from --fee wins. The library sends the fee in fee-1.0 form, and a
	// fee check without a phase prices the registry's default (1 year) only.
	if d.notes["fee"] != "" || !c.Greeting.SupportsExtension(epp.ExtFee10) || (d.period != 1 && d.phase == "") {
		return
	}
	if len(amounts) > 1 {
		// The library sends a single fee element; don't guess how the registry
		// wants several items accepted.
		warnf("The registry quoted %d create fee items (%s); pass --fee to accept them.", len(amounts), d.price)
		return
	}
	d.fee, d.currency = amounts[0], currency
	d.notes["fee"] = "fee check"
}

// createFees returns every create fee item quoted for domain, e.g. an
// application fee and a registration fee, with their currency.
func createFees(dc *epp.DomainCheckResponse, domain string) (amounts []string, currency string) {
	for _, ch := range dc.Charges {
		if ch.Domain != "" && !strings.EqualFold(ch.Domain, domain) {
			continue
		}
		for _, f := range ch.Fees {
			if f.Name == "create" && f.Amount != "" {
				amounts = append(amounts, f.Amount)
				currency = firstNonEmpty(currency, f.Currency, ch.Currency, dc.Currency)
			}
		}
	}
	return amounts, currency
}

// sumAmounts adds decimal amounts, keeping the largest number of decimals.
// If an amount can't be parsed it returns them joined with " + ".
func sumAmounts(amounts []string) string {
	if len(amounts) == 1 {
		return amounts[0]
	}
	total := new(big.Rat)
	decimals := 0
	for _, a := range amounts {
		r, ok := new(big.Rat).SetString(a)
		if !ok {
			return strings.Join(amounts, " + ")
		}
		total.Add(total, r)
		if i := strings.IndexByte(a, '.'); i >= 0 {
			decimals = max(decimals, len(a)-i-1)
		}
	}
	return total.FloatString(decimals)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (d *domainCreate) rows() []reviewRow {
	rows := []reviewRow{{label: "period", value: fmt.Sprintf("%dy", d.period)}}
	rows = append(rows, d.contactRows()...)
	rows = append(rows,
		reviewRow{"nameservers", strings.Join(d.ns, ", "), d.notes["ns"]},
		reviewRow{"authInfo", d.auth, d.notes["auth"]},
	)
	if d.phase != "" {
		rows = append(rows, reviewRow{label: "phase", value: d.phase})
	}
	fee := strings.TrimSpace(d.fee + " " + d.currency)
	if d.fee != "" {
		rows = append(rows, reviewRow{"fee", fee, d.notes["fee"]})
	}
	if d.price != "" && d.price != fee {
		rows = append(rows, reviewRow{"price", d.price, "fee check, not sent"})
	}
	return rows
}

// contactRows shows every role with its policy, including roles not sent.
func (d *domainCreate) contactRows() []reviewRow {
	// A registry or TLD without contacts gets a single line.
	if len(d.contacts) == 0 {
		first := d.policies[contactRoles[0]]
		same := true
		for _, role := range contactRoles {
			same = same && d.policies[role] == first
		}
		if same && first.policy == prohibited {
			return []reviewRow{{"contacts", "none", first.String()}}
		}
	}

	var rows []reviewRow
	for _, role := range contactRoles {
		p := d.policies[role]
		id := d.contacts[role]
		note := p.String()
		switch {
		case id == "" && p.policy != prohibited:
			note += ", not sent"
		case id != "" && p.policy == prohibited:
			note += ", sent anyway"
		}
		if src := d.notes[role]; id != "" && src != "" {
			note += ", " + src
		}
		rows = append(rows, reviewRow{role, id, note})
	}
	return rows
}

// edit lets the user change the values; it reports whether the period changed.
func (d *domainCreate) edit() (periodChanged bool) {
	// First pick which contacts to send, then edit the values.
	var selected []string
	var options []huh.Option[string]
	for _, role := range contactRoles {
		options = append(options, huh.NewOption(fmt.Sprintf("%-10s  %s", role, d.policies[role]), role))
		if d.contacts[role] != "" {
			selected = append(selected, role)
		}
	}
	runForm(huh.NewMultiSelect[string]().
		Title("Contacts to send").
		Description("space toggles, enter continues").
		Options(options...).
		Value(&selected))

	defaultID, _ := defaultContact()
	period := strconv.Itoa(d.period)
	ns := strings.Join(d.ns, ",")
	auth := d.auth

	fields := []huh.Field{
		huh.NewInput().Title("Period (years)").Value(&period).Validate(validPeriod),
	}
	ids := map[string]*string{}
	for _, role := range contactRoles {
		if !slices.Contains(selected, role) {
			continue
		}
		v := d.contacts[role]
		if v == "" {
			v = defaultID
		}
		ids[role] = &v
		fields = append(fields, huh.NewInput().Title(strings.ToUpper(role[:1])+role[1:]+" contact").Value(&v).Validate(notEmpty))
	}
	fields = append(fields,
		huh.NewInput().Title("Nameservers").Description("comma separated").Value(&ns),
		huh.NewInput().Title("authInfo").Value(&auth).Validate(notEmpty),
	)
	runForm(fields...)

	changed := func(key, before, after string) {
		if before != after {
			d.notes[key] = "edited"
		}
	}

	p, _ := strconv.Atoi(period)
	periodChanged = p != d.period
	d.period = p

	for _, role := range contactRoles {
		v := ""
		if id, ok := ids[role]; ok {
			v = strings.TrimSpace(*id)
		}
		changed(role, d.contacts[role], v)
		if v == "" {
			delete(d.contacts, role)
		} else {
			d.contacts[role] = v
		}
	}
	changed("ns", strings.Join(d.ns, ","), ns)
	d.ns = parseList(ns)
	changed("auth", d.auth, auth)
	d.auth = auth
	return periodChanged
}

func validPeriod(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 10 {
		return errors.New("enter a number of years from 1 to 10")
	}
	return nil
}

// args is the equivalent command line for this create. It names the
// contact roles sent, and passes empty nameservers when the profile would
// otherwise fill them in, so a replay doesn't depend on defaults.
func (d *domainCreate) args() []string {
	a := []string{"create", "domain", d.name, "--period", strconv.Itoa(d.period)}
	var sent []string
	for _, role := range contactRoles {
		if d.contacts[role] != "" {
			sent = append(sent, role)
		}
	}
	if len(sent) == 0 {
		a = append(a, "--contacts", "none")
	} else {
		a = append(a, "--contacts", strings.Join(sent, ","))
	}
	for _, role := range sent {
		a = append(a, "--"+role, d.contacts[role])
	}
	if len(d.ns) > 0 || d.defaultNS {
		a = append(a, "--ns", strings.Join(d.ns, ","))
	}
	a = append(a, "--auth", d.auth)
	if d.phase != "" {
		a = append(a, "--phase", d.phase)
	}
	if d.fee != "" {
		a = append(a, "--fee", d.fee)
		if d.currency != "" {
			a = append(a, "--currency", d.currency)
		}
	}
	return a
}

// ---------------------------------------------------------------------------
// create contact

type contactCreateFlags struct {
	id, email, name, org, street, city, sp, pc, cc, voice, auth string
	makeDefault                                                 bool
	given                                                       func(name string) bool
}

func newCreateContactCmd() *cobra.Command {
	var fl contactCreateFlags
	cmd := &cobra.Command{
		Use:   "contact",
		Short: "Create a contact; anything left out gets a test value",
		Long: `Create a contact. Every field is optional: the ID and authInfo are
generated, the email comes from the profile's email = ... setting (or
test@example.com), and the address is a placeholder.

Use --default to make it the default contact for domain creates in this profile.`,
		Example: `  epp create contact
  epp create contact --default -y
  epp create contact --id CID-1 --name "John Doe" --email john@example.com --city "New York" --cc US`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fl.given = cmd.Flags().Changed
			runCreateContact(&fl)
		},
	}
	f := cmd.Flags()
	f.StringVar(&fl.id, "id", "", "contact ID (default: generated)")
	f.StringVar(&fl.name, "name", "", "contact name")
	f.StringVar(&fl.org, "org", "", "organization")
	f.StringVar(&fl.street, "street", "", "street address")
	f.StringVar(&fl.city, "city", "", "city")
	f.StringVar(&fl.sp, "sp", "", "state/province")
	f.StringVar(&fl.pc, "pc", "", "postal code")
	f.StringVar(&fl.cc, "cc", "", "country code")
	f.StringVar(&fl.email, "email", "", "email address (default: profile email)")
	f.StringVar(&fl.voice, "voice", "", "phone number, e.g. +1.5555550100")
	f.StringVar(&fl.auth, "auth", "", "authInfo (default: generated)")
	f.BoolVar(&fl.makeDefault, "default", false, "use as the profile's default contact")
	return cmd
}

func runCreateContact(fl *contactCreateFlags) {
	c := session()

	tc := newTestContact(cfg.Email)
	notes := map[string]string{}
	// key is the flag name. A flag passed empty clears the placeholder.
	use := func(dst *string, flag, key, fallbackNote string) {
		if fl.given(key) {
			*dst = flag
		} else if *dst != "" {
			notes[key] = fallbackNote
		}
	}
	use(&tc.ID, fl.id, "id", "generated")
	use(&tc.Auth, fl.auth, "auth", "generated")
	emailNote := "placeholder"
	if cfg.Email != "" {
		emailNote = "profile email"
	}
	use(&tc.Email, fl.email, "email", emailNote)
	use(&tc.Voice, fl.voice, "voice", "placeholder")
	use(&tc.Postal.Name, fl.name, "name", "placeholder")
	use(&tc.Postal.Org, fl.org, "org", "")
	use(&tc.Postal.Street, fl.street, "street", "placeholder")
	use(&tc.Postal.City, fl.city, "city", "placeholder")
	use(&tc.Postal.SP, fl.sp, "sp", "placeholder")
	use(&tc.Postal.PC, fl.pc, "pc", "placeholder")
	use(&tc.Postal.CC, fl.cc, "cc", "placeholder")

	makeDefault := fl.makeDefault
	for {
		printReview(fmt.Sprintf("Create contact  (profile: %s)", opts.profile), contactRows(tc, notes))
		if !interactive() {
			break
		}
		var extra []huh.Option[string]
		if !makeDefault {
			extra = append(extra, huh.NewOption(
				fmt.Sprintf("Yes, and make it the default contact for profile %q", opts.profile), "default"))
		}
		choice := reviewChoice("Yes, create it", extra...)
		if choice == "default" {
			makeDefault = true
			break
		}
		if choice == reviewProceed {
			break
		}
		editContact(&tc, notes)
	}

	res, err := c.CreateContact(tc.ID, tc.Email, tc.Postal, tc.Voice, tc.Auth, nil)
	fatalif(err)

	color.Printf("@{g}Contact %s created!\n", tc.ID)
	fmt.Printf("Created:  %s\n", res.CrDate)
	fmt.Printf("AuthInfo: %s\n", tc.Auth)
	if makeDefault {
		saveContact(opts.profile, tc.ID)
		fmt.Fprintf(os.Stderr, "Saved %s as the default contact for profile %q.\n", tc.ID, opts.profile)
	}

	// Every field but org has a default, so pass even empty ones: otherwise a
	// replay would put the placeholder back.
	a := []string{"create", "contact"}
	for _, kv := range [][2]string{
		{"--id", tc.ID}, {"--name", tc.Postal.Name}, {"--org", tc.Postal.Org},
		{"--street", tc.Postal.Street}, {"--city", tc.Postal.City},
		{"--sp", tc.Postal.SP}, {"--pc", tc.Postal.PC}, {"--cc", tc.Postal.CC},
		{"--email", tc.Email}, {"--voice", tc.Voice}, {"--auth", tc.Auth},
	} {
		if kv[1] != "" || kv[0] != "--org" {
			a = append(a, kv[0], kv[1])
		}
	}
	if makeDefault {
		a = append(a, "--default")
	}
	printEquivalent(a...)
}

func contactRows(tc testContact, notes map[string]string) []reviewRow {
	rows := []reviewRow{
		{"id", tc.ID, notes["id"]},
		{"name", tc.Postal.Name, notes["name"]},
	}
	if tc.Postal.Org != "" {
		rows = append(rows, reviewRow{"org", tc.Postal.Org, notes["org"]})
	}
	return append(rows,
		reviewRow{"street", tc.Postal.Street, notes["street"]},
		reviewRow{"city", tc.Postal.City, notes["city"]},
		reviewRow{"state/prov", tc.Postal.SP, notes["sp"]},
		reviewRow{"postal code", tc.Postal.PC, notes["pc"]},
		reviewRow{"country", tc.Postal.CC, notes["cc"]},
		reviewRow{"email", tc.Email, notes["email"]},
		reviewRow{"voice", tc.Voice, notes["voice"]},
		reviewRow{"authInfo", tc.Auth, notes["auth"]},
	)
}

func editContact(tc *testContact, notes map[string]string) {
	type field struct {
		title, key string
		value      *string
		required   bool
	}
	fields := []field{
		{"ID", "id", &tc.ID, true},
		{"Name", "name", &tc.Postal.Name, true},
		{"Organization", "org", &tc.Postal.Org, false},
		{"Street", "street", &tc.Postal.Street, false},
		{"City", "city", &tc.Postal.City, true},
		{"State/province", "sp", &tc.Postal.SP, false},
		{"Postal code", "pc", &tc.Postal.PC, false},
		{"Country code", "cc", &tc.Postal.CC, true},
		{"Email", "email", &tc.Email, true},
		{"Voice", "voice", &tc.Voice, false},
		{"authInfo", "auth", &tc.Auth, true},
	}

	before := make([]string, len(fields))
	inputs := make([]huh.Field, len(fields))
	for i, f := range fields {
		before[i] = *f.value
		in := huh.NewInput().Title(f.title).Value(f.value)
		if f.required {
			in = in.Validate(notEmpty)
		}
		inputs[i] = in
	}
	runForm(inputs...)

	for i, f := range fields {
		*f.value = strings.TrimSpace(*f.value)
		if *f.value != before[i] {
			notes[f.key] = "edited"
		}
	}
}

// ---------------------------------------------------------------------------
// create host

func newCreateHostCmd() *cobra.Command {
	var ips, v6 string
	cmd := &cobra.Command{
		Use:     "host <name>",
		Short:   "Create a host (nameserver)",
		Example: `  epp create host ns1.geoff.test --ips 192.0.2.1,192.0.2.2 --v6 2001:db8::1`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			res, err := session().CreateHost(args[0], parseList(ips), parseList(v6))
			fatalif(err)
			color.Printf("@{g}Host %s created!\nCreated: %s\n", res.Host, res.CrDate)
		},
	}
	cmd.Flags().StringVar(&ips, "ips", "", "comma separated IPv4 addresses")
	cmd.Flags().StringVar(&v6, "v6", "", "comma separated IPv6 addresses")
	return cmd
}
