package main

import (
	epp "github.com/onasunnymorning/eppclient"
	"github.com/spf13/cobra"
	"github.com/wsxiaoys/terminal/color"
)

func newUpdateCmd() *cobra.Command {
	cmd := newGroupCmd("update", "Update a domain, contact or host")
	cmd.AddCommand(newUpdateDomainCmd(), newUpdateContactCmd(), newUpdateHostCmd())
	return cmd
}

func newUpdateDomainCmd() *cobra.Command {
	var addNS, remNS, addStatus, remStatus, registrant, auth string
	var addAdmin, addTech, addBilling, remAdmin, remTech, remBilling string

	cmd := &cobra.Command{
		Use:     "domain <name>",
		Short:   "Update a domain (nameservers, statuses, contacts, registrant, authInfo)",
		Example: `  epp update domain geoff.test --add-ns ns1.example.net,ns2.example.net --rem-ns ns1.old.example`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			add := make(map[string]interface{})
			rem := make(map[string]interface{})
			chg := make(map[string]string)

			if addNS != "" {
				add["ns"] = parseList(addNS)
			}
			if remNS != "" {
				rem["ns"] = parseList(remNS)
			}
			if addStatus != "" {
				add["status"] = parseMap(addStatus)
			}
			if remStatus != "" {
				rem["status"] = parseMap(remStatus)
			}
			if m := nonEmpty(map[string]string{"admin": addAdmin, "tech": addTech, "billing": addBilling}); len(m) > 0 {
				add["contacts"] = m
			}
			if m := nonEmpty(map[string]string{"admin": remAdmin, "tech": remTech, "billing": remBilling}); len(m) > 0 {
				rem["contacts"] = m
			}
			if registrant != "" {
				chg["registrant"] = registrant
			}
			if auth != "" {
				chg["auth"] = auth
			}

			fatalif(session().UpdateDomain(args[0], add, rem, chg))
			color.Printf("@{g}Domain %s updated!\n", args[0])
		},
	}
	f := cmd.Flags()
	f.StringVar(&addNS, "add-ns", "", "comma separated nameservers to add")
	f.StringVar(&remNS, "rem-ns", "", "comma separated nameservers to remove")
	f.StringVar(&addStatus, "add-status", "", "comma separated status codes to add (key=value or just key)")
	f.StringVar(&remStatus, "rem-status", "", "comma separated status codes to remove")
	f.StringVar(&registrant, "chg-registrant", "", "new registrant ID")
	f.StringVar(&auth, "chg-auth", "", "new authInfo")
	f.StringVar(&addAdmin, "add-admin", "", "admin contact to add")
	f.StringVar(&addTech, "add-tech", "", "tech contact to add")
	f.StringVar(&addBilling, "add-billing", "", "billing contact to add")
	f.StringVar(&remAdmin, "rem-admin", "", "admin contact to remove")
	f.StringVar(&remTech, "rem-tech", "", "tech contact to remove")
	f.StringVar(&remBilling, "rem-billing", "", "billing contact to remove")
	return cmd
}

func newUpdateContactCmd() *cobra.Command {
	var addStatus, remStatus string
	var name, org, street, city, sp, pc, cc string
	var email, voice, fax, auth string

	cmd := &cobra.Command{
		Use:   "contact <id>",
		Short: "Update a contact",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			add := make(map[string]interface{})
			rem := make(map[string]interface{})
			chg := make(map[string]interface{})

			if addStatus != "" {
				add["status"] = parseMap(addStatus)
			}
			if remStatus != "" {
				rem["status"] = parseMap(remStatus)
			}

			// RFC 5733 replaces postalInfo as a whole, so pass every field given.
			if name != "" || org != "" || street != "" || city != "" || sp != "" || pc != "" || cc != "" {
				chg["postal"] = epp.PostalInfo{
					Name: name, Org: org, Street: street, City: city, SP: sp, PC: pc, CC: cc,
				}
			}
			for k, v := range map[string]string{"email": email, "voice": voice, "fax": fax, "auth": auth} {
				if v != "" {
					chg[k] = v
				}
			}

			fatalif(session().UpdateContact(args[0], add, rem, chg))
			color.Printf("@{g}Contact %s updated!\n", args[0])
		},
	}
	f := cmd.Flags()
	f.StringVar(&addStatus, "add-status", "", "comma separated status codes to add")
	f.StringVar(&remStatus, "rem-status", "", "comma separated status codes to remove")
	f.StringVar(&name, "chg-name", "", "new name")
	f.StringVar(&org, "chg-org", "", "new organization")
	f.StringVar(&street, "chg-street", "", "new street")
	f.StringVar(&city, "chg-city", "", "new city")
	f.StringVar(&sp, "chg-sp", "", "new state/province")
	f.StringVar(&pc, "chg-pc", "", "new postal code")
	f.StringVar(&cc, "chg-cc", "", "new country code")
	f.StringVar(&email, "chg-email", "", "new email")
	f.StringVar(&voice, "chg-voice", "", "new voice")
	f.StringVar(&fax, "chg-fax", "", "new fax")
	f.StringVar(&auth, "chg-auth", "", "new authInfo")
	return cmd
}

func newUpdateHostCmd() *cobra.Command {
	var addIPs, addV6, remIPs, remV6, addStatus, remStatus, newName string

	cmd := &cobra.Command{
		Use:   "host <name>",
		Short: "Update a host",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			add := make(map[string]interface{})
			rem := make(map[string]interface{})
			chg := make(map[string]interface{})

			if addIPs != "" {
				add["ips"] = parseList(addIPs)
			}
			if addV6 != "" {
				add["v6"] = parseList(addV6)
			}
			if addStatus != "" {
				add["status"] = parseMap(addStatus)
			}
			if remIPs != "" {
				rem["ips"] = parseList(remIPs)
			}
			if remV6 != "" {
				rem["v6"] = parseList(remV6)
			}
			if remStatus != "" {
				rem["status"] = parseMap(remStatus)
			}
			if newName != "" {
				chg["name"] = newName
			}

			fatalif(session().UpdateHost(args[0], add, rem, chg))
			color.Printf("@{g}Host %s updated!\n", args[0])
		},
	}
	f := cmd.Flags()
	f.StringVar(&addIPs, "add-ips", "", "comma separated IPv4 to add")
	f.StringVar(&addV6, "add-v6", "", "comma separated IPv6 to add")
	f.StringVar(&remIPs, "rem-ips", "", "comma separated IPv4 to remove")
	f.StringVar(&remV6, "rem-v6", "", "comma separated IPv6 to remove")
	f.StringVar(&addStatus, "add-status", "", "comma separated status to add")
	f.StringVar(&remStatus, "rem-status", "", "comma separated status to remove")
	f.StringVar(&newName, "chg-name", "", "new host name")
	return cmd
}

// nonEmpty drops entries with empty values.
func nonEmpty(m map[string]string) map[string]string {
	for k, v := range m {
		if v == "" {
			delete(m, k)
		}
	}
	return m
}
