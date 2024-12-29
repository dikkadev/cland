package db

type NotificationStatus string

const (
	NotificationStatusInput NotificationStatus = "INPUT"
	NotificationStatusSent  NotificationStatus = "SENT"
	NotificationStatusError NotificationStatus = "ERROR"
)

const CREATE_DEVICES_TABLE = `
CREATE TABLE IF NOT EXISTS devices (
	device_id TEXT PRIMARY KEY,
	public_key TEXT NOT NULL,
	registration_date DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const CREATE_TOPICS_TABLE = `
CREATE TABLE IF NOT EXISTS topics (
	topic_id INTEGER PRIMARY KEY AUTOINCREMENT,
	topic_name TEXT NOT NULL UNIQUE,
	description TEXT,
	creation_date DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const CREATE_NOTIFICATIONS_TABLE = `
CREATE TABLE IF NOT EXISTS notifications (
	notification_id INTEGER PRIMARY KEY AUTOINCREMENT,
	topic_id INTEGER NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
	message TEXT NOT NULL,
	metadata TEXT,
	status TEXT CHECK(status IN ('INPUT', 'SENT', 'ERROR')) DEFAULT 'INPUT',
	FOREIGN KEY(topic_id) REFERENCES topics(topic_id)
);
`

const CREATE_ALL_TABLES = CREATE_DEVICES_TABLE + CREATE_TOPICS_TABLE + CREATE_NOTIFICATIONS_TABLE
