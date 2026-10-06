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
	ID     string `json:"id"`
	Status string `json:"status"`
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
		// --- CANCELLATION CHECK 1 ---
		// The job might have been sitting in the queue for a while.
		// Let's fetch the most recent status from our store.
		latestJob, _ := store.Get(job.ID)
		if latestJob.Status == "CANCELLED" {
			fmt.Printf("[Worker %d] Skipping cancelled job %s\n", workerID, job.ID)
			continue // Skip processing!
		}

		fmt.Printf("[Worker %d] Started processing job %s\n", workerID, job.ID)
		
		latestJob.Status = "PROCESSING"
		store.Set(latestJob)

		// Simulate hard work
		time.Sleep(5 * time.Second)

		// --- CANCELLATION CHECK 2 ---
		// Did the user cancel it while we were working on it?
		latestJob, _ = store.Get(job.ID)
		if latestJob.Status == "CANCELLED" {
			fmt.Printf("[Worker %d] Stopped processing cancelled job %s\n", workerID, job.ID)
			continue
		}

		latestJob.Status = "COMPLETED"
		store.Set(latestJob)
		fmt.Printf("[Worker %d] Finished processing job %s\n", workerID, job.ID)
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
		ID:     newID,
		Status: "PENDING",
	}

	store.Set(newJob)
	jobQueue <- newJob

	fmt.Printf("API: Job %s added to queue\n", newID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newJob)
}

// jobActionRouter handles /jobs/{id}, /jobs/{id}/cancel, and /jobs/{id}/retry
func jobActionRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/jobs/")
	parts := strings.Split(path, "/") // Splitting "/jobs/1/cancel" into ["1", "cancel"]

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

	// 1. GET /jobs/{id} - Check Status
	if len(parts) == 1 && r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(job)
		return
	}

	// 2. POST /jobs/{id}/cancel - Cancel a job
	if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
		if job.Status == "PENDING" || job.Status == "PROCESSING" {
			job.Status = "CANCELLED"
			store.Set(job)
		}
		json.NewEncoder(w).Encode(job)
		return
	}

	// 3. POST /jobs/{id}/retry - Retry a cancelled or failed job
	if len(parts) == 2 && parts[1] == "retry" && r.Method == http.MethodPost {
		if job.Status == "CANCELLED" || job.Status == "FAILED" {
			job.Status = "PENDING"
			store.Set(job)
			
			// Push it back to the queue!
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
