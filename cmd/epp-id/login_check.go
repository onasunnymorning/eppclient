package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	epp "github.com/freddymanullang/epp-id"
	"github.com/wsxiaoys/terminal/color"
)

const loginCheckLogTimeFormat = "20060102-150405"

type loginAttemptFunc func(*Config) (epp.Result, string, error)

type loginCheckResult struct {
	Profile         string
	Address         string
	Username        string
	TLSEnabled      bool
	StartedAt       time.Time
	EndedAt         time.Time
	LoginSuccessful bool
	EPPResult       epp.Result
	FailureStage    string
	Err             error
}

func loginCheckLogFilename(timestamp time.Time) string {
	return "test-login-" + timestamp.Format(loginCheckLogTimeFormat) + ".log"
}

func runLoginCheck(profile, directory string, startedAt time.Time) (string, error) {
	cfg, err := loadConfig(profile, true)
	if err != nil {
		result := loginCheckResult{
			Profile:      profile,
			StartedAt:    startedAt,
			EndedAt:      time.Now(),
			FailureStage: "credentials",
			Err:          err,
		}
		path, logErr := saveLoginCheckLog(directory, result)
		return path, errors.Join(err, logErr)
	}

	result := performLoginCheck(profile, cfg, startedAt, attemptEPPLogin)
	path, logErr := saveLoginCheckLog(directory, result)
	return path, errors.Join(result.Err, logErr)
}

func performLoginCheck(profile string, cfg *Config, startedAt time.Time, attempt loginAttemptFunc) (result loginCheckResult) {
	result = loginCheckResult{
		Profile:    profile,
		Address:    cfg.Addr,
		Username:   cfg.User,
		TLSEnabled: cfg.TLS,
		StartedAt:  startedAt,
	}
	defer func() {
		result.EndedAt = time.Now()
	}()

	result.EPPResult, result.FailureStage, result.Err = attempt(cfg)
	result.LoginSuccessful = result.Err == nil
	return result
}

func attemptEPPLogin(cfg *Config) (epp.Result, string, error) {
	tlsConfig, err := eppClientTLSConfig(cfg)
	if err != nil {
		return epp.Result{}, "TLS configuration", err
	}

	color.Fprintf(os.Stderr, "Connecting to %s\n", cfg.Addr)
	conn, err := net.Dial("tcp", cfg.Addr)
	if err != nil {
		return epp.Result{}, "TCP connection", err
	}

	if tlsConfig != nil {
		color.Fprintf(os.Stderr, "Establishing TLS connection\n")
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			_ = conn.Close()
			return epp.Result{}, "TLS handshake", err
		}
		conn = tlsConn
	}

	logger := epp.DebugLogger
	epp.DebugLogger = nil
	defer func() {
		epp.DebugLogger = logger
	}()

	color.Fprintf(os.Stderr, "Performing EPP handshake\n")
	eppConn, err := epp.NewConn(conn)
	if err != nil {
		_ = conn.Close()
		return epp.Result{}, "EPP handshake", err
	}

	color.Fprintf(os.Stderr, "Testing EPP login as %s...\n", cfg.User)
	result, err := eppConn.Login(cfg.User, cfg.Password, "")
	if err != nil {
		_ = eppConn.Conn.Close()
		var eppResult *epp.Result
		if errors.As(err, &eppResult) {
			result = *eppResult
		}
		return result, "EPP login", err
	}

	_ = eppConn.Close()
	return result, "", nil
}

func saveLoginCheckLog(directory string, result loginCheckResult) (string, error) {
	filename := loginCheckLogFilename(result.StartedAt)
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
			return "", fmt.Errorf("create login test log: %w", err)
		}

		if _, err := file.Write(formatLoginCheckLog(result)); err != nil {
			_ = file.Close()
			return "", fmt.Errorf("write login test log: %w", err)
		}
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close login test log: %w", err)
		}
		return path, nil
	}
}

func formatLoginCheckLog(result loginCheckResult) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "EPP-ID Login Test")
	fmt.Fprintln(&output, "================================")
	fmt.Fprintf(&output, "Profile: %s\n", result.Profile)
	fmt.Fprintf(&output, "EPP Server: %s\n", valueOrUnavailable(result.Address))
	fmt.Fprintf(&output, "Username: %s\n", valueOrUnavailable(result.Username))
	fmt.Fprintf(&output, "Credentials TLS Enabled: %t\n", result.TLSEnabled)
	fmt.Fprintln(&output)
	if result.LoginSuccessful {
		fmt.Fprintln(&output, "EPP Login: SUCCESS")
	} else {
		fmt.Fprintln(&output, "EPP Login: FAILED")
	}
	if result.EPPResult.Code != 0 {
		fmt.Fprintf(&output, "EPP Result Code: %d\n", result.EPPResult.Code)
	}
	if result.EPPResult.Message != "" {
		fmt.Fprintf(&output, "EPP Result Message: %s\n", result.EPPResult.Message)
	}
	fmt.Fprintln(&output)
	if result.LoginSuccessful {
		fmt.Fprintln(&output, "Result: SUCCESS")
	} else {
		fmt.Fprintln(&output, "Result: FAILED")
		if result.FailureStage != "" {
			fmt.Fprintf(&output, "Failure Stage: %s\n", result.FailureStage)
		}
		if result.Err != nil {
			fmt.Fprintf(&output, "Error: %v\n", result.Err)
		}
	}
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "Start Time: %s\n", result.StartedAt.Format(processTimeFormat))
	fmt.Fprintf(&output, "End Time: %s\n", result.EndedAt.Format(processTimeFormat))
	fmt.Fprintf(&output, "Duration Process: %s\n", result.EndedAt.Sub(result.StartedAt).Round(time.Millisecond))
	return output.Bytes()
}
