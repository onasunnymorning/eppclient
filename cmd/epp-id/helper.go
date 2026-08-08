package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	rawXMLLogTimeFormat = "20060102-150405"
	processTimeFormat   = "2006-01-02 15:04:05.000 MST"
)

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

func rawXMLLogContent(buf *bytes.Buffer, startedAt, endedAt time.Time) []byte {
	var content bytes.Buffer
	content.Write(buf.Bytes())
	switch {
	case bytes.HasSuffix(content.Bytes(), []byte("\n\n")):
	case bytes.HasSuffix(content.Bytes(), []byte("\n")):
		content.WriteByte('\n')
	default:
		content.WriteString("\n\n")
	}

	fmt.Fprintf(&content, "Start Time: %s\n", startedAt.Format(processTimeFormat))
	fmt.Fprintf(&content, "End Time: %s\n", endedAt.Format(processTimeFormat))
	fmt.Fprintf(&content, "Duration Process: %s\n", endedAt.Sub(startedAt).Round(time.Millisecond))
	return content.Bytes()
}

func saveRawXMLLog(directory string, startedAt, endedAt time.Time, buf *bytes.Buffer) (string, error) {
	if buf.Len() == 0 {
		return "", nil
	}

	filename := rawXMLLogFilename(startedAt)
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

		if _, err := file.Write(rawXMLLogContent(buf, startedAt, endedAt)); err != nil {
			_ = file.Close()
			return "", fmt.Errorf("write raw XML log: %w", err)
		}
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close raw XML log: %w", err)
		}
		return path, nil
	}
}

func printAndSaveRawXML(w io.Writer, directory string, startedAt, endedAt time.Time, buf *bytes.Buffer) {
	printRawXML(w, buf)

	path, err := saveRawXMLLog(directory, startedAt, endedAt, buf)
	if err != nil {
		fmt.Fprintf(w, "Warning: could not save raw XML log: %v\n", err)
		return
	}
	if path != "" {
		fmt.Fprintf(w, "Raw XML log saved to %s\n", path)
	}
}
