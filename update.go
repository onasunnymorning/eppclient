package epp

import (
	"bytes"
	"encoding/xml"
)

// UpdateDomain requests the update of a domain.
// https://tools.ietf.org/html/rfc5731#section-3.2.5
func (c *Conn) UpdateDomain(domain string, add, rem map[string]interface{}, chg map[string]string) error {
	x, err := encodeDomainUpdate(&c.Greeting, domain, add, rem, chg)
	if err != nil {
		return err
	}
	err = c.writeRequest(x)
	if err != nil {
		return err
	}
	_, err = c.readResponse()
	return err
}

func encodeDomainUpdate(greeting *Greeting, domain string, add, rem map[string]interface{}, chg map[string]string) ([]byte, error) {
	buf := bytes.NewBufferString(xmlCommandPrefix)
	buf.WriteString(`<update><domain:update xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">`)
	buf.WriteString(`<domain:name>`)
	xml.EscapeText(buf, []byte(domain))
	buf.WriteString(`</domain:name>`)

	if len(add) > 0 {
		buf.WriteString(`<domain:add>`)
		encodeDomainAddRem(buf, add)
		buf.WriteString(`</domain:add>`)
	}

	if len(rem) > 0 {
		buf.WriteString(`<domain:rem>`)
		encodeDomainAddRem(buf, rem)
		buf.WriteString(`</domain:rem>`)
	}

	if len(chg) > 0 {
		buf.WriteString(`<domain:chg>`)
		if registrant, ok := chg["registrant"]; ok {
			buf.WriteString(`<domain:registrant>`)
			xml.EscapeText(buf, []byte(registrant))
			buf.WriteString(`</domain:registrant>`)
		}
		if auth, ok := chg["auth"]; ok {
			buf.WriteString(`<domain:authInfo><domain:pw>`)
			xml.EscapeText(buf, []byte(auth))
			buf.WriteString(`</domain:pw></domain:authInfo>`)
		}
		buf.WriteString(`</domain:chg>`)
	}

	buf.WriteString(`</domain:update></update>`)
	buf.WriteString(xmlCommandSuffix)
	return buf.Bytes(), nil
}

func encodeDomainAddRem(buf *bytes.Buffer, data map[string]interface{}) {
	if ns, ok := data["ns"].([]string); ok && len(ns) > 0 {
		buf.WriteString(`<domain:ns>`)
		for _, host := range ns {
			buf.WriteString(`<domain:hostObj>`)
			xml.EscapeText(buf, []byte(host))
			buf.WriteString(`</domain:hostObj>`)
		}
		buf.WriteString(`</domain:ns>`)
	}
	if contacts, ok := data["contacts"].(map[string]string); ok && len(contacts) > 0 {
		for _, result := range []string{"admin", "tech", "billing"} {
			if id, ok := contacts[result]; ok {
				buf.WriteString(`<domain:contact type="`)
				buf.WriteString(result)
				buf.WriteString(`">`)
				xml.EscapeText(buf, []byte(id))
				buf.WriteString(`</domain:contact>`)
			}
		}
	}
	if status, ok := data["status"].(map[string]string); ok && len(status) > 0 {
		for s, txt := range status {
			buf.WriteString(`<domain:status s="`)
			buf.WriteString(s)
			buf.WriteString(`">`)
			xml.EscapeText(buf, []byte(txt))
			buf.WriteString(`</domain:status>`)
		}
	}
}

// UpdateContact requests the update of a contact.
// https://tools.ietf.org/html/rfc5733#section-3.2.5
func (c *Conn) UpdateContact(id string, add, rem, chg map[string]interface{}) error {
	x, err := encodeContactUpdate(&c.Greeting, id, add, rem, chg)
	if err != nil {
		return err
	}
	err = c.writeRequest(x)
	if err != nil {
		return err
	}
	_, err = c.readResponse()
	return err
}

func encodeContactUpdate(greeting *Greeting, id string, add, rem, chg map[string]interface{}) ([]byte, error) {
	buf := bytes.NewBufferString(xmlCommandPrefix)
	buf.WriteString(`<update><contact:update xmlns:contact="urn:ietf:params:xml:ns:contact-1.0">`)
	buf.WriteString(`<contact:id>`)
	xml.EscapeText(buf, []byte(id))
	buf.WriteString(`</contact:id>`)

	if len(add) > 0 {
		buf.WriteString(`<contact:add>`)
		encodeContactAddRem(buf, add)
		buf.WriteString(`</contact:add>`)
	}

	if len(rem) > 0 {
		buf.WriteString(`<contact:rem>`)
		encodeContactAddRem(buf, rem)
		buf.WriteString(`</contact:rem>`)
	}

	if len(chg) > 0 {
		buf.WriteString(`<contact:chg>`)
		if pi, ok := chg["postal"].(PostalInfo); ok {
			encodePostalInfo(buf, &pi)
		}
		if voice, ok := chg["voice"].(string); ok {
			buf.WriteString(`<contact:voice>`)
			xml.EscapeText(buf, []byte(voice))
			buf.WriteString(`</contact:voice>`)
		}
		if fax, ok := chg["fax"].(string); ok {
			buf.WriteString(`<contact:fax>`)
			xml.EscapeText(buf, []byte(fax))
			buf.WriteString(`</contact:fax>`)
		}
		if email, ok := chg["email"].(string); ok {
			buf.WriteString(`<contact:email>`)
			xml.EscapeText(buf, []byte(email))
			buf.WriteString(`</contact:email>`)
		}
		if auth, ok := chg["auth"].(string); ok {
			buf.WriteString(`<contact:authInfo><contact:pw>`)
			xml.EscapeText(buf, []byte(auth))
			buf.WriteString(`</contact:pw></contact:authInfo>`)
		}
		buf.WriteString(`</contact:chg>`)
	}

	buf.WriteString(`</contact:update></update>`)
	buf.WriteString(xmlCommandSuffix)
	return buf.Bytes(), nil
}

func encodeContactAddRem(buf *bytes.Buffer, data map[string]interface{}) {
	if status, ok := data["status"].(map[string]string); ok && len(status) > 0 {
		for s, txt := range status {
			buf.WriteString(`<contact:status s="`)
			buf.WriteString(s)
			buf.WriteString(`">`)
			xml.EscapeText(buf, []byte(txt))
			buf.WriteString(`</contact:status>`)
		}
	}
}

// UpdateHost requests the update of a host.
// https://tools.ietf.org/html/rfc5732#section-3.2.5
func (c *Conn) UpdateHost(host string, add, rem, chg map[string]interface{}) error {
	x, err := encodeHostUpdate(&c.Greeting, host, add, rem, chg)
	if err != nil {
		return err
	}
	err = c.writeRequest(x)
	if err != nil {
		return err
	}
	_, err = c.readResponse()
	return err
}

func encodeHostUpdate(greeting *Greeting, host string, add, rem, chg map[string]interface{}) ([]byte, error) {
	buf := bytes.NewBufferString(xmlCommandPrefix)
	buf.WriteString(`<update><host:update xmlns:host="urn:ietf:params:xml:ns:host-1.0">`)
	buf.WriteString(`<host:name>`)
	xml.EscapeText(buf, []byte(host))
	buf.WriteString(`</host:name>`)

	if len(add) > 0 {
		buf.WriteString(`<host:add>`)
		encodeHostAddRem(buf, add)
		buf.WriteString(`</host:add>`)
	}

	if len(rem) > 0 {
		buf.WriteString(`<host:rem>`)
		encodeHostAddRem(buf, rem)
		buf.WriteString(`</host:rem>`)
	}

	if len(chg) > 0 {
		buf.WriteString(`<host:chg>`)
		if newName, ok := chg["name"].(string); ok {
			buf.WriteString(`<host:name>`)
			xml.EscapeText(buf, []byte(newName))
			buf.WriteString(`</host:name>`)
		}
		buf.WriteString(`</host:chg>`)
	}

	buf.WriteString(`</host:update></update>`)
	buf.WriteString(xmlCommandSuffix)
	return buf.Bytes(), nil
}

func encodeHostAddRem(buf *bytes.Buffer, data map[string]interface{}) {
	if ips, ok := data["ips"].([]string); ok {
		for _, ip := range ips {
			buf.WriteString(`<host:addr ip="v4">`)
			xml.EscapeText(buf, []byte(ip))
			buf.WriteString(`</host:addr>`)
		}
	}
	if v6, ok := data["v6"].([]string); ok {
		for _, ip := range v6 {
			buf.WriteString(`<host:addr ip="v6">`)
			xml.EscapeText(buf, []byte(ip))
			buf.WriteString(`</host:addr>`)
		}
	}
	if status, ok := data["status"].(map[string]string); ok && len(status) > 0 {
		for s, txt := range status {
			buf.WriteString(`<host:status s="`)
			buf.WriteString(s)
			buf.WriteString(`">`)
			xml.EscapeText(buf, []byte(txt))
			buf.WriteString(`</host:status>`)
		}
	}
}
