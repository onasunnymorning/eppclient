package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/wsxiaoys/terminal/color"
	"golang.org/x/term"
)

// interactive reports whether we may ask questions: not with -y, and only
// when both input and the prompt output (stderr) are terminals.
func interactive() bool {
	return !opts.yes && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
}

func newForm(fields ...huh.Field) *huh.Form {
	return huh.NewForm(huh.NewGroup(fields...)).WithOutput(os.Stderr)
}

// runForm shows the fields; Ctrl-C/Esc cancels the whole command.
func runForm(fields ...huh.Field) {
	err := newForm(fields...).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		abort()
	}
	fatalif(err)
}

func choose(title string, options ...huh.Option[string]) string {
	var v string
	runForm(huh.NewSelect[string]().Title(title).Options(options...).Value(&v))
	return v
}

func ask(title string, validate func(string) error) string {
	var v string
	runForm(huh.NewInput().Title(title).Validate(validate).Value(&v))
	return v
}

// confirm asks a yes/no question. Cancelling counts as "no".
func confirm(title string, def bool) bool {
	v := def
	if err := newForm(huh.NewConfirm().Title(title).Value(&v)).Run(); err != nil {
		return false
	}
	return v
}

// confirmOrAbort asks before sending a billable or destructive command.
// It is skipped with -y and when not on a terminal.
func confirmOrAbort(title string, def bool) {
	if interactive() && !confirm(title, def) {
		abort()
	}
}

func notEmpty(s string) error {
	if s == "" {
		return errors.New("required")
	}
	return nil
}

// reviewRow is one line of a review summary.
type reviewRow struct {
	label string
	value string
	note  string // where the value came from, e.g. "generated"
	warn  bool   // highlight the note: something needs attention
}

// printReview prints the values a command is about to send.
func printReview(title string, rows []reviewRow) {
	width := 0
	for _, r := range rows {
		width = max(width, len(r.label))
	}
	color.Fprintf(os.Stderr, "\n@{!}%s\n", title)
	for _, r := range rows {
		value := r.value
		if value == "" {
			value = "–"
		}
		fmt.Fprintf(os.Stderr, "  %-*s  %s", width, r.label, value)
		switch {
		case r.note != "" && r.warn:
			color.Fprintf(os.Stderr, "  @{y}(%s)", r.note)
		case r.note != "":
			color.Fprintf(os.Stderr, "  @{.}(%s)", r.note)
		}
		fmt.Fprintln(os.Stderr)
	}
	fmt.Fprintln(os.Stderr)
}

const (
	reviewProceed = "proceed"
	reviewEdit    = "edit"
	reviewCancel  = "cancel"
)

// reviewChoice asks what to do after printReview. extra options are shown
// between "proceed" and "edit".
func reviewChoice(proceedLabel string, extra ...huh.Option[string]) string {
	options := []huh.Option[string]{huh.NewOption(proceedLabel, reviewProceed)}
	options = append(options, extra...)
	options = append(options,
		huh.NewOption("Edit values", reviewEdit),
		huh.NewOption("Cancel", reviewCancel),
	)
	choice := choose("Proceed?", options...)
	if choice == reviewCancel {
		abort()
	}
	return choice
}

// row is a review line with an informational note.
func row(label, value, note string) reviewRow {
	return reviewRow{label: label, value: value, note: note}
}
