package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// --- 1. Data Structures ---

type Job struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// --- 2. Shared State ---

// SharedJobStore wraps our raw map into a dedicated object.
// We are explicitly setting this up to introduce a Mutex later.
type SharedJobStore struct {
	data map[string]Job
}

// Set is a "method" on the SharedJobStore struct.
// It saves a job to the map.
func (s *SharedJobStore) Set(job Job) {
	s.data[job.ID] = job
}

// Get retrieves a job from the map.
func (s *SharedJobStore) Get(id string) (Job, bool) {
	job, exists := s.data[id]
	return job, exists
}

// We initialize our global store using the new struct.
var store = SharedJobStore{
	data: make(map[string]Job),
}

var jobCounter int
var jobQueue = make(chan Job, 100)

// --- 3. Worker Pool ---

func worker(workerID int) {
	for job := range jobQueue {
		fmt.Printf("[Worker %d] Started processing job %s\n", workerID, job.ID)

		// Update via our new SharedJobStore method
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

	// Read via our new SharedJobStore method
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
