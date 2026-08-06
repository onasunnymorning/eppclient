# EPP for Go

[![build status](https://github.com/onasunnymorning/eppclient/actions/workflows/goreleaser.yaml/badge.svg)](https://github.com/onasunnymorning/eppclient/actions)
[![pkg.go.dev](https://img.shields.io/badge/docs-pkg.go.dev-blue.svg)](https://pkg.go.dev/github.com/onasunnymorning/eppclient)

EPP ([Extensible Provisioning Protocol](https://tools.ietf.org/html/rfc5730)) client for [Go](https://golang.org/).

## EPP CLI

The `epp-id` command-line tool allows you to interact with EPP servers directly.

### Installation (Homebrew)

You can install the `epp-id` CLI using Homebrew:

```bash
brew tap onasunnymorning/tap
brew install epp-id
```

### Usage

The basic syntax is `epp-id [options] <command> [arguments]`.

#### Domain Operations

```bash
# Check domain availability and pricing (with optional launch phase and fee)
epp-id check example.com
epp-id check -phase sunrise -fee 10.00 -currency USD example.com

# Get detailed domain info
epp-id info domain example.com

# Create a new domain (with optional launch phase and fee)
epp-id create domain example.com -period 1 -auth secret123 -registrant contact-id
epp-id create domain example.com -period 1 -auth secret123 -registrant contact-id -phase sunrise -fee 10.00 -currency USD

# Renew a domain (automatically fetches current expiry if -exp is omitted)
epp-id renew domain example.com -period 1

# Delete a domain
epp-id delete domain example.com

# Restore a domain (RGP)
epp-id restore domain example.com

# Check EPP poll messages
epp-id poll

# Acknowledge a poll message
epp-id poll -ack 12345

# Transfer operations (query, request, approve, reject, cancel)
epp-id transfer domain example.com -op request -auth secret123
```

#### Contact, Host & Other Operations

```bash
# Print version information
epp-id version

# Create a new contact
epp-id create contact -id CID-1 -email user@example.com -name "John Doe" -city "New York" -cc US -auth secret123

# Create a new host
epp-id create host -ips 192.0.2.1,192.0.2.2 -v6 2001:db8::1 ns1.example.com

# Update a domain (add/remove nameservers, statuses)
epp-id update domain -add-ns ns1.example.net,ns2.example.net -rem-ns ns1.example.com example.com

# Send raw XML from a file
epp-id raw request.xml
```

## Library Installation

`go get github.com/onasunnymorning/eppclient`

## Library Usage

```go
import (
	"crypto/tls"
	"github.com/onasunnymorning/eppclient"
)

// ...

tconn, err := tls.Dial("tcp", "epp.example.com:700", nil)
if err != nil {
	return err
}

conn, err := epp.NewConn(tconn)
if err != nil {
	return err
}

// ...
```

## Author

© 2021-2025 nb.io LLC & onasunnymorning
