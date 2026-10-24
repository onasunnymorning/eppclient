package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"unicode"

	epp "github.com/onasunnymorning/eppclient"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
)

// opts holds the flags shared by every command.
var opts struct {
	profile string
	verbose bool
	yes     bool
	xml     bool
}

// logBuf captures the raw XML exchanged during this run.
var logBuf bytes.Buffer

func main() {
	os.Exit(run(normalizeLegacyFlags(os.Args[1:])))
}

func run(args []string) (code int) {
	logBuf.Reset()
	cfg = nil
	root := newRootCmd()
	root.SetArgs(args)

	defer func() {
		offerXML := code != 0
		if r := recover(); r != nil {
			switch r.(type) {
			case fatalError:
				code, offerXML = 1, true
			case abortError:
				code, offerXML = 130, false
			default:
				closeSession()
				panic(r)
			}
		}
		closeSession()
		finishXMLLog(offerXML)
	}()

	if err := root.Execute(); err != nil {
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "epp",
		Short: "Send EPP commands to a registry from the command line",
		Long: `Send EPP commands to a registry from the command line.

Commands fill in sensible defaults for anything you leave out (authInfo,
contacts, period) and ask only what they can't guess. Before anything
billable or destructive is sent you get a summary to confirm.

Use -y to accept all defaults without prompts. When input is not a
terminal (scripts, CI), commands never prompt.

Credentials and per-profile defaults live in ~/.epp/credentials.`,
		Example: `  epp check geoff.test
  epp create domain geoff.test
  epp create domain geoff.test --period 2 --ns ns1.example.net,ns2.example.net
  epp --profile ote create domain geoff.test -y
  epp info domain geoff.test --xml`,
		// Usage is printed for argument and flag errors only: command failures
		// exit through fatalif instead of returning an error.
		SilenceUsage: false,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if opts.verbose {
				epp.DebugLogger = io.MultiWriter(os.Stderr, &logBuf)
			} else {
				epp.DebugLogger = &logBuf
			}
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&opts.profile, "profile", "default", "profile name in ~/.epp/credentials")
	pf.BoolVarP(&opts.verbose, "verbose", "v", false, "print raw XML as it is sent and received")
	pf.BoolVarP(&opts.yes, "yes", "y", false, "accept defaults; skip questions and confirmations")
	pf.BoolVar(&opts.xml, "xml", false, "print the raw XML of the session when done")
	_ = root.RegisterFlagCompletionFunc("profile", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return listProfiles(), cobra.ShellCompDirectiveNoFileComp
	})

	root.AddCommand(
		newCheckCmd(),
		newInfoCmd(),
		newCreateCmd(),
		newRenewCmd(),
		newRestoreCmd(),
		newDeleteCmd(),
		newTransferCmd(),
		newUpdateCmd(),
		newPollCmd(),
		newRawCmd(),
		newVersionCmd(),
	)
	return root
}

// normalizeLegacyFlags rewrites single-dash long flags (-period 2) to the
// double-dash form (--period 2), so commands written for the older CLI keep
// working. Short flags (-v, -y, -h, -vy) and negative numbers are left alone.
func normalizeLegacyFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for i, a := range args {
		if a == "--" {
			return append(out, args[i:]...)
		}
		if isLegacyLongFlag(a) {
			a = "-" + a
		}
		out = append(out, a)
	}
	return out
}

func isLegacyLongFlag(a string) bool {
	if len(a) < 3 || a[0] != '-' || a[1] == '-' || !unicode.IsLetter(rune(a[1])) {
		return false
	}
	name := a[1:]
	if i := strings.IndexByte(name, '='); i >= 0 {
		name = name[:i]
	}
	return strings.Trim(name, "vyh") != ""
}

// newGroupCmd returns a parent command such as "create". Without arguments
// it shows help; an unknown subcommand is an error.
func newGroupCmd(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}
