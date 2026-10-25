package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/wsxiaoys/terminal/color"
)

func newPollCmd() *cobra.Command {
	var ack string
	cmd := &cobra.Command{
		Use:   "poll",
		Short: "Read (or acknowledge) the next poll message",
		Example: `  epp poll
  epp poll --ack 12345`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			c := session()
			if ack != "" {
				res, err := c.PollAck(ack)
				fatalif(err)
				color.Printf("@{g}Message %s acknowledged.\n", ack)
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
		},
	}
	cmd.Flags().StringVar(&ack, "ack", "", "acknowledge this message ID")
	return cmd
}

func newRawCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "raw <file|->",
		Short: "Send raw XML from a file, or from stdin with -",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var data []byte
			var err error
			if args[0] == "-" {
				data, err = io.ReadAll(os.Stdin)
			} else {
				data, err = os.ReadFile(args[0])
			}
			fatalif(err)

			res, err := session().Raw(data)
			fatalif(err)
			fmt.Printf("%s\n", string(res))
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("version.BuildInfo{Version:%q, GitCommit:%q, GoVersion:%q}\n", version, commit, runtime.Version())
		},
	}
}
