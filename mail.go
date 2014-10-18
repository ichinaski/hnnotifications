package main

import (
	"bytes"
	"fmt"
	"net/mail"
	"net/smtp"
)

func auth() smtp.Auth {
	// Set up authentication information.
	return smtp.PlainAuth("", hnUsername, hnPassword, hnSmtpHost)
}

func loadEmail(templ string, data interface{}) string {
	var doc bytes.Buffer
	if err := templates.ExecuteTemplate(&doc, templ, data); err != nil {
		Logger.Println("templates.ExecuteTemplate: %v", err)
	}
	return doc.String()
}

// sendVerification delivers an email with the account verification link
func sendVerification(email, url string) error {
	title := "HN Notifications - Email verification needed"
	message := loadEmail("activate.html", map[string]string{"url": url})
	return sendMail(email, title, message)
}

func sendItem(id int, title, url, email string) error {
	data := map[string]string{
		"title":      title,
		"link":       url,
		"discussion": fmt.Sprintf(commentsUrl, id),
	}
	message := loadEmail("item.html", data)

	return sendMail(email, title, message)
}

func sendMail(email, title, message string) error {
	from := mail.Address{"HN Notifications", hnEmail}
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = email
	headers["Subject"] = title
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	body := ""
	for k, v := range headers {
		body += fmt.Sprintf("%s: %s\n", k, v)
	}
	body += "\n" + message

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	return smtp.SendMail(hnSmtpAddr, auth(), hnEmail, []string{email}, []byte(body))
}
