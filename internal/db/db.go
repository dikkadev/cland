package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dikkadev/cland/pkg/exchange"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

const (
	MaxTopicNameLength = 255
)

var (
	ErrEmptyDeviceID  = errors.New("device ID cannot be empty")
	ErrEmptyPublicKey = errors.New("public key cannot be empty")
	ErrEmptyTopic     = errors.New("topic name cannot be empty")
	ErrTopicTooLong   = errors.New("topic name exceeds maximum length")
	ErrEmptyMessage   = errors.New("notification message cannot be empty")
)

type LibSQL struct {
	db *sql.DB
}

func NewLibSQL(url string) (*LibSQL, error) {
	db, err := sql.Open("libsql", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &LibSQL{db: db}, nil
}

func (s *LibSQL) Initialize(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, CREATE_ALL_TABLES); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return tx.Commit()
}

func (s *LibSQL) Close() error {
	return s.db.Close()
}

func validateDevice(deviceID, publicKey string) error {
	if deviceID == "" {
		return ErrEmptyDeviceID
	}
	if publicKey == "" {
		return ErrEmptyPublicKey
	}
	return nil
}

func validateTopic(topicName string) error {
	if topicName == "" {
		return ErrEmptyTopic
	}
	if len(topicName) > MaxTopicNameLength {
		return ErrTopicTooLong
	}
	return nil
}

func validateNotification(notif exchange.Notification) error {
	if err := validateTopic(notif.Topic); err != nil {
		return err
	}
	if notif.Message == "" {
		return ErrEmptyMessage
	}
	return nil
}

func (s *LibSQL) InsertDevice(ctx context.Context, deviceID, publicKey string) error {
	if err := validateDevice(deviceID, publicKey); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "INSERT INTO devices (device_id, public_key) VALUES (?, ?)",
		deviceID, publicKey); err != nil {
		return fmt.Errorf("failed to insert device: %w", err)
	}

	return tx.Commit()
}

func (s *LibSQL) GetOrCreateTopic(ctx context.Context, topicName string, description string) (int, error) {
	if err := validateTopic(topicName); err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var topicID int64
	err = tx.QueryRowContext(ctx, "SELECT topic_id FROM topics WHERE topic_name = ?", topicName).Scan(&topicID)
	if err == nil {
		tx.Commit()
		return int(topicID), nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to get topic: %w", err)
	}

	res, err := tx.ExecContext(ctx, "INSERT INTO topics (topic_name, description) VALUES (?, ?)",
		topicName, description)
	if err != nil {
		return 0, fmt.Errorf("failed to insert topic: %w", err)
	}

	topicID, err = res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get topic ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(topicID), nil
}

func (s *LibSQL) InsertNotification(ctx context.Context, notif exchange.Notification) (int, error) {
	if err := validateNotification(notif); err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	topicID, err := s.GetOrCreateTopic(ctx, notif.Topic, "")
	if err != nil {
		return 0, fmt.Errorf("failed to get or create topic: %w", err)
	}

	metadataJSON, err := json.Marshal(notif.Metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata into JSON: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		"INSERT INTO notifications (topic_id, message, metadata) VALUES (?, ?, ?)",
		topicID, notif.Message, metadataJSON)
	if err != nil {
		return 0, fmt.Errorf("failed to insert notification: %w", err)
	}

	notificationID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get notification ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(notificationID), nil
}

func (s *LibSQL) MarkNotificationSent(ctx context.Context, notificationID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"UPDATE notifications SET status = ? WHERE notification_id = ? AND status = ?",
		NotificationStatusSent, notificationID, NotificationStatusInput)
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return nil
	}

	return tx.Commit()
}

func (s *LibSQL) MarkNotificationError(ctx context.Context, notificationID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"UPDATE notifications SET status = ? WHERE notification_id = ? AND status = ?",
		NotificationStatusError, notificationID, NotificationStatusInput)
	if err != nil {
		return fmt.Errorf("failed to mark notification as error: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return nil
	}

	return tx.Commit()
}
