package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/stockyard-dev/stockyard-wrangler/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	port   int
	limits Limits
	client *http.Client
	stop   chan struct{}
}

func New(db *store.DB, port int, limits Limits) *Server {
	s := &Server{
		db:     db,
		mux:    http.NewServeMux(),
		port:   port,
		limits: limits,
		client: &http.Client{Timeout: 30 * time.Second},
		stop:   make(chan struct{}),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Queues
	s.mux.HandleFunc("POST /api/queues", s.handleCreateQueue)
	s.mux.HandleFunc("GET /api/queues", s.handleListQueues)
	s.mux.HandleFunc("GET /api/queues/{id}", s.handleGetQueue)
	s.mux.HandleFunc("DELETE /api/queues/{id}", s.handleDeleteQueue)

	// Jobs
	s.mux.HandleFunc("POST /api/queues/{id}/jobs", s.handleEnqueueJob)
	s.mux.HandleFunc("GET /api/queues/{id}/jobs", s.handleListJobs)
	s.mux.HandleFunc("GET /api/jobs/{id}", s.handleGetJob)
	s.mux.HandleFunc("DELETE /api/jobs/{id}", s.handleCancelJob)
	s.mux.HandleFunc("POST /api/jobs/{id}/retry", s.handleRetryJob)

	// Queue stats
	s.mux.HandleFunc("GET /api/queues/{id}/stats", s.handleQueueStats)

	// DLQ
	s.mux.HandleFunc("GET /api/dlq", s.handleDLQ)

	// Status
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /ui", s.handleUI)

	s.mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"product": "stockyard-wrangler", "version": "0.1.0"})
	})
}

func (s *Server) Start() error {
	// Start the worker loop
	go s.workerLoop()
	log.Printf("[wrangler] worker loop started")

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[wrangler] listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// --- Worker loop ---

func (s *Server) workerLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.processNextJob()
		}
	}
}

func (s *Server) processNextJob() {
	job, err := s.db.ClaimNextJob()
	if err != nil {
		if err == sql.ErrNoRows {
			return // No jobs ready
		}
		log.Printf("[worker] claim error: %v", err)
		return
	}

	log.Printf("[worker] processing %s → %s", job.ID, job.CallbackURL)

	// POST the payload to the callback URL
	req, err := http.NewRequest("POST", job.CallbackURL, bytes.NewReader([]byte(job.Payload)))
	if err != nil {
		s.db.FailJob(job.ID, fmt.Sprintf("invalid callback URL: %v", err), job.MaxAttempts)
		log.Printf("[worker] %s failed: invalid URL: %v", job.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Wrangler-Job-ID", job.ID)
	req.Header.Set("X-Wrangler-Queue-ID", job.QueueID)
	req.Header.Set("X-Wrangler-Attempt", strconv.Itoa(job.Attempts))

	resp, err := s.client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("callback error: %v", err)
		s.db.FailJob(job.ID, errMsg, job.MaxAttempts)
		log.Printf("[worker] %s failed (attempt %d/%d): %v", job.ID, job.Attempts, job.MaxAttempts, err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.db.CompleteJob(job.ID)
		log.Printf("[worker] %s done (%d)", job.ID, resp.StatusCode)
	} else {
		errMsg := fmt.Sprintf("callback returned %d", resp.StatusCode)
		s.db.FailJob(job.ID, errMsg, job.MaxAttempts)
		log.Printf("[worker] %s failed (attempt %d/%d): %s", job.ID, job.Attempts, job.MaxAttempts, errMsg)
	}
}

// --- Queue handlers ---

func (s *Server) handleCreateQueue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name is required"})
		return
	}

	if s.limits.MaxQueues > 0 {
		queues, _ := s.db.ListQueues()
		if LimitReached(s.limits.MaxQueues, len(queues)) {
			writeJSON(w, 402, map[string]string{
				"error":   fmt.Sprintf("free tier limit: %d queue(s) max — upgrade to Pro", s.limits.MaxQueues),
				"upgrade": "https://stockyard.dev/wrangler/",
			})
			return
		}
	}

	q, err := s.db.CreateQueue(req.Name)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]any{"queue": q})
}

func (s *Server) handleListQueues(w http.ResponseWriter, r *http.Request) {
	queues, err := s.db.ListQueues()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if queues == nil {
		queues = []store.Queue{}
	}
	writeJSON(w, 200, map[string]any{"queues": queues, "count": len(queues)})
}

func (s *Server) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	q, err := s.db.GetQueue(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "queue not found"})
		return
	}
	writeJSON(w, 200, map[string]any{"queue": q})
}

func (s *Server) handleDeleteQueue(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.GetQueue(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": "queue not found"})
		return
	}
	s.db.DeleteQueue(id)
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// --- Job handlers ---

func (s *Server) handleEnqueueJob(w http.ResponseWriter, r *http.Request) {
	queueID := r.PathValue("id")
	if _, err := s.db.GetQueue(queueID); err != nil {
		writeJSON(w, 404, map[string]string{"error": "queue not found"})
		return
	}

	var req struct {
		CallbackURL    string `json:"callback_url"`
		Payload        any    `json:"payload"`
		RunAt          string `json:"run_at"`
		MaxAttempts    int    `json:"max_attempts"`
		BackoffSeconds int    `json:"backoff_seconds"`
		Priority       int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.CallbackURL == "" {
		writeJSON(w, 400, map[string]string{"error": "callback_url is required"})
		return
	}

	// Monthly job limit
	if s.limits.MaxJobsPerMonth > 0 {
		count, _ := s.db.MonthlyJobCount()
		if LimitReached(s.limits.MaxJobsPerMonth, count) {
			writeJSON(w, 402, map[string]string{
				"error":   fmt.Sprintf("free tier limit: %d jobs/month — upgrade to Pro", s.limits.MaxJobsPerMonth),
				"upgrade": "https://stockyard.dev/wrangler/",
			})
			return
		}
	}

	// Scheduling requires Pro
	if req.RunAt != "" && !s.limits.Scheduling {
		writeJSON(w, 402, map[string]string{
			"error":   "scheduled jobs (run_at) require Pro — upgrade at https://stockyard.dev/wrangler/",
			"upgrade": "https://stockyard.dev/wrangler/",
		})
		return
	}

	// Priority requires Pro
	if req.Priority > 0 && !s.limits.PriorityJobs {
		writeJSON(w, 402, map[string]string{
			"error":   "priority jobs require Pro — upgrade at https://stockyard.dev/wrangler/",
			"upgrade": "https://stockyard.dev/wrangler/",
		})
		return
	}

	// Cap max_attempts on free tier
	if s.limits.MaxAttempts > 0 && req.MaxAttempts > s.limits.MaxAttempts {
		req.MaxAttempts = s.limits.MaxAttempts
	}

	payloadJSON := "{}"
	if req.Payload != nil {
		b, _ := json.Marshal(req.Payload)
		payloadJSON = string(b)
	}

	job, err := s.db.EnqueueJob(queueID, payloadJSON, req.CallbackURL, req.MaxAttempts, req.BackoffSeconds, req.Priority, req.RunAt)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]any{"job": job})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	queueID := r.PathValue("id")
	statusFilter := r.URL.Query().Get("status")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	jobs, err := s.db.ListJobs(queueID, statusFilter, limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if jobs == nil {
		jobs = []store.Job{}
	}
	writeJSON(w, 200, map[string]any{"jobs": jobs, "count": len(jobs)})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.db.GetJob(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "job not found"})
		return
	}
	writeJSON(w, 200, map[string]any{"job": job})
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.GetJob(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": "job not found"})
		return
	}
	s.db.CancelJob(id)
	writeJSON(w, 200, map[string]string{"status": "cancelled"})
}

func (s *Server) handleRetryJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.GetJob(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": "job not found"})
		return
	}
	if err := s.db.RetryJob(id); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "retrying"})
}

func (s *Server) handleQueueStats(w http.ResponseWriter, r *http.Request) {
	q, err := s.db.GetQueue(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "queue not found"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"queue": q.Name, "depth": q.Pending, "running": q.Running,
		"done": q.Done, "failed": q.Failed, "dead": q.Dead,
	})
}

func (s *Server) handleDLQ(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	jobs, err := s.db.DLQ(limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if jobs == nil {
		jobs = []store.Job{}
	}
	writeJSON(w, 200, map[string]any{"dead_jobs": jobs, "count": len(jobs)})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.db.Stats())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func itoa(n int) string { return strconv.Itoa(n) }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
