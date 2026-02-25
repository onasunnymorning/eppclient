package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	epp "github.com/onasunnymorning/eppclient"
)

func main() {
	server := "epp.ote.centralnicregistry.com:700"
	nc, err := tls.Dial("tcp", server, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Fatalf("Dial error: %v", err)
	}
	conn, err := epp.NewConn(nc)
	if err != nil {
		log.Fatalf("Conn error: %v", err)
	}

	err = conn.Login("H1609329485", "Str0ngP@zzzZZZZZz", "")
	if err != nil {
		log.Fatalf("Login error: %v", err)
	}

	domain := "amateur.radio" + fmt.Sprintf("%d", time.Now().Unix())

	rawXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<epp xmlns="urn:ietf:params:xml:ns:epp-1.0">
	<command>
		<create>
			<domain:create xmlns:domain="urn:ietf:params:xml:ns:domain-1.0">
				<domain:name>` + domain + `</domain:name>
				<domain:period unit="y">1</domain:period>
				<domain:registrant>dummyContact</domain:registrant>
				<domain:authInfo><domain:pw>Str0ngP@zzzZZZZ</domain:pw></domain:authInfo>
			</domain:create>
		</create>
		<extension>
			<fee:command name="create" phase="sunrise" xmlns:fee="urn:ietf:params:xml:ns:epp:fee-1.0">
				<period xmlns="fee" unit="y">1</period>
				<fee xmlns="fee">200.00</fee>
			</fee:command>
		</extension>
		<clTRID>test-create-phase-3</clTRID>
	</command>
</epp>`)

	res, err := conn.Raw(rawXML)
	if err != nil {
		log.Fatalf("Raw error: %v", err)
	}

	fmt.Println(string(res))
	conn.Close()
}
