package main

import (
	"bytes"
	"fmt"
	"io"
)

func printRawXML(w io.Writer, buf *bytes.Buffer) {
	if buf.Len() == 0 {
		return
	}
	fmt.Fprintln(w, "\n--- Raw XML Log ---")
	fmt.Fprint(w, buf.String())
	fmt.Fprintln(w, "-------------------")
}
