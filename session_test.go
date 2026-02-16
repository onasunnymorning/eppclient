package epp

import (
	"encoding/xml"
	"net"
	"testing"

	"github.com/nbio/st"
)

func TestEncodeLogin(t *testing.T) {
	x, err := encodeLogin("jane", "battery", "", "1.0", "en", nil, nil)
	st.Expect(t, err, nil)
	st.Expect(t, string(x), `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><login><clID>jane</clID><pw>battery</pw><options><version>1.0</version><lang>en</lang></options><svcs></svcs></login></command></epp>`)
	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

func TestEncodeLoginChangePassword(t *testing.T) {
	x, err := encodeLogin("jane", "battery", "horse", "1.0", "en", nil, nil)
	st.Expect(t, err, nil)
	st.Expect(t, string(x), `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0"><command><login><clID>jane</clID><pw>battery</pw><newPW>horse</newPW><options><version>1.0</version><lang>en</lang></options><svcs></svcs></login></command></epp>`)
	var v struct{}
	err = xml.Unmarshal(x, &v)
	st.Expect(t, err, nil)
}

var (
	testObjects = []string{
		ObjContact,
		ObjDomain,
		ObjFinance,
		ObjHost,
	}
	testExtensions = []string{
		ExtCharge,
		ExtFee05,
		ExtFee06,
		ExtIDN,
		ExtLaunch,
		ExtRGP,
		ExtSecDNS,
	}
)

func TestLoginReturnsResult(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()
	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		// Send greeting
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		// Read login request
		_, err = readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		// Send login response
		err = writeDataUnit(conn, []byte(testXMLLoginResponse))
		st.Assert(t, err, nil)
		// Read logout request
		_, err = readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		conn.Close()
	})
	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)

	result, err := c.Login("jane", "battery", "")
	st.Expect(t, err, nil)
	st.Expect(t, result.Code, 1000)
	st.Expect(t, result.Message, "Command completed successfully")

	c.Close()
}

var testXMLLoginResponse = `<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
	<response>
		<result code="1000">
			<msg>Command completed successfully</msg>
		</result>
		<trID>
			<svTRID>12345</svTRID>
		</trID>
	</response>
</epp>`

func BenchmarkEncodeLogin(b *testing.B) {
	for i := 0; i < b.N; i++ {
		encodeLogin("jane", "battery", "horse", "1.0", "en", testObjects, testExtensions)
	}
}
