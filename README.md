# epp-id

[![build status](https://github.com/freddymanullang/epp-id/actions/workflows/goreleaser.yaml/badge.svg)](https://github.com/freddymanullang/epp-id/actions)
[![pkg.go.dev](https://img.shields.io/badge/docs-pkg.go.dev-blue.svg)](https://pkg.go.dev/github.com/freddymanullang/epp-id)

EPP ([Extensible Provisioning Protocol](https://tools.ietf.org/html/rfc5730)) client for [Go](https://golang.org/).

## epp-id CLI

The `epp-id` command-line tool allows you to interact with EPP servers directly.

### Installation (Linux Server)

Linux installation requires Git and Go 1.25 or newer. Install a supported Go release by following the [official Go installation guide](https://go.dev/doc/install), then build `epp-id` from source:

```bash
sudo apt-get update
sudo apt-get install -y git ca-certificates

git clone https://github.com/freddymanullang/epp-id.git
cd epp-id
go build -trimpath -o epp-id ./cmd/epp-id
sudo install -m 0755 epp-id /usr/local/bin/epp-id

epp-id version
```

The first execution automatically installs the embedded sample configuration under `~/.epp`. After `epp-id version` finishes, edit the generated credentials and replace the sample certificates before connecting:

```bash
nano ~/.epp/credentials
chmod 600 ~/.epp/credentials
epp-id check ssl
epp-id check example.id
```

### Sample Configuration

The repository includes the following safe sample layout under `config/.epp`:

```text
config/
└── .epp/
    ├── credentials
    ├── ssl/
    │   ├── certificate.pem
    │   └── certificate.key
    └── xml/
        ├── example.xml
        └── example-batch.xml
```

The `go build` command embeds this complete tree into the `epp-id` binary. On its first execution, the application creates any missing files under the current user's `~/.epp` directory and automatically updates certificate paths in `credentials` for that home directory. Existing files are never overwritten, so no manual copy is required.

After building, run the binary once and edit the installed files directly:

```bash
./epp-id version
nano "$HOME/.epp/credentials"
chmod 600 "$HOME/.epp/credentials" "$HOME/.epp/ssl/certificate.key"
```

Replace all `CHANGE_ME` values, the certificate chain, and the private key before connecting to a real server. The embedded certificate chain and private key are public test fixtures. Never use them for a real EPP account or server. `certificate.pem` contains the sample client certificate, intermediate CA certificate, and root CA certificate in that order.

### Update (Linux Server)

Update the local source checkout, rebuild the binary, and install the new build over the existing executable:

```bash
cd /path/to/epp-id
git checkout main
git pull --ff-only origin main
go test ./...
go build -trimpath -o epp-id ./cmd/epp-id
sudo install -m 0755 epp-id /usr/local/bin/epp-id
epp-id version
```

Updating the binary does not overwrite files already stored under `~/.epp`.

### Uninstall (Linux Server)

Remove the installed executable and verify that it is no longer available:

```bash
sudo rm -f /usr/local/bin/epp-id
command -v epp-id
```

The uninstall command intentionally preserves `~/.epp` because it may contain credentials, private keys, XML requests, and logs. To stop using that configuration without deleting it, move it to a backup location:

```bash
mv "$HOME/.epp" "$HOME/.epp.backup"
```

### Usage

The basic syntax is `epp-id [options] <command> [arguments]`.

#### SSL Connection Test

Test the TLS connection configured in the selected `~/.epp/credentials` profile without performing an EPP greeting or login:

```bash
epp-id check ssl
epp-id -profile sandbox check ssl
```

The test connects to `addr`, loads the client certificate from `cert`, loads its private key from `key`, and uses `ca` to verify the server certificate. When `ca` is not configured, the system certificate pool is used. The handshake is restricted to TLS 1.2, matching an OpenSSL test with `-tls1_2`.

Every attempt, including a failed connection or handshake, creates a diagnostic log in the current directory:

```text
test-ssl-YYYYMMDD-hhmmss.log
```

The log starts with a separate TCP connection test. `IP EPP Server` records the resolved IP address actually used by the successful TCP connection. `IP EPP Client` is the public outgoing IP reported by the RIPEstat `whats-my-ip` service at `https://stat.ripe.net/data/whats-my-ip/data.json`. If the public IP lookup fails, the error is recorded in the log and the TCP test still runs. The TLS test runs only when the TCP connection succeeds; a failed TCP result ends the log without an SSL/TLS section. Successful TCP connections continue with the negotiated cipher suite, server certificate details, TLS errors, and total processing duration. Existing logs are never overwritten; attempts started within the same second receive suffixes such as `-2`. Files are created with owner-only permissions (`0600`).

#### Domain Operations

```bash
# Check domain availability and pricing (with optional launch phase and fee)
epp-id check example.id
epp-id check -phase sunrise -fee 10.00 -currency USD example.id

# Get detailed domain info
epp-id info domain example.id

# Create a new domain (with optional launch phase and fee)
epp-id create domain example.id -period 1 -auth secret123 -registrant contact-id
epp-id create domain example.id -period 1 -auth secret123 -registrant contact-id -phase sunrise -fee 10.00 -currency USD

# Renew a domain (automatically fetches current expiry if -exp is omitted)
epp-id renew domain example.id -period 1

# Delete a domain
epp-id delete domain example.id

# Restore a domain (RGP)
epp-id restore domain example.id

# Check EPP poll messages
epp-id poll

# Acknowledge a poll message
epp-id poll -ack 12345

# Transfer operations (query, request, approve, reject, cancel)
epp-id transfer domain example.id -op request -auth secret123
```

#### Contact, Host & Other Operations

```bash
# Print version information
epp-id version

# Create a new contact
epp-id create contact -id CID-1 -email user@example.id -name "John Doe" -city "Jakarta" -cc ID -auth secret123

# Create a new host
epp-id create host -ips 192.0.2.1,192.0.2.2 -v6 2001:db8::1 ns1.example.id

# Update a domain (add/remove nameservers, statuses)
epp-id update domain -add-ns ns1.example.id,ns2.example.id -rem-ns ns3.example.id example.id

# Send raw XML from a file
epp-id raw request.xml
```

#### Batch Operations

Batch files use an `<eppBatch>` root containing one or more complete EPP commands. The CLI sends each direct `<epp>...</epp>` child in order and waits only between commands:

```bash
epp-id batch 5s gabungan-epp.xml
```

The delay accepts Go duration values such as `500ms`, `5s`, or `1m`. Batch mode does not automatically log in or log out, so the file is responsible for including `<login>` and `<logout>` commands when the server requires them. Processing stops on the first transport error.

Because raw XML output is enabled, credentials contained in batch XML may be printed to the terminal. Protect batch files and captured terminal output accordingly.

#### Raw XML Log Files

Every command that produces raw XML also saves it in the current directory. Log filenames use the command start time in local time with a 24-hour clock:

```text
output-YYYYMMDD-hhmmss.log
```

For example, a command started on 6 August 2026 at 17:04:05 creates `output-20260806-170405.log`. If multiple commands start within the same second, later files receive a numeric suffix such as `output-20260806-170405-2.log` so existing logs are never overwritten.

Each log ends with the command start time, end time, and total processing duration:

```text
Start Time: 2026-08-06 17:04:05.000 WIB
End Time: 2026-08-06 17:04:10.125 WIB
Duration Process: 5.125s
```

For batch operations, the duration covers the entire batch, including the delay between commands. Log files are created with owner-only permissions (`0600`) and are excluded by the repository's `.gitignore`. They can contain EPP credentials and personal data, so store and share them carefully.

## Library Installation

`go get github.com/freddymanullang/epp-id`

## Library Usage

```go
import (
	"crypto/tls"
	epp "github.com/freddymanullang/epp-id"
)

// ...

tconn, err := tls.Dial("tcp", "epp.example.id:700", nil)
if err != nil {
	return err
}

conn, err := epp.NewConn(tconn)
if err != nil {
	return err
}

// ...
```

## Maintainer

[@freddymanullang](https://github.com/freddymanullang)
