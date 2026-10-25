package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wsxiaoys/terminal/color"
)

func newRenewCmd() *cobra.Command {
	var period int
	var curExp, fee, currency string

	domain := &cobra.Command{
		Use:   "domain <name>",
		Short: "Renew a domain (fetches the current expiry date if --exp is omitted)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c := session()
			name := args[0]

			var date time.Time
			var err error
			if curExp != "" {
				date, err = time.Parse("2006-01-02", curExp)
				fatalif(err)
			} else {
				fmt.Printf("Fetching info for %s to determine current expiry date...\n", name)
				info, err := c.DomainInfo(name, nil)
				fatalif(err)
				date = info.ExDate
				fmt.Printf("Current expiry: %s\n", date.Format("2006-01-02"))
			}

			var extData map[string]string
			if fee != "" {
				extData = map[string]string{"fee:fee": fee}
				if currency != "" {
					extData["fee:currency"] = currency
				}
			}

			confirmOrAbort(fmt.Sprintf("Renew %s for %dy?", name, period), true)
			res, err := c.RenewDomain(name, date, period, "y", extData)
			fatalif(err)
			color.Printf("@{g}Domain %s renewed!\nNew Expiry: %s\n", res.Domain, res.ExDate)
		},
	}
	f := domain.Flags()
	f.IntVar(&period, "period", 1, "renewal period in years")
	f.StringVar(&curExp, "exp", "", "current expiry date (YYYY-MM-DD); fetched if omitted")
	f.StringVar(&fee, "fee", "", "fee amount")
	f.StringVar(&currency, "currency", "", "fee currency")

	cmd := newGroupCmd("renew", "Renew a domain")
	cmd.AddCommand(domain)
	return cmd
}

func newRestoreCmd() *cobra.Command {
	cmd := newGroupCmd("restore", "Restore a deleted domain (RGP)")
	cmd.AddCommand(&cobra.Command{
		Use:   "domain <name>",
		Short: "Restore a domain in its redemption grace period",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c := session()
			confirmOrAbort("Restore "+args[0]+"?", true)
			_, err := c.RestoreDomain(args[0], nil)
			fatalif(err)
			color.Printf("@{g}Domain %s restored!\n", args[0])
		},
	})
	return cmd
}
