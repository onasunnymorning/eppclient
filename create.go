package epp

import (
	"bytes"
	"encoding/xml"
	"time"

	"github.com/nbio/xx"
)

// CreateDomain requests the creation of a domain.
// https://tools.ietf.org/html/rfc5731#section-3.2.1
func (c *Conn) CreateDomain(domain string, period int, unit string, auth string, registrant string, contacts map[string]string, ns []string, extData map[string]string) (*DomainCreateResponse, error) {
	x, err := encodeDomainCreate(&c.Greeting, domain, period, unit, auth, registrant, contacts, ns, extData)
	if err != nil {
		return nil, err
	}
	err = c.writeRequest(x)
	if err != nil {
		return nil, err
	}
	res, err := c.readResponse()
	if err != nil {
		return nil, err
	}
	return &res.DomainCreateResponse, nil
}

func encodeDomainCreate(greeting *Greeting, domain string, period int, unit string, auth string, registrant string, contacts map[string]string, ns []string, extData map[string]string) ([]byte, error) {
	buf := bytes.NewBufferString(xmlCommandPrefix)
	buf.WriteString(`<create><domain:create xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">`)
	buf.WriteString(`<domain:name>`)
	xml.EscapeText(buf, []byte(domain))
	buf.WriteString(`</domain:name>`)

	if period > 0 {
		buf.WriteString(`<domain:period unit="`)
		buf.WriteString(unit)
		buf.WriteString(`">`)
		buf.WriteString(xmlInt(period))
		buf.WriteString(`</domain:period>`)
	}

	if len(ns) > 0 {
		buf.WriteString(`<domain:ns>`)
		for _, host := range ns {
			buf.WriteString(`<domain:hostObj>`)
			xml.EscapeText(buf, []byte(host))
			buf.WriteString(`</domain:hostObj>`)
		}
		buf.WriteString(`</domain:ns>`)
	}

	if registrant != "" {
		buf.WriteString(`<domain:registrant>`)
		xml.EscapeText(buf, []byte(registrant))
		buf.WriteString(`</domain:registrant>`)
	}

	for _, result := range []string{"admin", "tech", "billing"} {
		if id, ok := contacts[result]; ok {
			buf.WriteString(`<domain:contact type="`)
			buf.WriteString(result)
			buf.WriteString(`">`)
			xml.EscapeText(buf, []byte(id))
			buf.WriteString(`</domain:contact>`)
		}
	}

	if auth != "" {
		buf.WriteString(`<domain:authInfo><domain:pw>`)
		xml.EscapeText(buf, []byte(auth))
		buf.WriteString(`</domain:pw></domain:authInfo>`)
	}

	buf.WriteString(`</domain:create></create>`)

	// Extensions
	fee, hasFee := extData["fee:fee"]
	phase, hasLaunch := extData["launch:phase"]
	hasExtension := hasFee || hasLaunch

	if hasExtension {
		buf.WriteString(`<extension>`)

		if hasFee {
			buf.WriteString(`<fee:create xmlns:fee="`)
			buf.WriteString(ExtFee10)
			buf.WriteString(`">`)
			if currency, ok := extData["fee:currency"]; ok {
				buf.WriteString(`<fee:currency>`)
				buf.WriteString(currency)
				buf.WriteString(`</fee:currency>`)
			}
			buf.WriteString(`<fee:fee>`)
			buf.WriteString(fee)
			buf.WriteString(`</fee:fee>`)
			buf.WriteString(`</fee:create>`)
		}

		if hasLaunch {
			buf.WriteString(`<launch:create xmlns:launch="`)
			buf.WriteString(ExtLaunch)
			buf.WriteString(`">`)
			buf.WriteString(`<launch:phase>`)
			xml.EscapeText(buf, []byte(phase))
			buf.WriteString(`</launch:phase>`)
			buf.WriteString(`</launch:create>`)
		}

		buf.WriteString(`</extension>`)
	}

	buf.WriteString(xmlCommandSuffix)
	return buf.Bytes(), nil
}

// DomainCreateResponse represents an EPP response for a domain create request.
type DomainCreateResponse struct {
	Domain string    // <domain:name>
	CrDate time.Time // <domain:crDate>
	ExDate time.Time // <domain:exDate>
}

// CreateHost requests the creation of a host.
// https://tools.ietf.org/html/rfc5732#section-3.2.1
func (c *Conn) CreateHost(host string, ips []string, v6 []string) (*HostCreateResponse, error) {
	x, err := encodeHostCreate(&c.Greeting, host, ips, v6)
	if err != nil {
		return nil, err
	}
	err = c.writeRequest(x)
	if err != nil {
		return nil, err
	}
	res, err := c.readResponse()
	if err != nil {
		return nil, err
	}
	return &res.HostCreateResponse, nil
}

func encodeHostCreate(greeting *Greeting, host string, ips []string, v6 []string) ([]byte, error) {
	buf := bytes.NewBufferString(xmlCommandPrefix)
	buf.WriteString(`<create><host:create xmlns:host="urn:ietf:params:xml:ns:host-1.0">`)
	buf.WriteString(`<host:name>`)
	xml.EscapeText(buf, []byte(host))
	buf.WriteString(`</host:name>`)

	for _, ip := range ips {
		buf.WriteString(`<host:addr ip="v4">`)
		xml.EscapeText(buf, []byte(ip))
		buf.WriteString(`</host:addr>`)
	}
	for _, ip := range v6 {
		buf.WriteString(`<host:addr ip="v6">`)
		xml.EscapeText(buf, []byte(ip))
		buf.WriteString(`</host:addr>`)
	}

	buf.WriteString(`</host:create></create>`)
	buf.WriteString(xmlCommandSuffix)
	return buf.Bytes(), nil
}

// HostCreateResponse represents an EPP response for a host create request.
type HostCreateResponse struct {
	Host   string    // <host:name>
	CrDate time.Time // <host:crDate>
}

func init() {
	path := "epp > response > resData > " + ObjDomain + " creData"
	scanResponse.MustHandleCharData(path+">name", func(c *xx.Context) error {
		dcr := &c.Value.(*Response).DomainCreateResponse
		dcr.Domain = string(c.CharData)
		return nil
	})
	scanResponse.MustHandleCharData(path+">crDate", func(c *xx.Context) error {
		dcr := &c.Value.(*Response).DomainCreateResponse
		var err error
		dcr.CrDate, err = time.Parse(time.RFC3339, string(c.CharData))
		return err
	})
	scanResponse.MustHandleCharData(path+">exDate", func(c *xx.Context) error {
		dcr := &c.Value.(*Response).DomainCreateResponse
		var err error
		dcr.ExDate, err = time.Parse(time.RFC3339, string(c.CharData))
		return err
	})

	pathHost := "epp > response > resData > " + ObjHost + " creData"
	scanResponse.MustHandleCharData(pathHost+">name", func(c *xx.Context) error {
		hcr := &c.Value.(*Response).HostCreateResponse
		hcr.Host = string(c.CharData)
		return nil
	})
	scanResponse.MustHandleCharData(pathHost+">crDate", func(c *xx.Context) error {
		hcr := &c.Value.(*Response).HostCreateResponse
		var err error
		hcr.CrDate, err = time.Parse(time.RFC3339, string(c.CharData))
		return err
	})
}
