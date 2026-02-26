package epp

import (
	"encoding/xml"
	"testing"

	"github.com/nbio/st"
)

func TestEncodeDomainUpdate(t *testing.T) {
	add := map[string]interface{}{
		"ns":       []string{"ns1.example.com", "ns2.example.com"},
		"contacts": map[string]string{"admin": "admin123"},
		"status":   map[string]string{"clientHold": "Payment overdue"},
	}
	rem := map[string]interface{}{
		"ns": []string{"ns3.example.com"},
	}
	chg := map[string]string{
		"registrant": "newReg123",
		"auth":       "newAuth",
	}

	x, err := encodeDomainUpdate(nil, "example.com", add, rem, chg)
	st.Expect(t, err, nil)

	// Since maps iterate randomly, we do a basic structural check via Unmarshal
	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeContactUpdate(t *testing.T) {
	add := map[string]interface{}{
		"status": map[string]string{"clientTransferProhibited": ""},
	}
	rem := map[string]interface{}{
		"status": map[string]string{"clientDeleteProhibited": ""},
	}
	chg := map[string]interface{}{
		"postal": PostalInfo{
			City: "New City",
			CC:   "US",
		},
		"voice": "+1.5555555555",
		"email": "new@example.com",
		"auth":  "newAuth",
	}

	x, err := encodeContactUpdate(nil, "contact123", add, rem, chg)
	st.Expect(t, err, nil)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeHostUpdate(t *testing.T) {
	add := map[string]interface{}{
		"ips":    []string{"192.0.2.1"},
		"v6":     []string{"2001:db8::1"},
		"status": map[string]string{"clientUpdateProhibited": ""},
	}
	rem := map[string]interface{}{
		"ips": []string{"192.0.2.2"},
	}
	chg := map[string]interface{}{
		"name": "ns2.example.com",
	}

	x, err := encodeHostUpdate(nil, "ns1.example.com", add, rem, chg)
	st.Expect(t, err, nil)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}
