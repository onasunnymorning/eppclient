package epp

import (
	"encoding/xml"
	"testing"

	"github.com/nbio/st"
)

func TestEncodeDomainTransfer(t *testing.T) {
	x, err := encodeDomainTransfer(nil, "request", "example.com", 1, "y", "auth123", nil)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><transfer op="request"><domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0"><domain:name>example.com</domain:name><domain:period unit="y">1</domain:period><domain:authInfo><domain:pw>auth123</domain:pw></domain:authInfo></domain:transfer></transfer></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeDomainTransferWithFee(t *testing.T) {
	extData := map[string]string{
		"fee:fee":      "15.00",
		"fee:currency": "USD",
	}
	x, err := encodeDomainTransfer(nil, "request", "example.com", 1, "y", "auth123", extData)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><transfer op="request"><domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0"><domain:name>example.com</domain:name><domain:period unit="y">1</domain:period><domain:authInfo><domain:pw>auth123</domain:pw></domain:authInfo></domain:transfer></transfer><extension><fee:transfer xmlns:fee="urn:ietf:params:xml:ns:epp:fee-1.0"><fee:currency>USD</fee:currency><fee:fee>15.00</fee:fee></fee:transfer></extension></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}
