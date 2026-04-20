package providers

import (
	"fmt"
	"os"

	"notification-service/models"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// SMSProvider implements the Provider interface for Twilio SMS.
type SMSProvider struct{}

func NewSMSProvider() *SMSProvider {
	return &SMSProvider{}
}

func (s *SMSProvider) ID() string {
	return "sms"
}

func (s *SMSProvider) Send(notification *models.Notification, target string) error {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	fromNumber := os.Getenv("TWILIO_PHONE_NUMBER")

	if accountSid == "" || authToken == "" || fromNumber == "" {
		return fmt.Errorf("Twilio credentials are not fully configured")
	}

	// Twilio Go client automatically picks up TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN
	// from environment variables if we initialize the defaults.
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	params := &openapi.CreateMessageParams{}
	params.SetTo(target)
	params.SetFrom(fromNumber)
	params.SetBody(notification.Message) 

	_, err := client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send SMS via Twilio: %v", err)
	}

	return nil
}
