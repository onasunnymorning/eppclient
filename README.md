# EPP for Go

[![build status](https://github.com/onasunnymorning/eppclient/actions/workflows/goreleaser.yaml/badge.svg)](https://github.com/onasunnymorning/eppclient/actions)
[![pkg.go.dev](https://img.shields.io/badge/docs-pkg.go.dev-blue.svg)](https://pkg.go.dev/github.com/onasunnymorning/eppclient)

EPP ([Extensible Provisioning Protocol](https://tools.ietf.org/html/rfc5730)) client for [Go](https://golang.org/).

## EPP CLI

The `epp` command-line tool allows you to interact with EPP servers directly.

### Installation (Homebrew)

You can install the `epp` CLI using Homebrew:

```bash
brew tap onasunnymorning/tap
brew install epp
```

### Usage

The basic syntax is `epp [options] <command> [arguments]`.

#### Domain Operations

```bash
# Check domain availability and pricing
epp check example.com

# Get detailed domain info
epp info domain example.com

# Create a new domain
epp create domain example.com -period 1 -auth secret123 -registrant contact-id

# Renew a domain (automatically fetches current expiry if -exp is omitted)
epp renew domain example.com -period 1

# Delete a domain
epp delete domain example.com

# Restore a domain (RGP)
epp restore domain example.com

# Check EPP poll messages
epp poll

# Acknowledge a poll message
epp poll -ack 12345

# Transfer operations (query, request, approve, reject, cancel)
epp transfer domain example.com -op request -auth secret123
```

#### Contact, Host & Other Operations

```bash
# Print version information
epp version

# Create a new contact
epp create contact -id CID-1 -email user@example.com -name "John Doe" -city "New York" -cc US -auth secret123

# Create a new host
epp create host -ips 192.0.2.1,192.0.2.2 -v6 2001:db8::1 ns1.example.com

# Update a domain (add/remove nameservers, statuses)
epp update domain -add-ns ns1.example.net,ns2.example.net -rem-ns ns1.example.com example.com

# Send raw XML from a file
epp raw request.xml
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
