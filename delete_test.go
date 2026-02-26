package epp

import (
	"encoding/xml"
	"testing"

	"github.com/nbio/st"
)

func TestEncodeDomainDelete(t *testing.T) {
	x, err := encodeDomainDelete(nil, "example.com", nil)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><delete><domain:delete xmlns:domain="urn:ietf:params:xml:ns:domain-1.0"><domain:name>example.com</domain:name></domain:delete></delete></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeContactDelete(t *testing.T) {
	x, err := encodeContactDelete(nil, "contact123", nil)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><delete><contact:delete xmlns:contact="urn:ietf:params:xml:ns:contact-1.0"><contact:id>contact123</contact:id></contact:delete></delete></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeHostDelete(t *testing.T) {
	x, err := encodeHostDelete(nil, "ns1.example.com")
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><delete><host:delete xmlns:host="urn:ietf:params:xml:ns:host-1.0"><host:name>ns1.example.com</host:name></host:delete></delete></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}
