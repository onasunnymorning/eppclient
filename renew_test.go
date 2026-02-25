package epp

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/nbio/st"
)

func TestEncodeDomainRenew(t *testing.T) {
	curExp, _ := time.Parse("2006-01-02", "2025-04-03")
	x, err := encodeDomainRenew(nil, "example.com", curExp, 1, "y", nil)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><renew><domain:renew xmlns:domain="urn:ietf:params:xml:ns:domain-1.0"><domain:name>example.com</domain:name><domain:curExpDate>2025-04-03</domain:curExpDate><domain:period unit="y">1</domain:period></domain:renew></renew></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeDomainRenewWithFee(t *testing.T) {
	curExp, _ := time.Parse("2006-01-02", "2025-04-03")
	extData := map[string]string{
		"fee:fee":      "50.00",
		"fee:currency": "USD",
	}
	x, err := encodeDomainRenew(nil, "example.com", curExp, 1, "y", extData)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><renew><domain:renew xmlns:domain="urn:ietf:params:xml:ns:domain-1.0"><domain:name>example.com</domain:name><domain:curExpDate>2025-04-03</domain:curExpDate><domain:period unit="y">1</domain:period></domain:renew></renew><extension><fee:renew xmlns:fee="urn:ietf:params:xml:ns:epp:fee-1.0"><fee:currency>USD</fee:currency><fee:fee>50.00</fee:fee></fee:renew></extension></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}
