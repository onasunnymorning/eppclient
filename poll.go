package epp

import (
	"bytes"
	"fmt"
	"time"

	"github.com/nbio/xx"
)

// PollReq requests a message from the server's message queue.
func (c *Conn) PollReq() (*PollResponse, error) {
	buf := bytes.NewBufferString(xmlCommandPrefix)
	buf.WriteString(`<poll op="req"/>`)
	buf.WriteString(xmlCommandSuffix)

	err := c.writeRequest(buf.Bytes())
	if err != nil {
		return nil, err
	}

	res, err := c.readResponse()
	if err != nil {
		return nil, err
	}

	return &res.PollResponse, nil
}

// PollAck acknowledges a message from the server's message queue.
func (c *Conn) PollAck(msgID string) (*PollResponse, error) {
	buf := bytes.NewBufferString(xmlCommandPrefix)
	fmt.Fprintf(buf, `<poll op="ack" msgID="%s"/>`, msgID)
	buf.WriteString(xmlCommandSuffix)

	err := c.writeRequest(buf.Bytes())
	if err != nil {
		return nil, err
	}

	res, err := c.readResponse()
	if err != nil {
		return nil, err
	}

	return &res.PollResponse, nil
}

// PollResponse represents a response to an EPP poll command.
type PollResponse struct {
	Count   int
	ID      string
	Date    time.Time
	Message string
}

func init() {
	path := "epp > response > msgQ"
	scanResponse.MustHandleStartElement(path, func(c *xx.Context) error {
		pr := &c.Value.(*Response).PollResponse
		pr.Count = c.AttrInt("", "count")
		pr.ID = c.Attr("", "id")
		return nil
	})
	scanResponse.MustHandleCharData(path+"> qDate", func(c *xx.Context) error {
		pr := &c.Value.(*Response).PollResponse
		if date, err := time.Parse(time.RFC3339, string(c.CharData)); err == nil {
			pr.Date = date
		}
		return nil
	})
	scanResponse.MustHandleCharData(path+"> msg", func(c *xx.Context) error {
		c.Value.(*Response).PollResponse.Message = string(c.CharData)
		return nil
	})
}
