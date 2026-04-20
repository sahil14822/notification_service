// Package repository provides Cassandra-backed storage for templates and notifications.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"

	"notification-service/models"
)

// Store wraps a Cassandra session for templates and notifications.
type Store struct {
	session  *gocql.Session
	keyspace string
}

// New connects to Cassandra, bootstraps the schema, and returns a ready Store.
//
// hosts    — comma-separated contact points, e.g. "127.0.0.1" or "node1,node2"
// keyspace — Cassandra keyspace to use, e.g. "notification_service"
func New(_ context.Context, hosts string, keyspace string) (*Store, error) {
	hostList := strings.Split(hosts, ",")
	for i := range hostList {
		hostList[i] = strings.TrimSpace(hostList[i])
	}

	// ── Step 1: connect without a keyspace to create it if missing ──────────
	bootstrap := gocql.NewCluster(hostList...)
	bootstrap.Consistency = gocql.One
	bootstrap.Timeout = 30 * time.Second
	bootstrap.ConnectTimeout = 30 * time.Second

	initSession, err := bootstrap.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("cassandra: initial connect failed: %w", err)
	}

	createKS := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH replication = {
			'class': 'SimpleStrategy',
			'replication_factor': 1
		}`, keyspace)

	if err := initSession.Query(createKS).Exec(); err != nil {
		initSession.Close()
		return nil, fmt.Errorf("cassandra: create keyspace %q: %w", keyspace, err)
	}
	initSession.Close()

	// ── Step 2: reconnect with the keyspace ─────────────────────────────────
	cluster := gocql.NewCluster(hostList...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.One
	cluster.Timeout = 30 * time.Second
	cluster.ConnectTimeout = 30 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("cassandra: connect to keyspace %q: %w", keyspace, err)
	}

	s := &Store{session: session, keyspace: keyspace}

	if err := s.bootstrapSchema(); err != nil {
		session.Close()
		return nil, fmt.Errorf("cassandra: schema bootstrap: %w", err)
	}

	return s, nil
}

// bootstrapSchema creates required tables and indexes when they don't exist.
//
// Tables:
//
//	templates     — keyed by id (TEXT PRIMARY KEY)
//	notifications — keyed by id (TEXT PRIMARY KEY) + secondary index on user_id
func (s *Store) bootstrapSchema() error {
	stmts := []string{
		// ── templates ────────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS templates (
			id         TEXT PRIMARY KEY,
			content    TEXT,
			is_deleted BOOLEAN,
			created_at TIMESTAMP
		)`,

		// Ensure is_deleted column exists (for migrations)
		`ALTER TABLE templates ADD is_deleted BOOLEAN`,

		// Index for filtering deleted templates
		`CREATE INDEX IF NOT EXISTS templates_is_deleted_idx
			ON templates (is_deleted)`,

		// ── notifications ────────────────────────────────────────────────────
		// PRIMARY KEY is notification-id so we can UPDATE/SELECT by id directly.
		// A secondary index on user_id supports efficient per-user queries.
		`CREATE TABLE IF NOT EXISTS notifications (
			id          TEXT PRIMARY KEY,
			user_id     TEXT,
			template_id TEXT,
			message     TEXT,
			targets     TEXT,
			priority    TEXT,
			type        TEXT,
			read        BOOLEAN,
			created_at  TIMESTAMP
		)`,

		// Secondary index — allows SELECT … WHERE user_id = ? efficiently.
		`CREATE INDEX IF NOT EXISTS notifications_user_id_idx
			ON notifications (user_id)`,
	}

	for _, stmt := range stmts {
		if err := s.session.Query(stmt).Exec(); err != nil {
			// Ignore error if column/index already exists (for ALTER/CREATE)
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "already exists") || strings.Contains(errStr, "conflict") {
				continue
			}

			// Trim the statement so the error message stays readable.
			preview := stmt
			if len(preview) > 60 {
				preview = preview[:60] + "..."
			}
			return fmt.Errorf("executing %q: %w", preview, err)
		}
	}
	return nil
}

// Close terminates the Cassandra session.
func (s *Store) Close(_ context.Context) error {
	s.session.Close()
	return nil
}

// ─── Template Methods ─────────────────────────────────────────────────────────

// SaveTemplate inserts a new template using a Lightweight Transaction (IF NOT EXISTS).
// Returns models.ErrTemplateAlreadyExists when the ID is already taken.
//
// CQL:
//
//	INSERT INTO templates (id, content, created_at) VALUES (?, ?, ?) IF NOT EXISTS
func (s *Store) SaveTemplate(t *models.Template) error {
	var existingID, existingContent *string
	var existingCreatedAt *time.Time
	var existingIsDeleted bool

	applied, err := s.session.Query(`
		INSERT INTO templates (id, content, is_deleted, created_at)
		VALUES (?, ?, ?, ?)
		IF NOT EXISTS`,
		t.ID, t.Content, false, t.CreatedAt,
	).ScanCAS(&existingID, &existingContent, &existingIsDeleted, &existingCreatedAt)

	if err != nil {
		return fmt.Errorf("SaveTemplate: %w", err)
	}
	if !applied {
		return models.ErrTemplateAlreadyExists
	}
	return nil
}

// GetTemplate returns a single template by ID if not deleted.
//
// CQL:
//
//	SELECT id, content, is_deleted, created_at FROM templates WHERE id = ?
func (s *Store) GetTemplate(id string) (*models.Template, error) {
	var t models.Template
	err := s.session.Query(`
		SELECT id, content, is_deleted, created_at
		FROM templates
		WHERE id = ?`,
		id,
	).Scan(&t.ID, &t.Content, &t.IsDeleted, &t.CreatedAt)

	if err == gocql.ErrNotFound {
		return nil, models.ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetTemplate: %w", err)
	}

	if t.IsDeleted {
		return nil, models.ErrTemplateNotFound
	}

	return &t, nil
}

// UpdateTemplate updates the content of an existing template.
// Returns models.ErrTemplateNotFound if the ID does not exist.
//
// CQL:
//
//	UPDATE templates SET content = ? WHERE id = ?
func (s *Store) UpdateTemplate(t *models.Template) error {
	// Existence check — Cassandra UPDATE is an upsert, so we check explicitly.
	if _, err := s.GetTemplate(t.ID); err != nil {
		return err // propagates ErrTemplateNotFound
	}

	if err := s.session.Query(`
		UPDATE templates SET content = ? WHERE id = ?`,
		t.Content, t.ID,
	).Exec(); err != nil {
		return fmt.Errorf("UpdateTemplate: %w", err)
	}
	return nil
}

// DeleteTemplate soft-deletes a template by ID.
// Returns models.ErrTemplateNotFound if the ID does not exist.
//
// CQL:
//
//	UPDATE templates SET is_deleted = true WHERE id = ?
func (s *Store) DeleteTemplate(id string) error {
	// Existence check
	if _, err := s.GetTemplate(id); err != nil {
		return err // propagates ErrTemplateNotFound
	}

	if err := s.session.Query(`
		UPDATE templates SET is_deleted = true WHERE id = ?`, id,
	).Exec(); err != nil {
		return fmt.Errorf("DeleteTemplate: %w", err)
	}
	return nil
}

// AllTemplates returns all non-deleted templates.
//
// CQL:
//
//	SELECT id, content, is_deleted, created_at FROM templates WHERE is_deleted = false
func (s *Store) AllTemplates() []*models.Template {
	iter := s.session.Query(`
		SELECT id, content, is_deleted, created_at FROM templates
		WHERE is_deleted = false ALLOW FILTERING`,
	).Iter()

	var results []*models.Template
	var t models.Template
	for iter.Scan(&t.ID, &t.Content, &t.IsDeleted, &t.CreatedAt) {
		cp := t // copy before taking address
		results = append(results, &cp)
	}

	if err := iter.Close(); err != nil {
		return []*models.Template{}
	}
	if results == nil {
		return []*models.Template{}
	}
	return results
}

// ─── Notification Methods ─────────────────────────────────────────────────────

// SaveNotification persists a new notification.
// The Targets map is JSON-encoded because Cassandra maps require fixed value types.
//
// CQL:
//
//	INSERT INTO notifications (id, user_id, template_id, message, targets,
//	                           priority, type, read, created_at)
//	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
func (s *Store) SaveNotification(n *models.Notification) {
	targetsJSON, _ := json.Marshal(n.Targets)

	_ = s.session.Query(`
		INSERT INTO notifications
			(id, user_id, template_id, message, targets, priority, type, read, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.UserID, n.TemplateID, n.Message,
		string(targetsJSON),
		string(n.Priority), string(n.Type), n.Read, n.CreatedAt,
	).Exec() //nolint:errcheck
}

// GetNotificationsByUser returns all notifications for the given user_id.
// Uses the secondary index on user_id.
//
// CQL:
//
//	SELECT id, user_id, template_id, message, targets, priority, type, read, created_at
//	FROM notifications WHERE user_id = ?
func (s *Store) GetNotificationsByUser(userID string) []*models.Notification {
	iter := s.session.Query(`
		SELECT id, user_id, template_id, message, targets, priority, type, read, created_at
		FROM notifications
		WHERE user_id = ?`,
		userID,
	).Iter()

	var results []*models.Notification
	for {
		var n models.Notification
		var targetsJSON string
		var priority, nType string

		if !iter.Scan(
			&n.ID, &n.UserID, &n.TemplateID, &n.Message, &targetsJSON,
			&priority, &nType, &n.Read, &n.CreatedAt,
		) {
			break
		}

		n.Priority = models.Priority(priority)
		n.Type = models.NotificationType(nType)
		_ = json.Unmarshal([]byte(targetsJSON), &n.Targets)

		cp := n
		results = append(results, &cp)
	}

	if err := iter.Close(); err != nil {
		return []*models.Notification{}
	}
	if results == nil {
		return []*models.Notification{}
	}
	return results
}

// MarkNotificationRead sets a notification's `read` field to true.
// Fetches the row first to confirm it exists; returns ErrNotificationNotFound if not.
//
// CQL (check):
//
//	SELECT id FROM notifications WHERE id = ?
//
// CQL (update):
//
//	UPDATE notifications SET read = true WHERE id = ?
func (s *Store) MarkNotificationRead(id string) error {
	// Verify the notification exists (Cassandra UPDATE is an upsert).
	var existingID string
	err := s.session.Query(`
		SELECT id FROM notifications WHERE id = ?`, id,
	).Scan(&existingID)

	if err == gocql.ErrNotFound {
		return models.ErrNotificationNotFound
	}
	if err != nil {
		return fmt.Errorf("MarkNotificationRead (check): %w", err)
	}

	if err := s.session.Query(`
		UPDATE notifications SET read = true WHERE id = ?`, id,
	).Exec(); err != nil {
		return fmt.Errorf("MarkNotificationRead (update): %w", err)
	}
	return nil
}
