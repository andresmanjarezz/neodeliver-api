package main

import (
	"os"
	"time"

	"github.com/inconshreveable/log15"
	dkim "github.com/toorop/go-dkim"
)

func LoadEmlFile(filename string) ([]byte, error) {
	bs, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return signDKIM(bs)
}

func signDKIM(bs []byte) ([]byte, error) {
	log15.Info("Singing DKIM")

	// Read private key
	privateKey, err := os.ReadFile("./ssl/private.key")
	if err != nil {
		return nil, err
	}

	options := dkim.NewSigOptions()
	options.PrivateKey = privateKey
	options.Domain = "neodeliver.io"
	options.Selector = "n1"
	options.SignatureExpireIn = 3600
	options.BodyLength = 50
	options.Headers = []string{"Subject", "From", "Date", "To"}
	options.AddSignatureTimestamp = true
	options.Canonicalization = "relaxed/relaxed"

	bs = append([]byte("Date: "+time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")+"\n"), bs...)
	err = dkim.Sign(&bs, options)
	return bs, err
}
