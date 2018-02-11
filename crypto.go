package main

import (
	"bytes"
	"io/ioutil"
	"log"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

//Symmetric-key encryption and decryption functions, based on https://gist.github.com/jyap808/8250124

func symmetricDecrypt(message string, key string) ([]byte, error) {
	decbuf := bytes.NewBuffer([]byte(message))
	result, err := armor.Decode(decbuf)
	if err != nil {
		log.Fatal(err)
	}
	md, err := openpgp.ReadMessage(result.Body, nil, func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		return []byte(key), nil
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	return bytes, nil
}

func symmetricEncrypt(message, key string) string {
	encryptionPassphrase := []byte(key)
	encryptionType := "PGP SIGNATURE"

	encbuf := bytes.NewBuffer(nil)
	w, err := armor.Encode(encbuf, encryptionType, nil)
	if err != nil {
		log.Fatal(err)
	}

	plaintext, err := openpgp.SymmetricallyEncrypt(w, encryptionPassphrase, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	messagebytes := []byte(message)
	_, err = plaintext.Write(messagebytes)

	plaintext.Close()
	w.Close()
	return encbuf.String()
}
