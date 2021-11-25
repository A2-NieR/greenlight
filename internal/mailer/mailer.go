package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

// var holds email templates, store contents of ./templates in embedded file system variable

//go:embed "templates"
var templateFS embed.FS

// Mailer contains mail.Dialer instance & sender info
type Mailer struct {
	dialer *mail.Dialer
	sender string
}

// New initializes a new mail.Dialer instance with given SMTP settings
func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// Send mail methos on Mailer type
func (m Mailer) Send(recipient, templateFile string, data interface{}) error {
	// Parse required template file from embedded file system
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// Execute named templates, passing in dynamic data and store results in bytes.Buffer variables
	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// Initialize new mail.Message instance & set email data
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// Open connection to SMTP server, send message & close connection. Attempt 3 sends with 500ms between each attempt until failure
	for i := 1; i <= 3; i++ {
		err = m.dialer.DialAndSend(msg)
		// On success return nil
		if nil == err {
			return nil
		}

		// On failure sleep for 500ms then retry
		time.Sleep(500 * time.Millisecond)
	}

	return err
}
