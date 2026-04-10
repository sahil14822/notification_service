// Package store provides MongoDB-backed storage for templates and notifications.
package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"notification-service/models"
)

// Store wraps MongoDB collections for templates and notifications.
type Store struct {
	client        *mongo.Client
	templates     *mongo.Collection
	notifications *mongo.Collection
}

// New connects to MongoDB, ensures indexes, and returns a ready Store.
// mongoURI = "mongodb+srv://sahilkanani8320_db_user:Sahil@testcluster.kzigwb7.mongodb.net/?appName=testCluster"

func New(ctx context.Context, mongoURI string) (*Store, error) {
	opts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}

	// Verify connectivity.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, err
	}

	db := client.Database("notification_service")
	s := &Store{
		client:        client,
		templates:     db.Collection("templates"),
		notifications: db.Collection("notifications"),
	}

	// Create an index on notifications.user_id for fast per-user lookups.
	if err := s.ensureIndexes(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

// ensureIndexes creates required MongoDB indexes if they don't already exist.
func (s *Store) ensureIndexes(ctx context.Context) error {
	idxModel := mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	}
	_, err := s.notifications.Indexes().CreateOne(ctx, idxModel)
	return err
}

// Close disconnects the MongoDB client.
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// ─── Template Methods ────────────────────────────────────────────────────────

// SaveTemplate inserts a new template. Returns ErrTemplateAlreadyExists on duplicate ID.
func (s *Store) SaveTemplate(t *models.Template) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.templates.InsertOne(ctx, t)
	if err != nil {
		var writeErr mongo.WriteException
		if errors.As(err, &writeErr) {
			for _, we := range writeErr.WriteErrors {
				if we.Code == 11000 { // duplicate key
					return models.ErrTemplateAlreadyExists
				}
			}
		}
		return err
	}
	return nil
}

// GetTemplate returns a template by ID.
func (s *Store) GetTemplate(id string) (*models.Template, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var t models.Template
	err := s.templates.FindOne(ctx, bson.M{"_id": id, "is_deleted": bson.M{"$ne": true}}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTemplate updates the content of an existing template.
func (s *Store) UpdateTemplate(t *models.Template) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := s.templates.UpdateOne(
		ctx,
		bson.M{"_id": t.ID, "is_deleted": bson.M{"$ne": true}},
		bson.M{"$set": bson.M{"content": t.Content}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return models.ErrTemplateNotFound
	}
	return nil
}

func (s *Store) DeleteTemplate(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := s.templates.UpdateOne(
		ctx, 
		bson.M{"_id": id, "is_deleted": bson.M{"$ne": true}},
		bson.M{"$set": bson.M{"is_deleted": true}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return models.ErrTemplateNotFound
	}
	return nil
}

// AllTemplates returns a slice of all templates.
func (s *Store) AllTemplates() []*models.Template {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.templates.Find(ctx, bson.M{"is_deleted": bson.M{"$ne": true}})
	if err != nil {
		return []*models.Template{}
	}
	defer cursor.Close(ctx)

	var result []*models.Template
	if err := cursor.All(ctx, &result); err != nil {
		return []*models.Template{}
	}
	return result
}

// ─── Notification Methods ────────────────────────────────────────────────────

// SaveNotification persists a new notification.
func (s *Store) SaveNotification(n *models.Notification) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.notifications.InsertOne(ctx, n) //nolint:errcheck
}

// GetNotificationsByUser returns all notifications for a given user_id.
func (s *Store) GetNotificationsByUser(userID string) []*models.Notification {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.notifications.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return []*models.Notification{}
	}
	defer cursor.Close(ctx)

	var result []*models.Notification
	if err := cursor.All(ctx, &result); err != nil {
		return []*models.Notification{}
	}
	return result
}

// MarkNotificationRead sets a notification's Read field to true.
func (s *Store) MarkNotificationRead(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := s.notifications.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"read": true}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return models.ErrNotificationNotFound
	}
	return nil
}
