package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/mail"
	"strconv"
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

func parseEmail(r *http.Request) (string, bool) {
	email := r.FormValue("email")
	return email, validateAddress(email)
}

func parseScore(r *http.Request) (int, bool) {
	score, err := strconv.Atoi(r.FormValue("score"))
	return score, err == nil
}
