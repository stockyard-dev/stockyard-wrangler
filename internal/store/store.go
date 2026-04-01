package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ conn *sql.DB }

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	conn, err := sql.Open("sqlite", filepath.Join(dataDir, "wrangler.db"))
	if err != nil {
		return nil, err
	}
	conn.Exec("PRAGMA journal_mode=WAL")
	conn.Exec("PRAGMA busy_timeout=5000")
	conn.SetMaxOpenConns(4)
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Conn() *sql.DB { return db.conn }
func (db *DB) Close() error  { return db.conn.Close() }

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
CREATE TABLE IF NOT EXISTS queues (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    queue_id TEXT NOT NULL,
    payload TEXT DEFAULT '{}',
    callback_url TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    priority INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    backoff_seconds INTEGER DEFAULT 60,
    run_at TEXT DEFAULT (datetime('now')),
    started_at TEXT DEFAULT '',
    completed_at TEXT DEFAULT '',
    last_error TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_jobs_queue ON jobs(queue_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_run ON jobs(status, run_at);
CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(priority DESC, run_at ASC);
`)
	return err
}

// --- Queue types ---

type Queue struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Pending   int    `json:"pending"`
	Running   int    `json:"running"`
	Done      int    `json:"done"`
	Failed    int    `json:"failed"`
	Dead      int    `json:"dead"`
}

func (db *DB) CreateQueue(name string) (*Queue, error) {
	id := "q_" + genID(8)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec("INSERT INTO queues (id,name,created_at) VALUES (?,?,?)", id, name, now)
	if err != nil {
		return nil, err
	}
	return &Queue{ID: id, Name: name, CreatedAt: now}, nil
}

func (db *DB) ListQueues() ([]Queue, error) {
	rows, err := db.conn.Query("SELECT id,name,created_at FROM queues ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Queue
	for rows.Next() {
		var q Queue
		rows.Scan(&q.ID, &q.Name, &q.CreatedAt)
		db.fillQueueCounts(&q)
		out = append(out, q)
	}
	return out, rows.Err()
}

func (db *DB) GetQueue(id string) (*Queue, error) {
	var q Queue
	err := db.conn.QueryRow("SELECT id,name,created_at FROM queues WHERE id=?", id).
		Scan(&q.ID, &q.Name, &q.CreatedAt)
	if err != nil {
		return nil, err
	}
	db.fillQueueCounts(&q)
	return &q, nil
}

func (db *DB) DeleteQueue(id string) error {
	db.conn.Exec("DELETE FROM jobs WHERE queue_id=?", id)
	_, err := db.conn.Exec("DELETE FROM queues WHERE id=?", id)
	return err
}

func (db *DB) fillQueueCounts(q *Queue) {
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE queue_id=? AND status='pending'", q.ID).Scan(&q.Pending)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE queue_id=? AND status='running'", q.ID).Scan(&q.Running)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE queue_id=? AND status='done'", q.ID).Scan(&q.Done)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE queue_id=? AND status='failed'", q.ID).Scan(&q.Failed)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE queue_id=? AND status='dead'", q.ID).Scan(&q.Dead)
}

// --- Job types ---

type Job struct {
	ID             string `json:"id"`
	QueueID        string `json:"queue_id"`
	Payload        string `json:"payload"`
	CallbackURL    string `json:"callback_url"`
	Status         string `json:"status"`
	Priority       int    `json:"priority"`
	Attempts       int    `json:"attempts"`
	MaxAttempts    int    `json:"max_attempts"`
	BackoffSeconds int    `json:"backoff_seconds"`
	RunAt          string `json:"run_at"`
	StartedAt      string `json:"started_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty"`
	LastError      string `json:"last_error,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func (db *DB) EnqueueJob(queueID, payload, callbackURL string, maxAttempts, backoffSec, priority int, runAt string) (*Job, error) {
	id := "job_" + genID(10)
	now := time.Now().UTC().Format(time.RFC3339)
	if runAt == "" {
		runAt = now
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if backoffSec <= 0 {
		backoffSec = 60
	}
	_, err := db.conn.Exec(`INSERT INTO jobs (id,queue_id,payload,callback_url,status,priority,max_attempts,backoff_seconds,run_at,created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`, id, queueID, payload, callbackURL, "pending", priority, maxAttempts, backoffSec, runAt, now)
	if err != nil {
		return nil, err
	}
	return &Job{ID: id, QueueID: queueID, Payload: payload, CallbackURL: callbackURL,
		Status: "pending", Priority: priority, MaxAttempts: maxAttempts,
		BackoffSeconds: backoffSec, RunAt: runAt, CreatedAt: now}, nil
}

func (db *DB) ListJobs(queueID, statusFilter string, limit int) ([]Job, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows *sql.Rows
	var err error
	if statusFilter != "" {
		rows, err = db.conn.Query(`SELECT id,queue_id,payload,callback_url,status,priority,attempts,max_attempts,backoff_seconds,
			run_at,started_at,completed_at,last_error,created_at
			FROM jobs WHERE queue_id=? AND status=? ORDER BY priority DESC, run_at ASC LIMIT ?`, queueID, statusFilter, limit)
	} else {
		rows, err = db.conn.Query(`SELECT id,queue_id,payload,callback_url,status,priority,attempts,max_attempts,backoff_seconds,
			run_at,started_at,completed_at,last_error,created_at
			FROM jobs WHERE queue_id=? ORDER BY created_at DESC LIMIT ?`, queueID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (db *DB) GetJob(id string) (*Job, error) {
	var j Job
	err := db.conn.QueryRow(`SELECT id,queue_id,payload,callback_url,status,priority,attempts,max_attempts,backoff_seconds,
		run_at,started_at,completed_at,last_error,created_at FROM jobs WHERE id=?`, id).
		Scan(&j.ID, &j.QueueID, &j.Payload, &j.CallbackURL, &j.Status, &j.Priority,
			&j.Attempts, &j.MaxAttempts, &j.BackoffSeconds, &j.RunAt, &j.StartedAt,
			&j.CompletedAt, &j.LastError, &j.CreatedAt)
	return &j, err
}

func (db *DB) CancelJob(id string) error {
	_, err := db.conn.Exec("UPDATE jobs SET status='cancelled' WHERE id=? AND status IN ('pending','failed')", id)
	return err
}

// ClaimNextJob atomically picks the next ready job and marks it running.
func (db *DB) ClaimNextJob() (*Job, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var j Job
	err = tx.QueryRow(`SELECT id,queue_id,payload,callback_url,status,priority,attempts,max_attempts,backoff_seconds,
		run_at,started_at,completed_at,last_error,created_at
		FROM jobs WHERE status='pending' AND run_at<=? ORDER BY priority DESC, run_at ASC LIMIT 1`, now).
		Scan(&j.ID, &j.QueueID, &j.Payload, &j.CallbackURL, &j.Status, &j.Priority,
			&j.Attempts, &j.MaxAttempts, &j.BackoffSeconds, &j.RunAt, &j.StartedAt,
			&j.CompletedAt, &j.LastError, &j.CreatedAt)
	if err != nil {
		return nil, err // sql.ErrNoRows when queue is empty
	}

	_, err = tx.Exec("UPDATE jobs SET status='running', started_at=?, attempts=attempts+1 WHERE id=?", now, j.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	j.Status = "running"
	j.StartedAt = now
	j.Attempts++
	return &j, nil
}

func (db *DB) CompleteJob(id string) {
	now := time.Now().UTC().Format(time.RFC3339)
	db.conn.Exec("UPDATE jobs SET status='done', completed_at=? WHERE id=?", now, id)
}

func (db *DB) FailJob(id, errMsg string, maxAttempts int) {
	now := time.Now().UTC().Format(time.RFC3339)
	var attempts int
	db.conn.QueryRow("SELECT attempts FROM jobs WHERE id=?", id).Scan(&attempts)

	if attempts >= maxAttempts {
		// Move to dead letter queue
		db.conn.Exec("UPDATE jobs SET status='dead', last_error=?, completed_at=? WHERE id=?", errMsg, now, id)
	} else {
		// Reschedule with backoff
		var backoff int
		db.conn.QueryRow("SELECT backoff_seconds FROM jobs WHERE id=?", id).Scan(&backoff)
		nextRun := time.Now().Add(time.Duration(backoff*attempts) * time.Second).UTC().Format(time.RFC3339)
		db.conn.Exec("UPDATE jobs SET status='pending', last_error=?, run_at=? WHERE id=?", errMsg, nextRun, id)
	}
}

func (db *DB) RetryJob(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec("UPDATE jobs SET status='pending', run_at=?, attempts=0, last_error='' WHERE id=? AND status IN ('dead','failed','cancelled')",
		now, id)
	return err
}

// DLQ returns all dead jobs
func (db *DB) DLQ(limit int) ([]Job, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.conn.Query(`SELECT id,queue_id,payload,callback_url,status,priority,attempts,max_attempts,backoff_seconds,
		run_at,started_at,completed_at,last_error,created_at
		FROM jobs WHERE status='dead' ORDER BY completed_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (db *DB) MonthlyJobCount() (int, error) {
	cutoff := time.Now().AddDate(0, -1, 0).Format("2006-01-02 15:04:05")
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE created_at >= ?", cutoff).Scan(&count)
	return count, err
}

// --- Stats ---

func (db *DB) Stats() map[string]any {
	var queues, total, pending, running, done, failed, dead int
	db.conn.QueryRow("SELECT COUNT(*) FROM queues").Scan(&queues)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs").Scan(&total)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE status='pending'").Scan(&pending)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE status='running'").Scan(&running)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE status='done'").Scan(&done)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE status='failed'").Scan(&failed)
	db.conn.QueryRow("SELECT COUNT(*) FROM jobs WHERE status='dead'").Scan(&dead)
	return map[string]any{
		"queues": queues, "total_jobs": total, "pending": pending,
		"running": running, "done": done, "failed": failed, "dead": dead,
	}
}

func (db *DB) Cleanup(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	res, err := db.conn.Exec("DELETE FROM jobs WHERE status IN ('done','dead','cancelled') AND completed_at < ? AND completed_at != ''", cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// --- Helpers ---

func scanJobs(rows *sql.Rows) ([]Job, error) {
	var out []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.QueueID, &j.Payload, &j.CallbackURL, &j.Status, &j.Priority,
			&j.Attempts, &j.MaxAttempts, &j.BackoffSeconds, &j.RunAt, &j.StartedAt,
			&j.CompletedAt, &j.LastError, &j.CreatedAt); err != nil {
			continue
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func genID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
