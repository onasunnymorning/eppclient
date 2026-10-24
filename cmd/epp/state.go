package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// State the CLI writes itself, kept apart from the hand-edited credentials
// file: ~/.epp/state.json, keyed by profile name.
type profileState struct {
	Contact string `json:"contact,omitempty"` // default contact created or chosen via a prompt
}

type stateFile map[string]profileState

func statePath() (string, error) {
	dir, err := eppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

func readState(path string) (stateFile, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return stateFile{}, nil
	}
	if err != nil {
		return nil, err
	}
	st := stateFile{}
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return st, nil
}

func writeState(path string, st stateFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

// savedContact returns the default contact stored for the profile, if any.
func savedContact(profile string) string {
	path, err := statePath()
	if err != nil {
		return ""
	}
	st, err := readState(path)
	if err != nil {
		warnf("Ignoring %s: %v", path, err)
		return ""
	}
	return st[profile].Contact
}

// saveContact stores id as the profile's default contact.
func saveContact(profile, id string) {
	path, err := statePath()
	if err == nil {
		var st stateFile
		if st, err = readState(path); err == nil {
			ps := st[profile]
			ps.Contact = id
			st[profile] = ps
			err = writeState(path, st)
		}
	}
	if err != nil {
		warnf("Could not save default contact: %v", err)
	}
}

// defaultContact picks the profile's default contact: the credentials file
// wins over the saved state. The second value says where it came from.
func defaultContact() (id, source string) {
	if cfg != nil && cfg.Contact != "" {
		return cfg.Contact, "profile default"
	}
	if id := savedContact(opts.profile); id != "" {
		return id, "saved default"
	}
	return "", ""
}
