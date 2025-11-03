package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func initDB() {
	var err error
	dbPath := filepath.Join(config.StoragePath, "database.db")
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database %s: %v", dbPath, err)
	}
	
	// Test connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	
	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * 60) // 5 minutes
	
	createTables()

	// Default user with correct bcrypt.GenerateFromPassword
	hashed, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash default password: %v", err)
	}
	
	if err := execDB("INSERT OR IGNORE INTO users (username, password) VALUES (?, ?)", "admin", string(hashed)); err != nil {
		log.Fatalf("Failed to insert default user: %v", err)
	}

	// Default settings
	defaultSettings := map[string]string{
		"encryption": "true",
		"autoDelete": "true", 
		"compress":   "false",
	}
	
	for k, v := range defaultSettings {
		if err := setSetting(k, v); err != nil {
			log.Printf("Failed to set %s setting: %v", k, err)
		}
	}
	
	log.Println("Database initialized successfully")
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS files (
			name TEXT PRIMARY KEY,
			size INTEGER,
			type TEXT,
			modified TIMESTAMP,
			url TEXT,
			version INTEGER,
			expiry TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS transfers (
			id TEXT PRIMARY KEY,
			file_name TEXT,
			file_size INTEGER,
			progress INTEGER,
			status TEXT,
			speed REAL,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			from_device TEXT,
			to_device TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS devices (
			id TEXT PRIMARY KEY,
			name TEXT,
			ip TEXT,
			last_seen TIMESTAMP,
			os TEXT,
			status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT,
			username TEXT,
			message TEXT,
			timestamp TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT
		)`,
	}
	
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			log.Fatalf("Failed to create table with query '%s': %v", query, err)
		}
	}
}

func insertFileInfo(f FileInfo) error {
	return execDB("INSERT OR REPLACE INTO files (name, size, type, modified, url, version, expiry) VALUES (?, ?, ?, ?, ?, ?, ?)",
		f.Name, f.Size, f.Type, f.Modified, f.URL, f.Version, f.Expiry)
}

func getAllFiles() []FileInfo {
	rows, err := db.Query("SELECT name, size, type, modified, url, version, expiry FROM files")
	if err != nil {
		log.Printf("Failed to query files: %v", err)
		return nil
	}
	defer rows.Close()
	
	var files []FileInfo
	for rows.Next() {
		var f FileInfo
		if err := rows.Scan(&f.Name, &f.Size, &f.Type, &f.Modified, &f.URL, &f.Version, &f.Expiry); err != nil {
			log.Printf("Failed to scan file: %v", err)
			continue
		}
		files = append(files, f)
	}
	
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating files: %v", err)
	}
	
	return files
}

func deleteFile(name string) error {
	return execDB("DELETE FROM files WHERE name = ?", name)
}

func insertTransfer(t Transfer) error {
	return execDB("INSERT OR REPLACE INTO transfers (id, file_name, file_size, progress, status, speed, started_at, completed_at, from_device, to_device) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.FileName, t.FileSize, t.Progress, t.Status, t.Speed, t.StartedAt, t.CompletedAt, t.FromDevice, t.ToDevice)
}

func updateTransfer(t Transfer) error {
	return execDB("UPDATE transfers SET progress=?, status=?, completed_at=? WHERE id=?", 
		t.Progress, t.Status, t.CompletedAt, t.ID)
}

func getAllTransfers() []Transfer {
	rows, err := db.Query("SELECT id, file_name, file_size, progress, status, speed, started_at, completed_at, from_device, to_device FROM transfers")
	if err != nil {
		log.Printf("Failed to query transfers: %v", err)
		return nil
	}
	defer rows.Close()
	
	var trans []Transfer
	for rows.Next() {
		var t Transfer
		if err := rows.Scan(&t.ID, &t.FileName, &t.FileSize, &t.Progress, &t.Status, 
			&t.Speed, &t.StartedAt, &t.CompletedAt, &t.FromDevice, &t.ToDevice); err != nil {
			log.Printf("Failed to scan transfer: %v", err)
			continue
		}
		trans = append(trans, t)
	}
	
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating transfers: %v", err)
	}
	
	return trans
}

func insertDevice(d Device) error {
	return execDB("INSERT OR REPLACE INTO devices (id, name, ip, last_seen, os, status) VALUES (?, ?, ?, ?, ?, ?)",
		d.ID, d.Name, d.IP, d.LastSeen, d.OS, d.Status)
}

func updateDevice(d Device) error {
	return execDB("UPDATE devices SET last_seen=?, status=? WHERE id=?", d.LastSeen, d.Status, d.ID)
}

func deleteDevice(id string) error {
	return execDB("DELETE FROM devices WHERE id=?", id)
}

func getAllDevices() []Device {
	rows, err := db.Query("SELECT id, name, ip, last_seen, os, status FROM devices")
	if err != nil {
		log.Printf("Failed to query devices: %v", err)
		return nil
	}
	defer rows.Close()
	
	var devs []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Name, &d.IP, &d.LastSeen, &d.OS, &d.Status); err != nil {
			log.Printf("Failed to scan device: %v", err)
			continue
		}
		devs = append(devs, d)
	}
	
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating devices: %v", err)
	}
	
	return devs
}

func insertChatMessage(m ChatMessage) error {
	return execDB("INSERT INTO chat_messages (type, username, message, timestamp) VALUES (?, ?, ?, ?)",
		m.Type, m.Username, m.Message, m.Timestamp)
}

func getAllChatMessages() []ChatMessage {
	rows, err := db.Query("SELECT type, username, message, timestamp FROM chat_messages ORDER BY timestamp DESC LIMIT 50")
	if err != nil {
		log.Printf("Failed to query chat messages: %v", err)
		return nil
	}
	defer rows.Close()
	
	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.Type, &m.Username, &m.Message, &m.Timestamp); err != nil {
			log.Printf("Failed to scan chat message: %v", err)
			continue
		}
		msgs = append(msgs, m)
	}
	
	// Reverse to show oldest first
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	
	return msgs
}

func getUserPassword(username string) string {
	row := db.QueryRow("SELECT password FROM users WHERE username=?", username)
	var pass string
	if err := row.Scan(&pass); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User not found: %s", username)
		} else {
			log.Printf("Failed to get user password for %s: %v", username, err)
		}
		return ""
	}
	return pass
}

func setSetting(key, value string) error {
	return execDB("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
}

func getSetting(key string) string {
	row := db.QueryRow("SELECT value FROM settings WHERE key=?", key)
	var val string
	if err := row.Scan(&val); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Failed to get setting %s: %v", key, err)
		}
		return ""
	}
	return val
}

func getAllSettings() map[string]string {
	rows, err := db.Query("SELECT key, value FROM settings")
	if err != nil {
		log.Printf("Failed to query settings: %v", err)
		return nil
	}
	defer rows.Close()
	
	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			log.Printf("Failed to scan setting: %v", err)
			continue
		}
		settings[k] = v
	}
	
	return settings
}

func execDB(query string, args ...interface{}) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query '%s': %v", query, err)
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(args...)
	if err != nil {
		return fmt.Errorf("failed to execute query '%s': %v", query, err)
	}
	
	return nil
}