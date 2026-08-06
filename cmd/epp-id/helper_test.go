package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrintRawXML(t *testing.T) {
	var output bytes.Buffer
	rawXML := bytes.NewBufferString("<epp>response</epp>\n")

	printRawXML(&output, rawXML)

	want := "\n--- Raw XML Log ---\n<epp>response</epp>\n-------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("printRawXML() output = %q, want %q", got, want)
	}
}

func TestPrintRawXMLWithEmptyBuffer(t *testing.T) {
	var output bytes.Buffer

	printRawXML(&output, &bytes.Buffer{})

	if output.Len() != 0 {
		t.Fatalf("printRawXML() wrote %q for an empty buffer", output.String())
	}
}

func TestRawXMLLogFilename(t *testing.T) {
	timestamp := time.Date(2026, time.August, 6, 17, 4, 5, 0, time.FixedZone("WIB", 7*60*60))

	if got, want := rawXMLLogFilename(timestamp), "output-20260806-170405.log"; got != want {
		t.Fatalf("rawXMLLogFilename() = %q, want %q", got, want)
	}
}

func TestSaveRawXMLLog(t *testing.T) {
	directory := t.TempDir()
	timestamp := time.Date(2026, time.August, 6, 17, 4, 5, 0, time.UTC)
	rawXML := bytes.NewBufferString("REQUEST\n<epp>command</epp>\n\nRESPONSE\n<epp>result</epp>\n")

	path, err := saveRawXMLLog(directory, timestamp, rawXML)
	if err != nil {
		t.Fatalf("saveRawXMLLog() error = %v", err)
	}
	if got, want := filepath.Base(path), "output-20260806-170405.log"; got != want {
		t.Fatalf("saveRawXMLLog() path = %q, want filename %q", path, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), rawXML.String(); got != want {
		t.Fatalf("saved log = %q, want %q", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0600); got != want {
		t.Fatalf("saved log permissions = %o, want %o", got, want)
	}
}

func TestSaveRawXMLLogSkipsEmptyBuffer(t *testing.T) {
	directory := t.TempDir()

	path, err := saveRawXMLLog(directory, time.Now(), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("saveRawXMLLog() error = %v", err)
	}
	if path != "" {
		t.Fatalf("saveRawXMLLog() path = %q, want empty", path)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("saveRawXMLLog() created %d files for empty XML", len(entries))
	}
}

func TestSaveRawXMLLogDoesNotOverwriteSameSecond(t *testing.T) {
	directory := t.TempDir()
	timestamp := time.Date(2026, time.August, 6, 17, 4, 5, 0, time.UTC)

	firstPath, err := saveRawXMLLog(directory, timestamp, bytes.NewBufferString("first command"))
	if err != nil {
		t.Fatal(err)
	}
	secondPath, err := saveRawXMLLog(directory, timestamp, bytes.NewBufferString("second command"))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := filepath.Base(firstPath), "output-20260806-170405.log"; got != want {
		t.Fatalf("first log filename = %q, want %q", got, want)
	}
	if got, want := filepath.Base(secondPath), "output-20260806-170405-2.log"; got != want {
		t.Fatalf("second log filename = %q, want %q", got, want)
	}
	firstData, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstData) != "first command" {
		t.Fatalf("first log was overwritten: %q", firstData)
	}
}

func TestPrintAndSaveRawXMLReportsPath(t *testing.T) {
	directory := t.TempDir()
	timestamp := time.Date(2026, time.August, 6, 17, 4, 5, 0, time.UTC)
	var output bytes.Buffer

	printAndSaveRawXML(&output, directory, timestamp, bytes.NewBufferString("<epp/>\n"))

	if !strings.Contains(output.String(), "Raw XML log saved to "+filepath.Join(directory, "output-20260806-170405.log")) {
		t.Fatalf("printAndSaveRawXML() output = %q", output.String())
	}
}
