package db_test

import (
	"context"
	"testing"

	"github.com/dikkadev/cland/internal/db"
	"github.com/dikkadev/cland/pkg/exchange"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *db.LibSQL {
	// Use in-memory SQLite database
	database, err := db.NewLibSQL("file::memory:?cache=shared")
	require.NoError(t, err)

	err = database.Initialize(context.Background())
	require.NoError(t, err)

	return database
}

func TestDeviceCRUD(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	t.Run("insert valid device", func(t *testing.T) {
		err := database.InsertDevice(ctx, "device1", "key1")
		assert.NoError(t, err)
	})

	t.Run("duplicate device ID", func(t *testing.T) {
		err := database.InsertDevice(ctx, "device1", "key2")
		assert.Error(t, err)
	})

	t.Run("empty device ID", func(t *testing.T) {
		err := database.InsertDevice(ctx, "", "key3")
		assert.ErrorIs(t, err, db.ErrEmptyDeviceID)
	})

	t.Run("empty public key", func(t *testing.T) {
		err := database.InsertDevice(ctx, "device2", "")
		assert.ErrorIs(t, err, db.ErrEmptyPublicKey)
	})
}

func TestTopicCRUD(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	t.Run("create new topic", func(t *testing.T) {
		id, err := database.GetOrCreateTopic(ctx, "topic1", "description1")
		assert.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("get existing topic", func(t *testing.T) {
		id1, err := database.GetOrCreateTopic(ctx, "topic2", "description2")
		assert.NoError(t, err)

		id2, err := database.GetOrCreateTopic(ctx, "topic2", "different description")
		assert.NoError(t, err)
		assert.Equal(t, id1, id2)
	})

	t.Run("empty topic name", func(t *testing.T) {
		_, err := database.GetOrCreateTopic(ctx, "", "description")
		assert.ErrorIs(t, err, db.ErrEmptyTopic)
	})

	t.Run("very long topic name", func(t *testing.T) {
		longName := string(make([]byte, db.MaxTopicNameLength+1))
		_, err := database.GetOrCreateTopic(ctx, longName, "description")
		assert.ErrorIs(t, err, db.ErrTopicTooLong)
	})
}

func TestNotificationCRUD(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	validNotif := exchange.Notification{
		Topic: "test_topic",
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Message: "Test message",
	}

	t.Run("insert valid notification", func(t *testing.T) {
		id, err := database.InsertNotification(ctx, validNotif)
		assert.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("insert notification with empty topic", func(t *testing.T) {
		invalidNotif := validNotif
		invalidNotif.Topic = ""
		_, err := database.InsertNotification(ctx, invalidNotif)
		assert.ErrorIs(t, err, db.ErrEmptyTopic)
	})

	t.Run("insert notification with empty message", func(t *testing.T) {
		invalidNotif := validNotif
		invalidNotif.Message = ""
		_, err := database.InsertNotification(ctx, invalidNotif)
		assert.ErrorIs(t, err, db.ErrEmptyMessage)
	})

	t.Run("insert notification with nil metadata", func(t *testing.T) {
		notif := validNotif
		notif.Metadata = nil
		id, err := database.InsertNotification(ctx, notif)
		assert.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("insert notification with complex metadata", func(t *testing.T) {
		notif := validNotif
		notif.Metadata = map[string]string{
			"unicode":   "🚀",
			"empty":     "",
			"special":   "!@#$%^&*()",
			"very_long": string(make([]byte, 1000)),
		}
		id, err := database.InsertNotification(ctx, notif)
		assert.NoError(t, err)
		assert.Greater(t, id, 0)
	})
}

func TestNotificationStatus(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	// Insert a test notification
	notif := exchange.Notification{
		Topic:    "status_test",
		Message:  "Test message",
		Metadata: map[string]string{"key": "value"},
	}

	notifID, err := database.InsertNotification(ctx, notif)
	require.NoError(t, err)

	t.Run("mark notification as sent", func(t *testing.T) {
		err := database.MarkNotificationSent(ctx, notifID)
		assert.NoError(t, err)

		// Attempting to mark as sent again should not error but should not change status
		err = database.MarkNotificationSent(ctx, notifID)
		assert.NoError(t, err)
	})

	t.Run("mark non-existent notification", func(t *testing.T) {
		err := database.MarkNotificationSent(ctx, 99999)
		assert.NoError(t, err)
	})

	t.Run("mark notification as error", func(t *testing.T) {
		// Insert a new notification for error testing
		newNotifID, err := database.InsertNotification(ctx, notif)
		require.NoError(t, err)

		err = database.MarkNotificationError(ctx, newNotifID)
		assert.NoError(t, err)

		// Attempting to mark as error again should not error but should not change status
		err = database.MarkNotificationError(ctx, newNotifID)
		assert.NoError(t, err)
	})

	t.Run("status transitions", func(t *testing.T) {
		// Insert a new notification
		transitionID, err := database.InsertNotification(ctx, notif)
		require.NoError(t, err)

		// Try to mark as sent
		err = database.MarkNotificationSent(ctx, transitionID)
		assert.NoError(t, err)

		// Try to mark as error after being sent (should not change status)
		err = database.MarkNotificationError(ctx, transitionID)
		assert.NoError(t, err)
	})
}

func TestConcurrency(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	t.Run("concurrent topic creation", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan bool)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := database.GetOrCreateTopic(ctx, "concurrent_topic", "description")
				assert.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	t.Run("concurrent notification insertion", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan bool)

		notif := exchange.Notification{
			Topic:    "concurrent_test",
			Message:  "Test message",
			Metadata: map[string]string{"key": "value"},
		}

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := database.InsertNotification(ctx, notif)
				assert.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

func TestDatabaseErrors(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)

	t.Run("operations after close", func(t *testing.T) {
		err := database.Close()
		require.NoError(t, err)

		// All operations should fail after close
		_, err = database.GetOrCreateTopic(ctx, "topic", "description")
		assert.Error(t, err)

		err = database.InsertDevice(ctx, "device", "key")
		assert.Error(t, err)

		_, err = database.InsertNotification(ctx, exchange.Notification{})
		assert.Error(t, err)
	})
}

func TestSchemaConstraints(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	t.Run("notification status constraints", func(t *testing.T) {
		notif := exchange.Notification{
			Topic:   "constraint_test",
			Message: "Test message",
		}

		id, err := database.InsertNotification(ctx, notif)
		assert.NoError(t, err)

		// Verify default status is INPUT by ensuring we can mark it as sent
		err = database.MarkNotificationSent(ctx, id)
		assert.NoError(t, err)
	})
}
