package epp

import (
	"encoding/xml"
	"testing"

	"github.com/nbio/st"
)

func TestEncodeDomainInfo(t *testing.T) {
	x, err := encodeDomainInfo(&Greeting{}, "example.com", nil)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><info><domain:info xmlns:domain="urn:ietf:params:xml:ns:domain-1.0"><domain:name hosts="none">example.com</domain:name></domain:info></info></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeDomainInfoWithNamestore(t *testing.T) {
	greeting := &Greeting{
		Extensions: []string{ExtNamestore},
	}
	extData := map[string]string{
		"namestoreExt:subProduct": "COM",
	}
	x, err := encodeDomainInfo(greeting, "example.com", extData)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><info><domain:info xmlns:domain="urn:ietf:params:xml:ns:domain-1.0"><domain:name hosts="none">example.com</domain:name></domain:info></info><extension><namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1"><namestoreExt:subProduct>COM</namestoreExt:subProduct></namestoreExt:namestoreExt></extension></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeContactInfo(t *testing.T) {
	x, err := encodeContactInfo(nil, "contact123", "auth123", nil)
	st.Expect(t, err, nil)
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><info><contact:info xmlns:contact="urn:ietf:params:xml:ns:contact-1.0"><contact:id>contact123</contact:id><contact:authInfo><contact:pw>auth123</contact:pw></contact:authInfo></contact:info></info></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}
