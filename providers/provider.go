package providers

import "notification-service/models"

// Provider defines the standard interface for all notification delivery methods.
type Provider interface {
	// ID returns the channel identifier, e.g., "email", "sms".
	ID() string

	// Send dispatches the rendered notification to the designated target address.
	Send(notification *models.Notification, target string) error
}
