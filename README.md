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

The syntax is `epp <command> <object> <name> [flags]`. Flags can go anywhere
and every command has `--help`.

You only need to give what you care about. Anything left out gets a sensible
default (a generated authInfo, a 1-year period, the profile's default contact),
and the CLI asks only what it can't guess. Before anything billable or
destructive is sent, you get a summary to confirm or edit.

```bash
epp create domain geoff.test
```

```text
Create domain geoff.test  (profile: default)
  period       1y
  registrant   epp-k3f9a2  (saved default)
  admin        epp-k3f9a2  (saved default)
  tech         epp-k3f9a2  (saved default)
  nameservers  –
  authInfo     q7#Rk2pX!m9Lw4Za  (generated)
  fee          12.00 USD  (fee check)

> Yes, create it
  Edit values
  Cancel
```

The first time a profile needs a contact, you're asked whether to create a test
contact or use an existing ID; the answer is remembered for that profile.
After each command the CLI prints the equivalent command without prompts, so
you can repeat it or put it in a script.

Global flags:

| Flag | Meaning |
| --- | --- |
| `--profile <name>` | profile in `~/.epp/credentials` (default `default`) |
| `-y`, `--yes` | accept all defaults, no questions or confirmations |
| `--xml` | print the raw XML of the session when done |
| `-v`, `--verbose` | print raw XML as it is sent and received |

When input isn't a terminal (scripts, CI), the CLI never prompts. It uses the
defaults, or fails with a message naming the missing flag. The raw XML of the
last session is always saved to `~/.epp/last.xml`.

#### Domain operations

```bash
# Check availability and pricing
epp check example.com other.example
epp check example.com --phase sunrise --period 2

# Domain details
epp info domain example.com

# Register a domain: defaults for everything, or override what you need
epp create domain example.com
epp create domain example.com -y
epp create domain example.com --period 2 --ns ns1.example.net,ns2.example.net
epp create domain example.com --registrant c-1 --admin c-2 --tech c-2 --auth 'S3cret!pw'
epp create domain example.com --phase sunrise --fee 10.00 --currency USD

# Renew (fetches the current expiry if --exp is omitted)
epp renew domain example.com --period 1

# Delete, restore (RGP)
epp delete domain example.com
epp restore domain example.com

# Transfers: query (default), request, approve, reject, cancel
epp transfer domain example.com --op request --auth secret123

# Poll messages
epp poll
epp poll --ack 12345
```

#### Contacts, hosts and other operations

```bash
# Create a contact: every field is optional and filled with test values
epp create contact
epp create contact --default        # also make it the profile's default contact
epp create contact --id CID-1 --email user@example.com --name "John Doe" --city "New York" --cc US

# Create a host
epp create host ns1.example.com --ips 192.0.2.1,192.0.2.2 --v6 2001:db8::1

# Update a domain (nameservers, statuses, contacts)
epp update domain example.com --add-ns ns1.example.net,ns2.example.net --rem-ns ns1.example.com

# Send raw XML from a file (or - for stdin)
epp raw request.xml

# Version, shell completion
epp version
epp completion zsh > "${fpath[1]}/_epp"
```

Older single-dash flags (`-period 2`, `-auth x`) still work.

#### Profiles and defaults

`~/.epp/credentials` is created on first run. Each `[profile]` section holds the
connection settings, plus optional defaults used when the matching flag is
left out:

```ini
[ote]
addr = epp.ote.registry.example:700
user = my-registrar
password = ...
tls = true

contact = my-contact-id                 # contact for registrant/admin/tech
contact_roles = registrant,admin,tech   # roles filled on domain create
ns = ns1.example.net,ns2.example.net    # nameservers for domain create
email = me@example.net                  # email for generated test contacts
```

A default contact chosen or created through a prompt is stored in
`~/.epp/state.json`. A `contact` set in the credentials file takes precedence.
Registries that don't use contacts (per their greeting) are never asked for one.

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
