package epp

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/nbio/st"
)

func TestPollResponse(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1301">
      <msg>Command completed successfully; ack to dequeue</msg>
    </result>
    <msgQ count="5" id="12345">
      <qDate>2025-12-24T09:30:00Z</qDate>
      <msg>Domain example.com has been deleted.</msg>
    </msgQ>
    <trID>
      <clTRID>ABC-12345</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	res := &Response{}
	err := IgnoreEOF(scanResponse.Scan(xml.NewDecoder(strings.NewReader(data)), res))
	st.Expect(t, err, nil)
	st.Expect(t, res.Result.Code, 1301)
	st.Expect(t, res.PollResponse.Count, 5)
	st.Expect(t, res.PollResponse.ID, "12345")
	st.Expect(t, res.PollResponse.Message, "Domain example.com has been deleted.")

	expectedDate, _ := time.Parse(time.RFC3339, "2025-12-24T09:30:00Z")
	st.Expect(t, res.PollResponse.Date.Unix(), expectedDate.Unix())
}

func TestPollAckResponse(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <msgQ count="4" id="12345"/>
    <trID>
      <clTRID>ABC-12346</clTRID>
      <svTRID>54323-XYZ</svTRID>
    </trID>
  </response>
</epp>`

	res := &Response{}
	err := IgnoreEOF(scanResponse.Scan(xml.NewDecoder(strings.NewReader(data)), res))
	st.Expect(t, err, nil)
	st.Expect(t, res.Result.Code, 1000)
	st.Expect(t, res.PollResponse.Count, 4)
	st.Expect(t, res.PollResponse.ID, "12345")
}
