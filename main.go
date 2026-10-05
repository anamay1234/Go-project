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

// --- 2. In-Memory Storage & Queue ---

var jobStore = make(map[string]Job)
var jobCounter int
var jobQueue = make(chan Job, 100)

// --- 3. Worker Pool ---

// worker simulates a background process that pulls jobs from the queue and works on them.
func worker(workerID int) {
	// The range keyword on a channel will continuously pull items off the queue
	// as soon as they arrive. If the queue is empty, the worker just waits (blocks).
	for job := range jobQueue {
		fmt.Printf("[Worker %d] Started processing job %s\n", workerID, job.ID)

		// 1. Update status to PROCESSING
		job.Status = "PROCESSING"
		jobStore[job.ID] = job

		// 2. Simulate some hard work (like video encoding or sending emails)
		time.Sleep(5 * time.Second)

		// 3. Update status to COMPLETED
		job.Status = "COMPLETED"
		jobStore[job.ID] = job

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

		jobStore[newID] = newJob

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

	job, exists := jobStore[id]
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func main() {
	// --- START THE WORKERS ---
	// We spin up 3 workers to process jobs concurrently.
	for i := 1; i <= 3; i++ {
		// The "go" keyword starts this function in a new goroutine!
		go worker(i)
	}

	// Register HTTP handlers
	http.HandleFunc("/jobs", jobsHandler)
	http.HandleFunc("/jobs/", jobStatusHandler)

	fmt.Println("Starting server on port 8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
