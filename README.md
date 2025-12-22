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

```bash
# Check domain availability
epp check example.com

# Get domain info
epp info example.com

# Send raw XML
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
