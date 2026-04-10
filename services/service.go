package services

import (
	"notification-service/models"
	"notification-service/providers"
)

// Repository defines the data operations required by the services.
type Repository interface {
	// Templates
	SaveTemplate(t *models.Template) error
	GetTemplate(id string) (*models.Template, error)
	UpdateTemplate(t *models.Template) error
	DeleteTemplate(id string) error
	AllTemplates() []*models.Template

	// Notifications
	SaveNotification(n *models.Notification)
	GetNotificationsByUser(userID string) []*models.Notification
	MarkNotificationRead(id string) error
}

// Service provides all business logic operations.
type Service struct {
	repo      Repository
	providers map[string]providers.Provider
}

// New creates a new business logic service.
func New(r Repository) *Service {
	s := &Service{
		repo:      r,
		providers: make(map[string]providers.Provider),
	}
	
	// Register currently supported providers
	s.providers["email"] = providers.NewEmailProvider()
	s.providers["sms"] = providers.NewSMSProvider()
	
	return s
}

// TemplateResponse represents a template including its extracted variables.
type TemplateResponse struct {
	*models.Template
	Variables []string `json:"variables"`
}
