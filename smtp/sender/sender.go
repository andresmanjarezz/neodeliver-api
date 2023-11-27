package main

import (
	"log"

	"github.com/inconshreveable/log15"
)

// check: https://github.com/toorop/tmail
// TODO check inbox placement
// BIMI ? https://www.emailonacid.com/blog/article/email-deliverability/bimi/

func main() {
	bs, err := LoadEmlFile("./mail.eml")
	if err != nil {
		panic(err)
	}

	// TODO test bounce email
	// TODO support ARC

	to := "test-9we8pz7tc@srv1.mail-tester.com"
	c, err := Dial(ClientConfig{
		Server: "92.eu.neodeliver.io",
	}, to)

	if err != nil {
		panic(err)
	}

	defer c.Close()
	log15.Info("Connected to client")

	// Set the sender and recipient first
	if err := c.Mail("sacha@kupalabs.com"); err != nil {
		log.Fatal(err)
	} else if err := c.Rcpt(to); err != nil { // TODO test bounce email when adding multiple receipts (call Rcpt multiple times)
		log.Fatal(err)
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		log.Fatal(err)
	}

	wc.Write(bs)

	if err = wc.Close(); err != nil {
		log.Fatal(err)
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
		log.Fatal(err)
	}
}
