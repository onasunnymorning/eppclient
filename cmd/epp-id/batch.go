package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const eppNamespace = "urn:ietf:params:xml:ns:epp-1.0"

type rawCommandSender interface {
	Raw([]byte) ([]byte, error)
}

func runBatch(c rawCommandSender, args []string) {
	delay, filename, err := parseBatchArgs(args)
	fatalif(err)

	data, err := os.ReadFile(filename)
	fatalif(err)

	commands, err := parseBatchCommands(data)
	fatalif(err)

	fmt.Fprintf(os.Stderr, "Loaded %d EPP commands from %s\n", len(commands), filename)
	fatalif(executeBatch(c, commands, delay, time.Sleep, os.Stdout, os.Stderr))
}

func parseBatchArgs(args []string) (time.Duration, string, error) {
	if len(args) != 2 {
		return 0, "", fmt.Errorf("usage: epp-id batch <delay> <batch-file.xml>")
	}

	delay, err := time.ParseDuration(args[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid batch delay %q: %w", args[0], err)
	}
	if delay < 0 {
		return 0, "", fmt.Errorf("batch delay must not be negative")
	}

	return delay, args[1], nil
}

func parseBatchCommands(data []byte) ([][]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	commands := make([][]byte, 0)

	depth := 0
	rootDepth := 0
	rootFound := false
	rootClosed := false
	commandStart := -1
	commandDepth := 0

	for {
		tokenStart := int(decoder.InputOffset())
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid batch XML: %w", err)
		}

		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if !rootFound {
				if depth != 1 || value.Name.Local != "eppBatch" {
					return nil, fmt.Errorf("batch root element must be <eppBatch>")
				}
				rootFound = true
				rootDepth = depth
				continue
			}

			if rootClosed {
				return nil, fmt.Errorf("unexpected element <%s> after </eppBatch>", value.Name.Local)
			}

			if commandStart < 0 && depth == rootDepth+1 {
				if value.Name.Local != "epp" {
					return nil, fmt.Errorf("batch child element must be <epp>, got <%s>", value.Name.Local)
				}
				if value.Name.Space != eppNamespace {
					return nil, fmt.Errorf("batch command %d must use EPP namespace %q", len(commands)+1, eppNamespace)
				}
				commandStart = tokenStart
				commandDepth = depth
			}

		case xml.EndElement:
			if commandStart >= 0 && depth == commandDepth {
				commandEnd := int(decoder.InputOffset())
				command := append([]byte(nil), data[commandStart:commandEnd]...)
				commands = append(commands, command)
				commandStart = -1
			}

			if rootFound && depth == rootDepth {
				rootClosed = true
			}
			depth--

		case xml.CharData:
			if strings.TrimSpace(string(value)) == "" {
				continue
			}
			if !rootFound || rootClosed || (depth == rootDepth && commandStart < 0) {
				return nil, fmt.Errorf("unexpected text outside an <epp> command")
			}
		}
	}

	if !rootFound {
		return nil, fmt.Errorf("batch root element must be <eppBatch>")
	}
	if !rootClosed {
		return nil, fmt.Errorf("batch XML is missing </eppBatch>")
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("batch file contains no <epp> commands")
	}

	return commands, nil
}

func executeBatch(
	sender rawCommandSender,
	commands [][]byte,
	delay time.Duration,
	sleep func(time.Duration),
	stdout io.Writer,
	stderr io.Writer,
) error {
	for index, command := range commands {
		fmt.Fprintf(stderr, "Executing batch command %d/%d\n", index+1, len(commands))

		response, err := sender.Raw(command)
		if err != nil {
			return fmt.Errorf("batch command %d/%d failed: %w", index+1, len(commands), err)
		}
		fmt.Fprintf(stdout, "\n--- Batch Response %d/%d ---\n%s\n", index+1, len(commands), response)

		if index < len(commands)-1 && delay > 0 {
			fmt.Fprintf(stderr, "Waiting %s before next command...\n", delay)
			sleep(delay)
		}
	}

	return nil
}
