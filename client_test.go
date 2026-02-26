package epp

import (
	"net"
	"testing"
	"time"

	"github.com/nbio/st"
)

func TestClientDomainInfo(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:infData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">
        <domain:name>example.com</domain:name>
        <domain:roid>EXAMPLE1-REP</domain:roid>
        <domain:status s="ok"/>
        <domain:clID>ClientX</domain:clID>
        <domain:crID>ClientY</domain:crID>
        <domain:crDate>1999-04-03T22:00:00.0Z</domain:crDate>
        <domain:exDate>2005-04-03T22:00:00.0Z</domain:exDate>
      </domain:infData>
    </resData>
    <trID>
      <clTRID>ABC-12345</clTRID>
      <svTRID>54321-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		// Send initial greeting
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)

		// Read Hello from client (conn setup)
		size, err := readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		buf := make([]byte, size)
		_, err = conn.Read(buf)
		st.Assert(t, err, nil)

		// Send greeting again as response to Hello
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)

		// Read DomainInfo request
		size, err = readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		buf = make([]byte, size)
		_, err = conn.Read(buf)
		st.Assert(t, err, nil)

		// Send DomainInfo response
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)

	err = c.Hello()
	st.Assert(t, err, nil)

	res, err := c.DomainInfo("example.com", nil)
	st.Expect(t, err, nil)
	st.Expect(t, res.Domain, "example.com")
	st.Expect(t, res.ID, "EXAMPLE1-REP")
	st.Expect(t, res.ClID, "ClientX")
}

func TestClientDomainRenew(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:renData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">
        <domain:name>example.com</domain:name>
        <domain:exDate>2027-04-03T22:00:00.0Z</domain:exDate>
      </domain:renData>
    </resData>
    <trID>
      <clTRID>ABC-12345</clTRID>
      <svTRID>54321-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)

		size, err := readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		buf := make([]byte, size)
		_, err = conn.Read(buf)

		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)

		size, err = readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		buf = make([]byte, size)
		_, err = conn.Read(buf)

		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	err = c.Hello()
	st.Assert(t, err, nil)

	curExp, _ := time.Parse("2006-01-02", "2025-04-03")
	res, err := c.RenewDomain("example.com", curExp, 1, "y", nil)
	st.Expect(t, err, nil)
	st.Expect(t, res.Domain, "example.com")
	st.Expect(t, res.ExDate.Year(), 2027)
}

func TestClientCreateDomain(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:creData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">
        <domain:name>example.com</domain:name>
        <domain:crDate>1999-04-03T22:00:00.0Z</domain:crDate>
        <domain:exDate>2001-04-03T22:00:00.0Z</domain:exDate>
      </domain:creData>
    </resData>
    <trID>
      <clTRID>ABC-12345</clTRID>
      <svTRID>54321-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)

		size, err := readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		buf := make([]byte, size)
		_, err = conn.Read(buf)

		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)

		size, err = readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		buf = make([]byte, size)
		_, err = conn.Read(buf)

		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	err = c.Hello()
	st.Assert(t, err, nil)

	res, err := c.CreateDomain("example.com", 1, "y", "auth123", "registrantID", nil, nil, nil)
	st.Expect(t, err, nil)
	st.Expect(t, res.Domain, "example.com")
	st.Expect(t, res.CrDate.Year(), 1999)
}

func TestClientTransferDomain(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1001">
      <msg>Command completed successfully; action pending</msg>
    </result>
    <resData>
      <domain:trnData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">
        <domain:name>example.com</domain:name>
        <domain:trStatus>pending</domain:trStatus>
        <domain:reID>ClientX</domain:reID>
        <domain:reDate>2000-06-08T22:00:00.0Z</domain:reDate>
        <domain:acID>ClientY</domain:acID>
        <domain:acDate>2000-06-13T22:00:00.0Z</domain:acDate>
        <domain:exDate>2002-09-08T22:00:00.0Z</domain:exDate>
      </domain:trnData>
    </resData>
    <trID>
      <clTRID>ABC-12345</clTRID>
      <svTRID>54321-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)

		size, err := readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		buf := make([]byte, size)
		_, err = conn.Read(buf)

		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)

		size, err = readDataUnitHeader(conn)
		st.Assert(t, err, nil)
		buf = make([]byte, size)
		_, err = conn.Read(buf)

		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	err = c.Hello()
	st.Assert(t, err, nil)

	res, err := c.TransferDomain("request", "example.com", 1, "y", "auth123", nil)
	st.Expect(t, err, nil)
	st.Expect(t, res.Domain, "example.com")
	st.Expect(t, res.Status, "pending")
	st.Expect(t, res.REID, "ClientX")
}

func TestClientDeleteDomain(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <trID>
      <clTRID>ABC-12345</clTRID>
      <svTRID>54321-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	err = c.DeleteDomain("example.com", nil)
	st.Expect(t, err, nil)
}

func TestClientRestoreDomain(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <trID>
      <clTRID>ABC-12345</clTRID>
      <svTRID>54321-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	_, err = c.RestoreDomain("example.com", nil)
	st.Expect(t, err, nil)
}

func TestClientUpdateDomain(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <trID>
      <clTRID>ABC-12345</clTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	err = c.UpdateDomain("example.com", nil, nil, nil)
	st.Expect(t, err, nil)
}

func TestClientCheckDomain(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:chkData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">
        <domain:cd>
          <domain:name avail="1">example.com</domain:name>
        </domain:cd>
      </domain:chkData>
    </resData>
    <trID>
      <clTRID>ABC-12345</clTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	res, err := c.CheckDomain("example.com")
	st.Expect(t, err, nil)
	st.Expect(t, res.Checks[0].Domain, "example.com")
	st.Expect(t, res.Checks[0].Available, true)
}

func TestClientCreateContact(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:creData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0">
        <contact:id>sh8013</contact:id>
        <contact:crDate>1999-04-03T22:00:00.0Z</contact:crDate>
      </contact:creData>
    </resData>
    <trID>
      <clTRID>ABC-12345</clTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	pi := PostalInfo{Name: "John Doe", City: "Miami", CC: "US"}
	res, err := c.CreateContact("sh8013", "jdoe@example.com", pi, "", "", nil)
	st.Expect(t, err, nil)
	st.Expect(t, res.ID, "sh8013")
	st.Expect(t, res.CrDate.Year(), 1999)
}

func TestClientCreateHost(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <host:creData xmlns:host="urn:ietf:params:xml:ns:host-1.0">
        <host:name>ns1.example.com</host:name>
        <host:crDate>1999-04-03T22:00:00.0Z</host:crDate>
      </host:creData>
    </resData>
    <trID>
      <clTRID>ABC-12345</clTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	res, err := c.CreateHost("ns1.example.com", []string{"192.0.2.2"}, nil)
	st.Expect(t, err, nil)
	st.Expect(t, res.Host, "ns1.example.com")
	st.Expect(t, res.CrDate.Year(), 1999)
}

func TestClientContactInfo(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:infData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0">
        <contact:id>sh8013</contact:id>
        <contact:roid>SH8013-REP</contact:roid>
        <contact:status s="linked"/>
        <contact:status s="clientDeleteProhibited"/>
        <contact:email>jdoe@example.com</contact:email>
        <contact:crDate>1999-04-03T22:00:00.0Z</contact:crDate>
      </contact:infData>
    </resData>
    <trID>
      <clTRID>ABC-12345</clTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	res, err := c.ContactInfo("sh8013", "2fooBAR", nil)
	st.Expect(t, err, nil)
	st.Expect(t, res.ID, "sh8013")
	st.Expect(t, res.ROID, "SH8013-REP")
	st.Expect(t, res.Email, "jdoe@example.com")
}

func TestClientPollReq(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1301">
      <msg>Command completed successfully; ack to dequeue</msg>
    </result>
    <msgQ count="5" id="12345">
      <qDate>2000-06-08T22:00:00.0Z</qDate>
      <msg>Transfer requested.</msg>
    </msgQ>
    <trID>
      <clTRID>ABC-12345</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	res, err := c.PollReq()
	st.Expect(t, err, nil)
	st.Expect(t, res.Count, 5)
	st.Expect(t, res.ID, "12345")
	st.Expect(t, res.Message, "Transfer requested.")
}

func TestClientDeleteContact(t *testing.T) {
	ls, err := newLocalServer()
	st.Assert(t, err, nil)
	defer ls.teardown()

	resXML := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <trID>
      <clTRID>ABC-12345</clTRID>
    </trID>
  </response>
</epp>`

	ls.buildup(func(ls *localServer, ln net.Listener) {
		conn, err := ls.Accept()
		st.Assert(t, err, nil)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ := readDataUnitHeader(conn)
		buf := make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(testXMLGreeting))
		st.Assert(t, err, nil)
		size, _ = readDataUnitHeader(conn)
		buf = make([]byte, size)
		conn.Read(buf)
		err = writeDataUnit(conn, []byte(resXML))
		st.Assert(t, err, nil)
	})

	nc, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	st.Assert(t, err, nil)
	c, err := NewConn(nc)
	st.Assert(t, err, nil)
	c.Hello()

	err = c.DeleteContact("sh8013", nil)
	st.Expect(t, err, nil)
}
