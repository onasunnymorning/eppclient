package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseBatchCommandsPreservesRawXML(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<eppBatch>
  <epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><login/></command></epp>
  <epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <command><logout/></command>
  </epp>
</eppBatch>`)

	commands, err := parseBatchCommands(data)
	if err != nil {
		t.Fatalf("parseBatchCommands() error = %v", err)
	}

	want := []string{
		`<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><login/></command></epp>`,
		"<epp xmlns=\"urn:ietf:params:xml:ns:epp-1.0\">\n    <command><logout/></command>\n  </epp>",
	}
	if len(commands) != len(want) {
		t.Fatalf("parseBatchCommands() returned %d commands, want %d", len(commands), len(want))
	}
	for index := range want {
		if got := string(commands[index]); got != want[index] {
			t.Errorf("command %d = %q, want %q", index+1, got, want[index])
		}
	}
}

func TestParseSampleBatchFile(t *testing.T) {
	data, err := os.ReadFile("../../config/.epp/xml/example-batch.xml")
	if err != nil {
		t.Fatal(err)
	}

	commands, err := parseBatchCommands(data)
	if err != nil {
		t.Fatalf("parseBatchCommands() error = %v", err)
	}
	if got, want := len(commands), 6; got != want {
		t.Fatalf("sample batch contains %d commands, want %d", got, want)
	}
}

func TestParseBatchCommandsRejectsInvalidDocuments(t *testing.T) {
	tests := []struct {
		name string
		xml  string
		want string
	}{
		{name: "wrong root", xml: `<batch/>`, want: "root element"},
		{name: "no commands", xml: `<eppBatch/>`, want: "contains no"},
		{name: "wrong child", xml: `<eppBatch><command/></eppBatch>`, want: "child element"},
		{name: "wrong namespace", xml: `<eppBatch><epp/></eppBatch>`, want: "namespace"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseBatchCommands([]byte(test.xml))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("parseBatchCommands() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestParseBatchArgs(t *testing.T) {
	delay, filename, err := parseBatchArgs([]string{"5s", "batch.xml"})
	if err != nil {
		t.Fatalf("parseBatchArgs() error = %v", err)
	}
	if delay != 5*time.Second || filename != "batch.xml" {
		t.Fatalf("parseBatchArgs() = %s, %q; want 5s, %q", delay, filename, "batch.xml")
	}
}

func TestParseBatchArgsRejectsInvalidDelay(t *testing.T) {
	tests := [][]string{
		{"invalid", "batch.xml"},
		{"-1s", "batch.xml"},
	}

	for _, args := range tests {
		if _, _, err := parseBatchArgs(args); err == nil {
			t.Fatalf("parseBatchArgs(%q) returned nil error", args)
		}
	}
}

func TestExecuteBatchWaitsOnlyBetweenCommands(t *testing.T) {
	sender := &fakeBatchSender{
		responses: [][]byte{[]byte("<response>1</response>"), []byte("<response>2</response>"), []byte("<response>3</response>")},
		failAt:    -1,
	}
	commands := [][]byte{[]byte("<epp>1</epp>"), []byte("<epp>2</epp>"), []byte("<epp>3</epp>")}
	var delays []time.Duration
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeBatch(sender, commands, 5*time.Second, func(delay time.Duration) {
		delays = append(delays, delay)
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("executeBatch() error = %v", err)
	}

	if !reflect.DeepEqual(sender.requests, commands) {
		t.Fatalf("requests = %q, want %q", sender.requests, commands)
	}
	if want := []time.Duration{5 * time.Second, 5 * time.Second}; !reflect.DeepEqual(delays, want) {
		t.Fatalf("delays = %v, want %v", delays, want)
	}
	if strings.Count(stdout.String(), "--- Batch Response") != len(commands) {
		t.Fatalf("stdout = %q, want %d response headings", stdout.String(), len(commands))
	}
	if strings.Count(stderr.String(), "Waiting 5s") != len(commands)-1 {
		t.Fatalf("stderr = %q, want %d wait messages", stderr.String(), len(commands)-1)
	}
}

func TestExecuteBatchStopsOnError(t *testing.T) {
	sender := &fakeBatchSender{failAt: 1}
	commands := [][]byte{[]byte("<epp>1</epp>"), []byte("<epp>2</epp>"), []byte("<epp>3</epp>")}
	var delays []time.Duration

	err := executeBatch(sender, commands, time.Second, func(delay time.Duration) {
		delays = append(delays, delay)
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "command 2/3") {
		t.Fatalf("executeBatch() error = %v, want command 2/3 failure", err)
	}
	if len(sender.requests) != 2 {
		t.Fatalf("sent %d requests, want 2", len(sender.requests))
	}
	if len(delays) != 1 {
		t.Fatalf("slept %d times, want 1", len(delays))
	}
}

type fakeBatchSender struct {
	requests  [][]byte
	responses [][]byte
	failAt    int
}

func (sender *fakeBatchSender) Raw(command []byte) ([]byte, error) {
	sender.requests = append(sender.requests, append([]byte(nil), command...))
	index := len(sender.requests) - 1
	if index == sender.failAt {
		return nil, errors.New("send failed")
	}
	if index < len(sender.responses) {
		return sender.responses[index], nil
	}
	return []byte("<response/>"), nil
}
