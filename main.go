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

// --- 2. Shared State (Now Thread-Safe!) ---

type SharedJobStore struct {
	mu   sync.Mutex     // <--- OUR MUTEX!
	data map[string]Job
}

func (s *SharedJobStore) Set(job Job) {
	// 1. Grab the lock. If another goroutine has it, wait here.
	s.mu.Lock()
	
	// 2. We are now guaranteed to be the only goroutine touching the data.
	s.data[job.ID] = job
	
	// 3. Release the lock for the next goroutine.
	s.mu.Unlock()
}

func (s *SharedJobStore) Get(id string) (Job, bool) {
	s.mu.Lock()
	// Using `defer` tells Go: "Make sure to run Unlock() the very moment 
	// this function finishes, no matter how it exits." This is best practice.
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
		fmt.Printf("[Worker %d] Started processing job %s\n", workerID, job.ID)

		job.Status = "PROCESSING"
		store.Set(job)

		time.Sleep(5 * time.Second)

		job.Status = "COMPLETED"
		store.Set(job)

		fmt.Printf("[Worker %d] Finished processing job %s\n", workerID, job.ID)
	}
}

// --- 4. HTTP Handlers ---

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		jobCounter++ // Note: In a real production app, this counter needs a lock too!
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
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func jobStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/jobs/")
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
	json.NewEncoder(w).Encode(job)
}

func main() {
	for i := 1; i <= 3; i++ {
		go worker(i)
	}

	http.HandleFunc("/jobs", jobsHandler)
	http.HandleFunc("/jobs/", jobStatusHandler)

	fmt.Println("Starting server on port 8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
