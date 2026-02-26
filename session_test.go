package epp

import (
	"encoding/xml"
	"net"
	"testing"

	"github.com/nbio/st"
)

func TestEncodeLogin(t *testing.T) {
	x, err := encodeLogin("user123", "pass123", "newpass", "1.0", "en", []string{ObjDomain, ObjContact}, []string{ExtFee10})
	st.Expect(t, err, nil)

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><login><clID>user123</clID><pw>pass123</pw><newPW>newpass</newPW><options><version>1.0</version><lang>en</lang></options><svcs><objURI>urn:ietf:params:xml:ns:domain-1.0</objURI><objURI>urn:ietf:params:xml:ns:contact-1.0</objURI><svcExtension><extURI>urn:ietf:params:xml:ns:epp:fee-1.0</extURI></svcExtension></svcs></login></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeLoginWithoutNewPassword(t *testing.T) {
	x, err := encodeLogin("user123", "pass123", "", "1.0", "en", []string{ObjDomain}, nil)
	st.Expect(t, err, nil)

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><login><clID>user123</clID><pw>pass123</pw><options><version>1.0</version><lang>en</lang></options><svcs><objURI>urn:ietf:params:xml:ns:domain-1.0</objURI></svcs></login></command></epp>`
	st.Expect(t, string(x), expected)

	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}
