//go:build linux

package proxy

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type User struct {
	Username  string
	Password  string
	Upload    int64
	Download  int64
	Enabled   bool
	CreatedAt time.Time
}

type Store struct {
	db *sql.DB
	mu sync.Mutex // protects in-memory traffic deltas between flushes
	// Accumulated traffic deltas since last flush, keyed by username.
	traffic map[string][2]int64 // [0]=upload, [1]=download

	connMu sync.Mutex                // protects online connection tracking
	conns  map[string]map[string]int // username -> sourceIP -> connection count
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return &Store{
		db:      db,
		traffic: make(map[string][2]int64),
		conns:   make(map[string]map[string]int),
	}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS admin (
			id               INTEGER PRIMARY KEY CHECK (id = 1),
			username         TEXT NOT NULL,
			password         TEXT NOT NULL,
			password_changed INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS users (
			username   TEXT PRIMARY KEY,
			password   TEXT NOT NULL,
			upload     INTEGER NOT NULL DEFAULT 0,
			download   INTEGER NOT NULL DEFAULT 0,
			enabled    INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
	`)
	if err != nil {
		return err
	}

	// Add password_changed column if missing (upgrade from older schema)
	db.Exec("ALTER TABLE admin ADD COLUMN password_changed INTEGER NOT NULL DEFAULT 0")

	// Ensure a default admin row exists.
	// If no row exists, insert default admin/admin.
	// If a row exists from before the password_changed feature (column was just added
	// with default 0), reset it to admin/admin so the first-login flow works.
	_, err = db.Exec(`
		INSERT INTO admin (id, username, password, password_changed) VALUES (1, 'admin', 'admin', 0)
		ON CONFLICT(id) DO UPDATE SET
			username = CASE WHEN admin.password_changed = 0 THEN 'admin' ELSE admin.username END,
			password = CASE WHEN admin.password_changed = 0 THEN 'admin' ELSE admin.password END
	`)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

// --- Admin ---

func (s *Store) SetAdmin(username, password string) error {
	_, err := s.db.Exec(`
		INSERT INTO admin (id, username, password, password_changed) VALUES (1, ?, ?, 1)
		ON CONFLICT(id) DO UPDATE SET username=excluded.username, password=excluded.password, password_changed=1
	`, username, password)
	return err
}

func (s *Store) GetAdmin() (username, password string, err error) {
	err = s.db.QueryRow("SELECT username, password FROM admin WHERE id=1").Scan(&username, &password)
	if err == sql.ErrNoRows {
		return "", "", fmt.Errorf("admin not configured")
	}
	return
}

// IsDefaultPassword returns true if the admin has not changed the default password.
func (s *Store) IsDefaultPassword() bool {
	var changed int
	err := s.db.QueryRow("SELECT password_changed FROM admin WHERE id=1").Scan(&changed)
	return err != nil || changed == 0
}

// --- Users ---

func (s *Store) AddUser(username, password string) error {
	_, err := s.db.Exec(
		"INSERT INTO users (username, password) VALUES (?, ?)",
		username, password,
	)
	if err != nil {
		return fmt.Errorf("user %q already exists or db error: %w", username, err)
	}
	return nil
}

func (s *Store) DeleteUser(username string) error {
	res, err := s.db.Exec("DELETE FROM users WHERE username=?", username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user %q not found", username)
	}
	return nil
}

func (s *Store) Authenticate(username, password string) bool {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username=? AND password=? AND enabled=1",
		username, password,
	).Scan(&count)
	return err == nil && count > 0
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query("SELECT username, password, upload, download, enabled, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var enabled int
		if err := rows.Scan(&u.Username, &u.Password, &u.Upload, &u.Download, &enabled, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Enabled = enabled == 1
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) GetUser(username string) (*User, error) {
	var u User
	var enabled int
	err := s.db.QueryRow(
		"SELECT username, password, upload, download, enabled, created_at FROM users WHERE username=?",
		username,
	).Scan(&u.Username, &u.Password, &u.Upload, &u.Download, &enabled, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user %q not found", username)
	}
	if err != nil {
		return nil, err
	}
	u.Enabled = enabled == 1
	return &u, nil
}

func (s *Store) HasUsers() bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM users WHERE enabled=1").Scan(&count)
	return count > 0
}

// UpdateUser updates a user's password and/or enabled status.
// Only non-nil fields are updated.
func (s *Store) UpdateUser(username string, password *string, enabled *bool) error {
	// Check user exists
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", username).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("user %q not found", username)
	}

	if password != nil {
		if _, err := s.db.Exec("UPDATE users SET password=? WHERE username=?", *password, username); err != nil {
			return err
		}
	}
	if enabled != nil {
		enabledInt := 0
		if *enabled {
			enabledInt = 1
		}
		if _, err := s.db.Exec("UPDATE users SET enabled=? WHERE username=?", enabledInt, username); err != nil {
			return err
		}
	}
	return nil
}

// ConnectUser tracks a new connection from sourceIP for the given user.
func (s *Store) ConnectUser(username, sourceIP string) {
	s.connMu.Lock()
	if s.conns[username] == nil {
		s.conns[username] = make(map[string]int)
	}
	s.conns[username][sourceIP]++
	s.connMu.Unlock()
}

// DisconnectUser removes a connection from sourceIP for the given user.
func (s *Store) DisconnectUser(username, sourceIP string) {
	s.connMu.Lock()
	s.conns[username][sourceIP]--
	if s.conns[username][sourceIP] <= 0 {
		delete(s.conns[username], sourceIP)
	}
	if len(s.conns[username]) == 0 {
		delete(s.conns, username)
	}
	s.connMu.Unlock()
}

// OnlineUsers returns a map of usernames to their unique device (source IP) counts.
func (s *Store) OnlineUsers() map[string]int {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	result := make(map[string]int, len(s.conns))
	for k, ips := range s.conns {
		result[k] = len(ips)
	}
	return result
}

// AddTraffic accumulates traffic in memory. Call Flush to persist.
func (s *Store) AddTraffic(username string, upload, download int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.traffic[username]
	t[0] += upload
	t[1] += download
	s.traffic[username] = t
}

// Flush writes accumulated traffic deltas to the database.
func (s *Store) Flush() error {
	s.mu.Lock()
	pending := s.traffic
	s.traffic = make(map[string][2]int64)
	s.mu.Unlock()

	if len(pending) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("UPDATE users SET upload=upload+?, download=download+? WHERE username=?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for username, t := range pending {
		stmt.Exec(t[0], t[1], username)
	}
	return tx.Commit()
}
