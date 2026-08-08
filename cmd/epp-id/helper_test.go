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
	location := time.FixedZone("WIB", 7*60*60)
	startedAt := time.Date(2026, time.August, 6, 17, 4, 5, 0, location)
	endedAt := startedAt.Add(5*time.Second + 125*time.Millisecond)
	rawXML := bytes.NewBufferString("REQUEST\n<epp>command</epp>\n\nRESPONSE\n<epp>result</epp>\n")

	path, err := saveRawXMLLog(directory, startedAt, endedAt, rawXML)
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
	want := rawXML.String() + "\n" +
		"Start Time: 2026-08-06 17:04:05.000 WIB\n" +
		"End Time: 2026-08-06 17:04:10.125 WIB\n" +
		"Duration Process: 5.125s\n"
	if got := string(data); got != want {
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
	startedAt := time.Now()

	path, err := saveRawXMLLog(directory, startedAt, startedAt.Add(time.Second), &bytes.Buffer{})
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
	startedAt := time.Date(2026, time.August, 6, 17, 4, 5, 0, time.UTC)
	endedAt := startedAt.Add(time.Second)

	firstPath, err := saveRawXMLLog(directory, startedAt, endedAt, bytes.NewBufferString("first command"))
	if err != nil {
		t.Fatal(err)
	}
	secondPath, err := saveRawXMLLog(directory, startedAt, endedAt, bytes.NewBufferString("second command"))
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
	if !strings.HasPrefix(string(firstData), "first command\n\nStart Time:") {
		t.Fatalf("first log was overwritten: %q", firstData)
	}
}

func TestRawXMLLogTimingFooterIsAtEnd(t *testing.T) {
	location := time.FixedZone("WIB", 7*60*60)
	startedAt := time.Date(2026, time.August, 6, 17, 4, 5, 250*1e6, location)
	endedAt := startedAt.Add(2*time.Minute + 750*time.Millisecond)

	content := rawXMLLogContent(bytes.NewBufferString("<epp/>"), startedAt, endedAt)
	wantSuffix := "Start Time: 2026-08-06 17:04:05.250 WIB\n" +
		"End Time: 2026-08-06 17:06:06.000 WIB\n" +
		"Duration Process: 2m0.75s\n"
	if !strings.HasSuffix(string(content), wantSuffix) {
		t.Fatalf("rawXMLLogContent() = %q, want suffix %q", content, wantSuffix)
	}
}

func TestPrintAndSaveRawXMLReportsPath(t *testing.T) {
	directory := t.TempDir()
	startedAt := time.Date(2026, time.August, 6, 17, 4, 5, 0, time.UTC)
	endedAt := startedAt.Add(time.Second)
	var output bytes.Buffer

	printAndSaveRawXML(&output, directory, startedAt, endedAt, bytes.NewBufferString("<epp/>\n"))

	if !strings.Contains(output.String(), "Raw XML log saved to "+filepath.Join(directory, "output-20260806-170405.log")) {
		t.Fatalf("printAndSaveRawXML() output = %q", output.String())
	}
}
