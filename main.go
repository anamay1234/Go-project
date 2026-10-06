package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// --- 1. Data Structures ---

type Job struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Attempt int    `json:"attempt"` // <-- Added to track which attempt this is
}

// --- 2. Shared State (Thread-Safe) ---

type SharedJobStore struct {
	mu   sync.Mutex
	data map[string]Job
}

func (s *SharedJobStore) Set(job Job) {
	s.mu.Lock()
	s.data[job.ID] = job
	s.mu.Unlock()
}

func (s *SharedJobStore) Get(id string) (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, exists := s.data[id]
	return job, exists
}

var store = SharedJobStore{
	data: make(map[string]Job),
}

var jobCounter int
var jobQueue = make(chan Job, 100)

// --- 3. Worker Pool ---

func worker(workerID int) {
	for job := range jobQueue {
		latestJob, _ := store.Get(job.ID)
		if latestJob.Status == "CANCELLED" {
			fmt.Printf("[Worker %d] Skipping cancelled job %s\n", workerID, job.ID)
			continue
		}

		// Save the attempt number we are currently working on
		currentAttempt := latestJob.Attempt

		fmt.Printf("[Worker %d] Started processing job %s (Attempt %d)\n", workerID, job.ID, currentAttempt)
		
		latestJob.Status = "PROCESSING"
		store.Set(latestJob)

		time.Sleep(5 * time.Second)

		// Did the user cancel it while we were working?
		latestJob, _ = store.Get(job.ID)
		if latestJob.Status == "CANCELLED" {
			fmt.Printf("[Worker %d] Stopped processing cancelled job %s\n", workerID, job.ID)
			continue
		}

		// MAGIC FIX: Did someone hit retry while we were sleeping, causing a newer attempt?
		if latestJob.Attempt > currentAttempt {
			fmt.Printf("[Worker %d] Bailing out! A newer attempt (%d) is running for job %s\n", workerID, latestJob.Attempt, job.ID)
			continue
		}

		latestJob.Status = "COMPLETED"
		store.Set(latestJob)
		fmt.Printf("[Worker %d] Finished processing job %s (Attempt %d)\n", workerID, job.ID, currentAttempt)
	}
}

// --- 4. HTTP Handlers ---

func createJobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobCounter++
	newID := fmt.Sprintf("%d", jobCounter)

	newJob := Job{
		ID:      newID,
		Status:  "PENDING",
		Attempt: 1, // First attempt!
	}

	store.Set(newJob)
	jobQueue <- newJob

	fmt.Printf("API: Job %s added to queue\n", newID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newJob)
}

func jobActionRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/jobs/")
	parts := strings.Split(path, "/")

	id := parts[0]
	if id == "" {
		http.Error(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	job, exists := store.Get(id)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(parts) == 1 && r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(job)
		return
	}

	if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
		if job.Status == "PENDING" || job.Status == "PROCESSING" {
			job.Status = "CANCELLED"
			store.Set(job)
		}
		json.NewEncoder(w).Encode(job)
		return
	}

	if len(parts) == 2 && parts[1] == "retry" && r.Method == http.MethodPost {
		if job.Status == "CANCELLED" || job.Status == "FAILED" {
			job.Status = "PENDING"
			job.Attempt++ // Increase the attempt number!
			store.Set(job)
			jobQueue <- job
		}
		json.NewEncoder(w).Encode(job)
		return
	}

	http.Error(w, "Invalid endpoint", http.StatusNotFound)
}

func main() {
	for i := 1; i <= 3; i++ {
		go worker(i)
	}

	http.HandleFunc("/jobs", createJobHandler)
	http.HandleFunc("/jobs/", jobActionRouter)

	fmt.Println("Starting server on port 8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
