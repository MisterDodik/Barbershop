package mailer

import (
	"bytes"
	"errors"
	"net/http"
	"text/template"

	gomail "gopkg.in/gomail.v2"
)

type MailTrapMailer struct {
	apiKey    string
	fromEmail string
	host      string
	port      int
	username  string
	password  string
}

func NewMailTrapMailer(apiKey, fromEmail, host, username, password string, port int) (*MailTrapMailer, error) {
	if apiKey == "" || fromEmail == "" || host == "" || username == "" || password == "" {
		return &MailTrapMailer{}, errors.New("some fields are missing")
	}

	return &MailTrapMailer{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		host:      host,
		port:      port,
		username:  username,
		password:  password,
	}, nil
}

func (m *MailTrapMailer) Send(templateFile, username, email string, data any, isSandbox bool) (int, error) {
	//Template parsing

	if !isSandbox {
		return http.StatusAccepted, errors.New("isSandbox is set to false")
	}

	tmpl, err := template.ParseFS(FS, "templates/"+templateFile)
	if err != nil {
		return -1, err
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return -1, err
	}
	body := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(body, "body", data)
	if err != nil {
		return -1, err
	}

	message := gomail.NewMessage()
	message.SetHeader("From", m.fromEmail)
	message.SetHeader("To", email)
	message.SetHeader("Subject", subject.String())

	message.AddAlternative("text/html", body.String())

	dialer := gomail.NewDialer(m.host, m.port, m.username, m.apiKey)

	for i := 0; i < 3; i++ {
		if err := dialer.DialAndSend(message); err == nil {
			return 200, nil
		}
	}
	return -1, err
}
