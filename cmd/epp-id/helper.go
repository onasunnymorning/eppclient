package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const rawXMLLogTimeFormat = "20060102-150405"

func printRawXML(w io.Writer, buf *bytes.Buffer) {
	if buf.Len() == 0 {
		return
	}
	fmt.Fprintln(w, "\n--- Raw XML Log ---")
	fmt.Fprint(w, buf.String())
	fmt.Fprintln(w, "-------------------")
}

func rawXMLLogFilename(timestamp time.Time) string {
	return "output-" + timestamp.Format(rawXMLLogTimeFormat) + ".log"
}

func saveRawXMLLog(directory string, timestamp time.Time, buf *bytes.Buffer) (string, error) {
	if buf.Len() == 0 {
		return "", nil
	}

	filename := rawXMLLogFilename(timestamp)
	path := filepath.Join(directory, filename)
	for suffix := 2; ; suffix++ {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			extension := filepath.Ext(filename)
			base := filename[:len(filename)-len(extension)]
			path = filepath.Join(directory, fmt.Sprintf("%s-%d%s", base, suffix, extension))
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create raw XML log: %w", err)
		}

		if _, err := file.Write(buf.Bytes()); err != nil {
			_ = file.Close()
			return "", fmt.Errorf("write raw XML log: %w", err)
		}
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close raw XML log: %w", err)
		}
		return path, nil
	}
}

func printAndSaveRawXML(w io.Writer, directory string, timestamp time.Time, buf *bytes.Buffer) {
	printRawXML(w, buf)

	path, err := saveRawXMLLog(directory, timestamp, buf)
	if err != nil {
		fmt.Fprintf(w, "Warning: could not save raw XML log: %v\n", err)
		return
	}
	if path != "" {
		fmt.Fprintf(w, "Raw XML log saved to %s\n", path)
	}
}
