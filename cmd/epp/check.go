package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	epp "github.com/onasunnymorning/eppclient"
	"github.com/spf13/cobra"
	"github.com/wsxiaoys/terminal/color"
)

func newCheckCmd() *cobra.Command {
	var phase string
	var period int
	cmd := &cobra.Command{
		Use:   "check <domain>...",
		Short: "Check domain availability and pricing",
		Example: `  epp check geoff.test
  epp check --phase sunrise --period 2 geoff.test other.test`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c := session()
			start := time.Now()

			extData := map[string]string{"fee:period": strconv.Itoa(period)}
			if phase != "" {
				extData["launch:phase"] = phase
				extData["fee:phase"] = phase
			}

			dc, err := c.CheckDomainExtensions(args, extData)
			logif(err)
			printDCR(dc)
			color.Fprintf(os.Stderr, "@{.}Query: %s\n", time.Since(start))
		},
	}
	cmd.Flags().StringVar(&phase, "phase", "", "launch phase (e.g., sunrise, open)")
	cmd.Flags().IntVar(&period, "period", 1, "registration period in years")
	return cmd
}

func printDCR(dcr *epp.DomainCheckResponse) {
	if dcr == nil {
		return
	}

	available := make(map[string]bool)
	for _, c := range dcr.Checks {
		available[c.Domain] = c.Available

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

	if len(dcr.Charges) == 0 {
		return
	}

	color.Println("\n@{.}Fees & Pricing:")
	for _, c := range dcr.Charges {
		if len(c.Fees) == 0 && c.Category == "" {
			continue
		}

		if available[c.Domain] {
			color.Printf("  @{g}%s", c.Domain)
		} else {
			color.Printf("  @{y}%s", c.Domain)
		}

		if c.Category != "" {
			color.Printf(" @{.}category=%s", c.Category)
		}
		color.Println()

		for _, f := range c.Fees {
			currStr := ""
			if c.Currency != "" {
				currStr = " " + c.Currency
			}
			color.Printf("    %-10s @{w}%8s%s", f.Name, f.Amount, currStr)

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
