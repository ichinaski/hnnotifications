package main

import (
	"bytes"
	"fmt"
	"github.com/jordan-wright/email"
	"net/mail"
	"net/smtp"
)

const (
	commentsUrl = "https://news.ycombinator.com/item?id=%d"
)

// auth sets up SMTP account credentials.
func auth() smtp.Auth {
	return smtp.PlainAuth("", config.SMTP.User, config.SMTP.Password, config.SMTP.Host)
}

// loadEmail applies the given data to a particular email template, returning the output bytes.
func loadEmail(templ string, data interface{}) ([]byte, error) {
	var doc bytes.Buffer
	err := useTemplate(templ, data, &doc)
	return doc.Bytes(), err
}

// sendVerification delivers an email with the account verification link.
func sendVerification(to, link string) {
	subject := "HN Notifications - Email verification needed"
	message, err := loadEmail("activate_email", map[string]string{"link": link})
	if err != nil {
		Logger.Println("Error: sendVerification() - ", err)
		return
	}

	e := email.NewEmail()
	e.From = config.Email
	e.To = []string{to}
	e.Subject = subject
	e.HTML = message
	err = e.Send(config.SMTP.Addr, auth())
	if err != nil {
		Logger.Println("Error: sendVerification() - ", err)
	}
}

// sendUnsubscription delivers an email with the unsubscription link.
func sendUnsubscription(to, link string) error {
	subject := "HN Notifications - Unsubscribe"
	message, err := loadEmail("unsubscribe_email", map[string]string{"link": link})
	if err != nil {
		return err
	}

	e := email.NewEmail()
	e.From = config.Email
	e.To = []string{to}
	e.Subject = subject
	e.HTML = message
	return e.Send(config.SMTP.Addr, auth())
}

// sendItem delivers a notification email for the given item.
func sendItem(id int, title, url string, bcc []string) error {
	data := map[string]string{
		"title":      title,
		"link":       url,
		"discussion": fmt.Sprintf(commentsUrl, id),
		"settings":   config.Url + "/settings",
	}
	message, err := loadEmail("item_email", data)
	if err != nil {
		return err
	}

	e := email.NewEmail()
	e.From = config.Email
	e.Bcc = bcc
	e.Subject = title
	e.HTML = message
	return e.Send(config.SMTP.Addr, auth())
}

// validateAddress is a simple email validation function.
func validateAddress(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
