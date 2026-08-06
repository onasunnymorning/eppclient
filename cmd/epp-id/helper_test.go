package main

import (
	"bytes"
	"testing"
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
