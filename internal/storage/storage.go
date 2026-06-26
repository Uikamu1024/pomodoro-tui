package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Session struct {
	Kind      string // "work" or "break"
	Duration  time.Duration
	StartedAt time.Time
	Completed bool
}

type Task struct {
	ID        int64
	Title     string
	Done      bool
	CreatedAt time.Time
}

type Store struct {
	db *sql.DB
}

func defaultPath() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, ".pomodoro-tui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.db"), nil
}

func Open() (*Store, error) {
	path, err := defaultPath()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind TEXT NOT NULL,
		duration_seconds INTEGER NOT NULL,
		started_at DATETIME NOT NULL,
		completed INTEGER NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		done INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Record(sess Session) error {
	_, err := s.db.Exec(
		`INSERT INTO sessions (kind, duration_seconds, started_at, completed) VALUES (?, ?, ?, ?)`,
		sess.Kind, int64(sess.Duration.Seconds()), sess.StartedAt, sess.Completed,
	)
	return err
}

func (s *Store) Recent(limit int) ([]Session, error) {
	rows, err := s.db.Query(
		`SELECT kind, duration_seconds, started_at, completed FROM sessions ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		var sess Session
		var seconds int64
		if err := rows.Scan(&sess.Kind, &seconds, &sess.StartedAt, &sess.Completed); err != nil {
			return nil, err
		}
		sess.Duration = time.Duration(seconds) * time.Second
		out = append(out, sess)
	}
	return out, rows.Err()
}

func (s *Store) AddTask(title string) (Task, error) {
	now := time.Now()
	res, err := s.db.Exec(
		`INSERT INTO tasks (title, done, created_at) VALUES (?, 0, ?)`, title, now,
	)
	if err != nil {
		return Task{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Task{}, err
	}
	return Task{ID: id, Title: title, Done: false, CreatedAt: now}, nil
}

func (s *Store) ListTasks() ([]Task, error) {
	rows, err := s.db.Query(
		`SELECT id, title, done, created_at FROM tasks ORDER BY done ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) ToggleTask(id int64) error {
	_, err := s.db.Exec(`UPDATE tasks SET done = NOT done WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteTask(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

func (s *Store) CompletedWorkCount() (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE kind = 'work' AND completed = 1`,
	).Scan(&count)
	return count, err
}
