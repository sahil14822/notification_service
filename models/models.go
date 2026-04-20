// Package models defines the core data structures and sentinel errors.
package models

import (
	"errors"
	"time"
)

// Sentinel errors used across the service.
var (
	ErrTemplateNotFound      = errors.New("template not found")
	ErrTemplateAlreadyExists = errors.New("template already exists")
	ErrNotificationNotFound  = errors.New("notification not found")
)

// Priority represents the urgency level of a notification.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

// NotificationType represents the category of the notification.
type NotificationType string

const (
	TypeAlert   NotificationType = "alert"
	TypeSuccess NotificationType = "success"
	TypeInfo    NotificationType = "info"
	TypeWarning NotificationType = "warning"
)

// Template is a reusable notification template with {{placeholder}} syntax.
type Template struct {
	ID        string    `json:"template_id"`
	Content   string    `json:"content"`
	IsDeleted bool      `json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
}

// Notification is the final rendered message sent to a user.
type Notification struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	TemplateID string            `json:"template_id"`
	Message    string            `json:"message"`
	Targets    map[string]string `json:"targets"`
	Priority   Priority          `json:"priority"`
	Type       NotificationType  `json:"type"`
	Read       bool              `json:"read"`
	CreatedAt  time.Time         `json:"created_at"`
}
