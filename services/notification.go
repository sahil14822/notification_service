package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"notification-service/models"
	"notification-service/providers"
	tmpl "notification-service/template"

	"github.com/google/uuid"
)

// CreateNotification builds and renders a notification, returning validation errors.
func (s *Service) CreateNotification(userID string, targets map[string]string, templateID string, data map[string]string, priority models.Priority, nType models.NotificationType) (*models.Notification, error) {
	userID = strings.TrimSpace(userID)
	templateID = strings.TrimSpace(templateID)

	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if templateID == "" {
		return nil, errors.New("template_id is required")
	}

	// Validate Priority
	switch priority {
	case models.PriorityLow, models.PriorityNormal, models.PriorityHigh:
		// valid
	case "":
		priority = models.PriorityNormal
	default:
		return nil, errors.New("invalid priority, must be low, normal, or high")
	}

	// Validate NotificationType
	switch nType {
	case models.TypeAlert, models.TypeSuccess, models.TypeInfo, models.TypeWarning:
		// valid
	case "":
		nType = models.TypeInfo
	default:
		return nil, errors.New("invalid type, must be alert, success, info, or warning")
	}

	// Fetch template
	t, err := s.repo.GetTemplate(templateID)
	if err != nil {
		return nil, err // ErrTemplateNotFound or internal errors
	}

	if data == nil {
		data = map[string]string{}
	}

	message, err := tmpl.Render(t.Content, data)
	if err != nil {
		// Missing variables
		return nil, err
	}

	n := &models.Notification{
		ID:         uuid.NewString(),
		UserID:     userID,
		TemplateID: templateID,
		Message:    message,
		Targets:    targets,
		Priority:   priority,
		Type:       nType,
		Read:       false,
		CreatedAt:  time.Now(),
	}

	s.repo.SaveNotification(n)

	// Dispatch using providers
	if targets != nil {
		for channel, targetAddr := range targets {
			if provider, exists := s.providers[channel]; exists {
				// Send asynchronously
				go func(p providers.Provider, addr string) {
					err := p.Send(n, addr)
					if err != nil {
						fmt.Printf("Failed to send %s notification to %s: %v\n", channel, addr, err)
					} else {
						fmt.Printf("Successfully sent %s notification to %s\n", channel, addr)
					}
				}(provider, targetAddr)
			} else {
				fmt.Printf("Unsupported notification channel skipped: %s\n", channel)
			}
		}
	}

	return n, nil
}

// GetUserNotifications returns all notifications for the given user.
func (s *Service) GetUserNotifications(userID string) ([]*models.Notification, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	notifications := s.repo.GetNotificationsByUser(userID)
	if notifications == nil {
		notifications = []*models.Notification{}
	}
	return notifications, nil
}

// MarkNotificationRead sets a notification's status to read.
func (s *Service) MarkNotificationRead(id string) error {
	if id == "" {
		return errors.New("id is required")
	}
	return s.repo.MarkNotificationRead(id)
}
