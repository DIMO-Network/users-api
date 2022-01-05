package services

import (
	"bytes"
	"fmt"
	"html/template"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"

	"github.com/DIMO-INC/users-api/internal/config"
)

//go:embed confirmation_email.html
var rawConfirmationEmail string

type EmailConfirmationService struct {
	settings     *config.Settings
	htmlTemplate *template.Template
}

func NewEmailConfirmationSercice(settings *config.Settings) *EmailConfirmationService {
	return &EmailConfirmationService{
		settings:     settings,
		htmlTemplate: template.Must(template.New("confirmation_email").Parse(rawConfirmationEmail)),
	}
}

func (e *EmailConfirmationService) SendConfirmationEmail(toAddress, confirmationKey string) error {
	auth := smtp.PlainAuth("", e.settings.EmailUsername, e.settings.EmailPassword, e.settings.EmailHost)
	addr := fmt.Sprintf("%s:%s", e.settings.EmailHost, e.settings.EmailPort)

	var partsBuffer bytes.Buffer
	w := multipart.NewWriter(&partsBuffer)
	defer w.Close() //nolint

	plainPart, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return err
	}
	plainPartWriter := quotedprintable.NewWriter(plainPart)
	if _, err := plainPartWriter.Write([]byte("Hi,\r\n\r\nYour email verification code is: " + confirmationKey + "\r\n")); err != nil {
		return err
	}
	plainPartWriter.Close()

	htmlPart, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return err
	}
	htmlPartWriter := quotedprintable.NewWriter(htmlPart)
	if err := e.htmlTemplate.Execute(htmlPartWriter, struct{ Key string }{confirmationKey}); err != nil {
		return err
	}
	htmlPartWriter.Close()

	var buffer bytes.Buffer
	buffer.WriteString("From: DIMO <" + e.settings.EmailFrom + ">\r\n" +
		"To: " + toAddress + "\r\n" +
		"Subject: [DIMO] Verification Code\r\n" +
		"Content-Type: multipart/alternative; boundary=\"" + w.Boundary() + "\"\r\n" +
		"\r\n")
	if _, err := partsBuffer.WriteTo(&buffer); err != nil {
		return err
	}

	if err := smtp.SendMail(addr, auth, e.settings.EmailFrom, []string{toAddress}, buffer.Bytes()); err != nil {
		return err
	}

	return nil
}
