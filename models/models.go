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
	ID        string    `json:"template_id"  bson:"_id"`
	Content   string    `json:"content"      bson:"content"`
	IsDeleted bool      `json:"is_deleted"   bson:"is_deleted,omitempty"`
	CreatedAt time.Time `json:"created_at"   bson:"created_at"`
}

// Notification is the final rendered message sent to a user.
type Notification struct {
	ID         string    `json:"id"          bson:"_id"`
	UserID     string    `json:"user_id"     bson:"user_id"`
	TemplateID string    `json:"template_id" bson:"template_id"`
	Message    string           `json:"message"     bson:"message"`
	Targets    map[string]string `json:"targets"     bson:"targets"`
	Priority   Priority         `json:"priority"    bson:"priority"`
	Type       NotificationType `json:"type"        bson:"type"`
	Read       bool             `json:"read"        bson:"read"`
	CreatedAt  time.Time        `json:"created_at"  bson:"created_at"`
}
