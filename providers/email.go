package providers

import (
	"fmt"
	"net/smtp"
	"os"

	"notification-service/models"
)

// EmailProvider implements the Provider interface for SMTP emails.
type EmailProvider struct{}

func NewEmailProvider() *EmailProvider {
	return &EmailProvider{}
}

func (e *EmailProvider) ID() string {
	return "email"
}

func (e *EmailProvider) Send(notification *models.Notification, target string) error {
	from := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST") // e.g. smtp.gmail.com
	port := os.Getenv("SMTP_PORT") // e.g. 587

	if host == "" || port == "" || from == "" || password == "" {
		return fmt.Errorf("SMTP credentials are not fully configured")
	}

	auth := smtp.PlainAuth("", from, password, host)
	addr := fmt.Sprintf("%s:%s", host, port)

	// Determine subject based on type
	subject := string(notification.Type) + " Notification"

	// RFC 822 format
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", target, subject, notification.Message))

	return smtp.SendMail(addr, auth, from, []string{target}, msg)
	
}