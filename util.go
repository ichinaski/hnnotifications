package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/mail"
)

func validateAddress(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func newToken() string {
	size := 16 // key size

	b := make([]byte, size)
	_, err := rand.Read(b)

	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(b)
}
