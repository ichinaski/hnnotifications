package main

import (
	"bytes"
	"fmt"
	"github.com/jordan-wright/email"
	"net/smtp"
)

const (
	commentsUrl    = "https://news.ycombinator.com/item?id=%d"
	unsubscribeUrl = host + "/unsubscribe"
)

const (
	//hnUsername = "hnnotifications"
	//hnPassword = "f(dY4Bx_9U"
	//hnSmtpAddr = "smtp.gmail.com:587"
	//hnEmail = "hnnotifications@gmail.com"

	// TODO: Shitfuck use env vars instead!!
	hnUsername = "inigo@ichinaski.com"
	hnPassword = "8gvBue45"
	hnSmtpHost = "smtp.zoho.com"
	hnSmtpAddr = "smtp.zoho.com:587"
	hnEmail    = "HN Notifications <inigo@ichinaski.com>"
)

func auth() smtp.Auth {
	// Set up authentication information.
	return smtp.PlainAuth("", hnUsername, hnPassword, hnSmtpHost)
}

func loadEmail(templ string, data interface{}) ([]byte, error) {
	var doc bytes.Buffer
	err := templates.ExecuteTemplate(&doc, templ, data)
	return doc.Bytes(), err
}

// sendVerification delivers an email with the account verification link
func sendVerification(to, link string) {
	subject := "HN Notifications - Email verification needed"
	message, err := loadEmail("activate_email.html", map[string]string{"link": link})
	if err != nil {
		Logger.Println("Error: sendVerification() - ", err)
		return
	}

	e := email.NewEmail()
	e.From = hnEmail
	e.To = []string{to}
	e.Subject = subject
	e.HTML = message
	err = e.Send(hnSmtpAddr, auth())
	if err != nil {
		Logger.Println("Error: sendVerification() - ", err)
	}
}

// sendUnsubscription delivers an email with the unsubscription link
func sendUnsubscription(to, link string) error {
	subject := "HN Notifications - Unsubscribe"
	message, err := loadEmail("unsubscribe_email.html", map[string]string{"link": link})
	if err != nil {
		return err
	}

	e := email.NewEmail()
	e.From = hnEmail
	e.To = []string{to}
	e.Subject = subject
	e.HTML = message
	return e.Send(hnSmtpAddr, auth())
}

func sendItem(id int, title, url string, bcc []string) error {
	data := map[string]string{
		"title":       title,
		"link":        url,
		"discussion":  fmt.Sprintf(commentsUrl, id),
		"unsubscribe": unsubscribeUrl,
	}
	message, err := loadEmail("item_email.html", data)
	if err != nil {
		return err
	}

	e := email.NewEmail()
	e.From = hnEmail
	e.Bcc = bcc
	e.Subject = title
	e.HTML = message
	return e.Send(hnSmtpAddr, auth())
}
