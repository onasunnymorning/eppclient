package epp

// This function lets you send a raw XML frame to the server
func (c *Conn) SendFrame(frame []byte) error {
	return c.writeRequest(frame)
}

// This function lets you receive a raw XML frame from the server
func (c *Conn) ReceiveFrame() (string, error) {
	response, err := c.readResponse()
	if err != nil {
		return "", err
	}
	return response.Reason, nil
}
