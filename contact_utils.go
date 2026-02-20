package epp

import (
	"bytes"
	"encoding/xml"
)

func encodePostalInfo(buf *bytes.Buffer, pi *PostalInfo) {
	// Postal Info "int" (international) is standard
	buf.WriteString(`<contact:postalInfo type="int">`)
	buf.WriteString(`<contact:name>`)
	xml.EscapeText(buf, []byte(pi.Name))
	buf.WriteString(`</contact:name>`)

	if pi.Org != "" {
		buf.WriteString(`<contact:org>`)
		xml.EscapeText(buf, []byte(pi.Org))
		buf.WriteString(`</contact:org>`)
	}

	buf.WriteString(`<contact:addr>`)
	if pi.Street != "" {
		buf.WriteString(`<contact:street>`)
		xml.EscapeText(buf, []byte(pi.Street))
		buf.WriteString(`</contact:street>`)
	}
	buf.WriteString(`<contact:city>`)
	xml.EscapeText(buf, []byte(pi.City))
	buf.WriteString(`</contact:city>`)

	if pi.SP != "" {
		buf.WriteString(`<contact:sp>`)
		xml.EscapeText(buf, []byte(pi.SP))
		buf.WriteString(`</contact:sp>`)
	}
	if pi.PC != "" {
		buf.WriteString(`<contact:pc>`)
		xml.EscapeText(buf, []byte(pi.PC))
		buf.WriteString(`</contact:pc>`)
	}

	buf.WriteString(`<contact:cc>`)
	xml.EscapeText(buf, []byte(pi.CC))
	buf.WriteString(`</contact:cc>`)
	buf.WriteString(`</contact:addr>`)
	buf.WriteString(`</contact:postalInfo>`)
}
