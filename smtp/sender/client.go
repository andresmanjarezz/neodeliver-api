package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/inconshreveable/log15"
)

// https://en.wikipedia.org/wiki/Simple_Mail_Transfer_Protocol

type ClientConfig struct {
	Server             string
	OnlyTLS            bool
	InsecureSkipVerify bool
	// TODO support security settings to force TLS usage
}

func Dial(config ClientConfig, email string) (*smtp.Client, error) {
	hosts, err := mxLookup(email)
	if err != nil {
		panic(err)
	}

	if len(hosts) == 0 {
		return nil, errors.New("no MX records found")
	}

	var e error
	for i, host := range hosts {
		c, err := connectSMTPServer(config, host)
		if err == nil {
			return c, nil
		}

		if i == 0 {
			e = err
		}
	}

	return nil, e
}

func connectSMTPServer(config ClientConfig, host string) (*smtp.Client, error) {
	log15.Info("Connecting to client", "host", host)

	// Connect to the remote SMTP server on the standard port 25.
	client, err := smtp.Dial(host + ":25")
	if err != nil {
		return nil, fmt.Errorf("failed to dial SMTP server: %v", err)
	} else if err = client.Hello(config.Server); err != nil {
		panic(err)
	}

	// Check if the server supports the STARTTLS extension.
	if ok, _ := client.Extension("STARTTLS"); ok {
		// Upgrade to a TLS connection.
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
			ServerName:         host,
		}

		if err := client.StartTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to upgrade to TLS: %v", err)
		}

		// fmt.Println("Upgraded to TLS")
	} else if config.OnlyTLS {
		client.Close()
		return nil, errors.New("server does not support TLS")
	}

	return client, nil
}

// ---

// find out all mx records of given email address
func mxLookup(email string) ([]string, error) {
	host := ""
	if i := strings.LastIndex(email, "@"); i > 0 {
		host = email[i+1:]
	} else {
		panic("invalid email address")
	}

	mxRecords, err := net.LookupMX(host)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup MX records: %v", err)
	}

	res := make([]string, len(mxRecords))
	for i, mx := range mxRecords {
		res[i] = strings.TrimSuffix(mx.Host, ".")
	}

	return res, nil
}
