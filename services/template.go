package services

import (
	"errors"
	"strings"
	"time"

	"notification-service/models"
	tmpl "notification-service/template"
)

// CreateTemplate validates input and creates a new template.
func (s *Service) CreateTemplate(id, content string) (*TemplateResponse, error) {
	id = strings.TrimSpace(id)
	content = strings.TrimSpace(content)

	if id == "" {
		return nil, errors.New("template_id is required")
	}
	if content == "" {
		return nil, errors.New("content is required")
	}
	if err := tmpl.Validate(content); err != nil {
		return nil, err
	}

	t := &models.Template{
		ID:        id,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveTemplate(t); err != nil {
		return nil, err
	}

	return &TemplateResponse{
		Template:  t,
		Variables: tmpl.ExtractVariables(t.Content),
	}, nil
}

// GetTemplate fetches a template by ID.
func (s *Service) GetTemplate(id string) (*models.Template, error) {
	if id == "" {
		return nil, errors.New("template_id is required")
	}
	return s.repo.GetTemplate(id)
}

// ListTemplates retrieves all active templates.
func (s *Service) ListTemplates() []*models.Template {
	return s.repo.AllTemplates()
}

// UpdateTemplate changes an existing template's content.
func (s *Service) UpdateTemplate(id, content string) (*TemplateResponse, error) {
	if id == "" {
		return nil, errors.New("template_id is required")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("content is required")
	}

	if err := tmpl.Validate(content); err != nil {
		return nil, err
	}

	t := &models.Template{
		ID:      id,
		Content: content,
	}

	if err := s.repo.UpdateTemplate(t); err != nil {
		return nil, err
	}

	updatedTmpl, err := s.repo.GetTemplate(id)
	if err != nil {
		// Just returning updated content directly since DB fetch failed
		return &TemplateResponse{
			Template:  t,
			Variables: tmpl.ExtractVariables(t.Content),
		}, nil
	}

	return &TemplateResponse{
		Template:  updatedTmpl,
		Variables: tmpl.ExtractVariables(updatedTmpl.Content),
	}, nil
}

// DeleteTemplate marks a template as deleted.
func (s *Service) DeleteTemplate(id string) error {
	if id == "" {
		return errors.New("template_id is required")
	}
	return s.repo.DeleteTemplate(id)
}
