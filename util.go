package main

import (
	"crypto/rand"
	"encoding/base64"
)

// newToken creates 16 byte random ID
func newToken() string {
	size := 16 // key size

	b := make([]byte, size)
	_, err := rand.Read(b)

	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(b)
}
