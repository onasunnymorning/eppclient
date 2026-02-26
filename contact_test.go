package epp

import (
	"encoding/xml"
	"testing"

	"github.com/nbio/st"
)

func TestEncodeContactCreate(t *testing.T) {
	pi := PostalInfo{
		Name:   "John Doe",
		Org:    "Example Inc",
		Street: "123 Main St",
		City:   "New York",
		SP:     "NY",
		PC:     "10001",
		CC:     "US",
	}
	x, err := encodeContactCreate(nil, "contact123", "john@example.com", pi, "+1.5555555555", "auth123", nil)
	st.Expect(t, err, nil)

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><create><contact:create xmlns:contact="urn:ietf:params:xml:ns:contact-1.0"><contact:id>contact123</contact:id><contact:postalInfo type="int"><contact:name>John Doe</contact:name><contact:org>Example Inc</contact:org><contact:addr><contact:street>123 Main St</contact:street><contact:city>New York</contact:city><contact:sp>NY</contact:sp><contact:pc>10001</contact:pc><contact:cc>US</contact:cc></contact:addr></contact:postalInfo><contact:voice>+1.5555555555</contact:voice><contact:email>john@example.com</contact:email><contact:authInfo><contact:pw>auth123</contact:pw></contact:authInfo></contact:create></create></command></epp>`

	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}
