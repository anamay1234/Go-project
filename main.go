package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// --- 1. Data Structures ---

// Job represents a single unit of work in our system.
// The json tags dictate how the struct fields will look when converted to JSON.
type Job struct {
	ID     string `json:"id"`
	Status string `json:"status"` // e.g., "PENDING", "PROCESSING", "COMPLETED"
}

// --- 2. In-Memory Storage ---

// jobStore acts as our database. It maps a string (Job ID) to a Job object.
var jobStore = make(map[string]Job)

// jobCounter helps us generate simple unique IDs (1, 2, 3...)
var jobCounter int

// --- 3. HTTP Handlers ---

// jobsHandler handles requests to the root "/jobs" endpoint (e.g., creating a job).
func jobsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// 1. Generate a new ID
		jobCounter++
		newID := fmt.Sprintf("%d", jobCounter)

		// 2. Create the Job struct
		newJob := Job{
			ID:     newID,
			Status: "PENDING", // Every new job starts as PENDING
		}

		// 3. Save it to our in-memory map
		jobStore[newID] = newJob

		// 4. Return the new job as JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newJob)
		return
	}

	// If someone tries to GET /jobs, tell them it's not allowed yet
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// jobStatusHandler handles GET requests to check a job's status (e.g., GET /jobs/1)
func jobStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// r.URL.Path will be something like "/jobs/1"
	// We trim "/jobs/" to extract just the ID part ("1")
	id := strings.TrimPrefix(r.URL.Path, "/jobs/")
	if id == "" {
		http.Error(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	// Look up the job in our map
	// `exists` is a boolean that is true if the key was found
	job, exists := jobStore[id]
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Send the found job back as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func main() {
	// Register the endpoints
	// We handle the exact "/jobs" path for POSTing new jobs
	http.HandleFunc("/jobs", jobsHandler)
	
	// We handle "/jobs/" (with trailing slash) for dynamic paths like GET /jobs/1
	http.HandleFunc("/jobs/", jobStatusHandler)

	fmt.Println("Starting server on port 8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
