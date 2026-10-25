package main

import (
	"github.com/spf13/cobra"
	"github.com/wsxiaoys/terminal/color"
)

func newDeleteCmd() *cobra.Command {
	cmd := newGroupCmd("delete", "Delete a domain, contact or host")
	cmd.AddCommand(
		&cobra.Command{
			Use:   "domain <name>",
			Short: "Delete a domain",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				c := session()
				confirmOrAbort("Delete domain "+args[0]+"?", false)
				fatalif(c.DeleteDomain(args[0], nil))
				color.Printf("@{g}Domain %s deleted!\n", args[0])
			},
		},
		&cobra.Command{
			Use:   "contact <id>",
			Short: "Delete a contact",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				c := session()
				confirmOrAbort("Delete contact "+args[0]+"?", false)
				fatalif(c.DeleteContact(args[0], nil))
				color.Printf("@{g}Contact %s deleted!\n", args[0])
			},
		},
		&cobra.Command{
			Use:   "host <name>",
			Short: "Delete a host",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				c := session()
				confirmOrAbort("Delete host "+args[0]+"?", false)
				fatalif(c.DeleteHost(args[0]))
				color.Printf("@{g}Host %s deleted!\n", args[0])
			},
		},
	)
	return cmd
}
