package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wsxiaoys/terminal/color"
)

func newTransferCmd() *cobra.Command {
	var op, auth, fee, currency string
	var period int

	domain := &cobra.Command{
		Use:   "domain <name>",
		Short: "Query, request, approve, reject or cancel a domain transfer",
		Example: `  epp transfer domain geoff.test
  epp transfer domain geoff.test --op request --auth 'secret'`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c := session()
			name := args[0]

			var extData map[string]string
			if fee != "" {
				extData = map[string]string{"fee:fee": fee}
				if currency != "" {
					extData["fee:currency"] = currency
				}
			}

			if op == "request" {
				confirmOrAbort(fmt.Sprintf("Request transfer of %s?", name), true)
			}
			res, err := c.TransferDomain(op, name, period, "y", auth, extData)
			fatalif(err)

			color.Printf("@{g}Domain %s %s operation successful!\n", name, op)
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
		},
	}
	f := domain.Flags()
	f.StringVar(&op, "op", "query", "transfer operation (query, request, approve, reject, cancel)")
	f.StringVar(&auth, "auth", "", "domain authInfo")
	f.IntVar(&period, "period", 1, "period in years (request only)")
	f.StringVar(&fee, "fee", "", "fee amount")
	f.StringVar(&currency, "currency", "", "fee currency")
	_ = domain.RegisterFlagCompletionFunc("op", cobra.FixedCompletions(
		[]string{"query", "request", "approve", "reject", "cancel"}, cobra.ShellCompDirectiveNoFileComp))

	cmd := newGroupCmd("transfer", "Transfer a domain")
	cmd.AddCommand(domain)
	return cmd
}
