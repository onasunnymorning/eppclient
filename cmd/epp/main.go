package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"unicode"

	epp "github.com/onasunnymorning/eppclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	os.Exit(run(os.Args[1:]))
}

func run(args []string) (code int) {
	logBuf.Reset()
	cfg = nil
	changesMade = nil
	root := newRootCmd()
	root.SetArgs(normalizeLegacyFlags(args, collectBoolFlags(root)))

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
// working. Short flags (-v, -y, -h, -vy) are left alone, and so is the value
// after any flag that takes one: "--auth -Secret1!" keeps its authInfo.
// boolFlags holds the names and shorthands of flags that take no value.
func normalizeLegacyFlags(args []string, boolFlags map[string]bool) []string {
	out := make([]string, 0, len(args))
	takesValue := false
	for i, a := range args {
		if takesValue {
			out = append(out, a)
			takesValue = false
			continue
		}
		if a == "--" {
			return append(out, args[i:]...)
		}
		if !isFlagToken(a) {
			out = append(out, a)
			continue
		}

		dashes := 2
		if a[1] != '-' {
			dashes = 1
		}
		name, _, hasValue := strings.Cut(a[dashes:], "=")
		shortGroup := dashes == 1 && allShorthands(name, boolFlags)
		if dashes == 1 && !shortGroup {
			a = "-" + a
		}
		out = append(out, a)

		// A flag that isn't boolean takes the next argument as its value,
		// unless it was written as --name=value.
		takesValue = !hasValue && !shortGroup && !boolFlags[name]
	}
	return out
}

// isFlagToken reports whether a looks like a flag rather than a value such
// as "-" (stdin) or a negative number.
func isFlagToken(a string) bool {
	if len(a) < 2 || a[0] != '-' {
		return false
	}
	if a[1] == '-' {
		return len(a) > 2
	}
	return unicode.IsLetter(rune(a[1]))
}

// allShorthands reports whether every letter of name is a boolean shorthand,
// as in -v or -vy.
func allShorthands(name string, boolFlags map[string]bool) bool {
	for _, r := range name {
		if !boolFlags[string(r)] {
			return false
		}
	}
	return name != ""
}

// collectBoolFlags returns the names and shorthands of every boolean flag in
// the command tree.
func collectBoolFlags(root *cobra.Command) map[string]bool {
	names := map[string]bool{"help": true, "h": true}
	add := func(f *pflag.Flag) {
		if f.Value.Type() == "bool" {
			names[f.Name] = true
			if f.Shorthand != "" {
				names[f.Shorthand] = true
			}
		}
	}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		c.Flags().VisitAll(add)
		c.PersistentFlags().VisitAll(add)
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return names
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
