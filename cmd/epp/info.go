package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	cmd := newGroupCmd("info", "Show details of a domain or contact")
	cmd.AddCommand(newInfoDomainCmd(), newInfoContactCmd())
	return cmd
}

func newInfoDomainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "domain <name>",
		Short: "Show domain details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			res, err := session().DomainInfo(args[0], nil)
			fatalif(err)

			fmt.Printf("Domain: %s\n", res.Domain)
			fmt.Printf("ROID: %s\n", res.ID)
			fmt.Printf("Status: %v\n", res.Status)
			fmt.Printf("Created: %s\n", res.CrDate)
			fmt.Printf("Expires: %s\n", res.ExDate)
		},
	}
}

func newInfoContactCmd() *cobra.Command {
	var auth string
	cmd := &cobra.Command{
		Use:   "contact <id>",
		Short: "Show contact details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			res, err := session().ContactInfo(args[0], auth, nil)
			fatalif(err)

			fmt.Printf("Contact: %s\n", res.ID)
			fmt.Printf("ROID: %s\n", res.ROID)
			fmt.Printf("Status: %v\n", res.Status)
			fmt.Printf("Email: %s\n", res.Email)
			fmt.Printf("Created: %s\n", res.CrDate)
		},
	}
	cmd.Flags().StringVar(&auth, "auth", "", "contact authInfo (needed for contacts you don't sponsor)")
	return cmd
}
