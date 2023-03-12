package epp

import (
	"io/ioutil"
	"io"
	"time"
)

// This function lets you send a raw XML frame to the server
func (c *Conn) SendFrame(frame []byte) error {
	return c.writeRequest(frame)
}


// readRawResponse dequeues and returns a raw EPP response frame from Conn
func (c *Conn) ReadRawResponse() ([]byte, error) {
	c.mRead.Lock()
	defer c.mRead.Unlock()
	if c.Timeout > 0 {
		c.Conn.SetReadDeadline(time.Now().Add(c.Timeout))
	}
	n, err := readDataUnitHeader(c.Conn)
	if err != nil {
		return nil, err
	}
	r := &io.LimitedReader{R: c.Conn, N: int64(n)}
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return body, err
	}
	return body, err
}
