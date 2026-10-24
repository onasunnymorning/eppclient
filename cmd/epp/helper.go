package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wsxiaoys/terminal/color"
)

// fatalError is panicked by fatalif and recovered in run(), so cleanup
// (logout, XML log) still happens before exiting with status 1.
type fatalError struct{ err error }

// abortError is panicked when the user cancels a prompt.
type abortError struct{}

func logif(err error) bool {
	if err != nil {
		color.Fprintf(os.Stderr, "@{r}%s\n", err)
		return true
	}
	return false
}

func fatalif(err error) {
	if logif(err) {
		panic(fatalError{err})
	}
}

func fatalf(format string, a ...any) {
	fatalif(fmt.Errorf(format, a...))
}

// changesMade lists what this run already changed (on the registry or in
// ~/.epp), so a later cancel can say so.
var changesMade []string

func recordChange(format string, a ...any) {
	changesMade = append(changesMade, fmt.Sprintf(format, a...))
}

func abort() {
	if len(changesMade) == 0 {
		fmt.Fprintln(os.Stderr, "Cancelled, nothing was changed.")
	} else {
		fmt.Fprintln(os.Stderr, "Cancelled. Already done before cancelling:")
		for _, c := range changesMade {
			fmt.Fprintf(os.Stderr, "  - %s\n", c)
		}
	}
	panic(abortError{})
}

func warnf(format string, a ...any) {
	color.Fprintf(os.Stderr, "@{y}%s\n", fmt.Sprintf(format, a...))
}

// eppDir is ~/.epp, where credentials and state live.
func eppDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".epp"), nil
}

func parseList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseMap(s string) map[string]string {
	m := make(map[string]string)
	for _, p := range strings.Split(s, ",") {
		kv := strings.SplitN(p, "=", 2)
		k := strings.TrimSpace(kv[0])
		if len(kv) == 2 {
			m[k] = strings.TrimSpace(kv[1])
		} else {
			m[k] = "" // No value, e.g. status without message
		}
	}
	return m
}

var shellSafe = regexp.MustCompile(`^[A-Za-z0-9_./:,@=+-]+$`)

func shellQuote(s string) string {
	if shellSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// printEquivalent shows the non-interactive command that does the same
// thing, so a prompted run can be repeated or scripted.
func printEquivalent(args ...string) {
	parts := []string{"epp"}
	if opts.profile != "default" {
		parts = append(parts, "--profile", shellQuote(opts.profile))
	}
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	parts = append(parts, "-y")
	color.Fprintf(os.Stderr, "\n@{.}To repeat without prompts:\n  %s\n", strings.Join(parts, " "))
}

// finishXMLLog saves the session's raw XML to ~/.epp/last.xml and prints it
// when asked for (--xml), or offers it after a failure.
func finishXMLLog(offer bool) {
	if logBuf.Len() == 0 {
		return
	}
	path := saveLastXML()

	switch {
	case opts.verbose:
		// Already printed as it happened.
	case opts.xml:
		printXMLLog()
	case offer && interactive():
		if confirm("Show the raw XML of this session?", false) {
			printXMLLog()
		}
	case offer && path != "":
		fmt.Fprintf(os.Stderr, "Raw XML of this session: %s\n", path)
	}
}

func printXMLLog() {
	fmt.Fprintln(os.Stderr, "\n--- Raw XML ---")
	fmt.Fprint(os.Stderr, logBuf.String())
	fmt.Fprintln(os.Stderr, "---------------")
}

func saveLastXML() string {
	dir, err := eppDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(dir, "last.xml")
	if os.MkdirAll(dir, 0700) != nil || os.WriteFile(path, logBuf.Bytes(), 0600) != nil {
		return ""
	}
	return path
}
