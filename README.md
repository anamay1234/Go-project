# Concurrent Job Processing System

A robust, concurrent backend service built in Go that processes jobs asynchronously using goroutines, channels, and worker pools.

## Features

* **Asynchronous Execution:** Jobs are submitted via a REST API and processed in the background.
* **Worker Pool Architecture:** Utilizes Go channels and goroutines to safely distribute work across multiple concurrent workers.
* **Thread-Safe State Management:** Employs `sync.Mutex` to handle concurrent reads and writes to the in-memory job store, preventing race conditions.
* **Fault-Tolerant:** Includes API endpoints for checking job status, safely cancelling running jobs, and retrying failed jobs (using attempt tracking to prevent double-processing).

## Architecture

1. **Client** makes a `POST /jobs` request.
2. **API Handler** creates a Job object, saves it to a thread-safe in-memory store, and pushes it to a buffered `JobQueue` channel.
3. **Worker Pool**, consisting of multiple goroutines, continuously pulls from the channel.
4. **State Transitions:** Jobs move from `PENDING` -> `PROCESSING` -> `COMPLETED`.
5. **Race Condition Prevention:** A Mutex locks the data store during reads/writes, preventing data corruption during simultaneous map access.

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/jobs`  | Submit a new job to the queue |
| `GET`  | `/jobs/{id}` | Check the status of a specific job |
| `POST` | `/jobs/{id}/cancel` | Cancel a job if it's currently pending or processing |
| `POST` | `/jobs/{id}/retry` | Retry a cancelled or failed job |

## Running the Project

1. Clone the repository.
2. Start the server:
   ```bash
   go run main.go
   ```
3. The server will start on `localhost:8080`.

### Testing Concurrent Safety (Race Detector)
To prove the thread-safety of the shared job store, a test is included that aggressively blasts the store with hundreds of concurrent reads and writes.

Run the test with Go's built-in race detector:
```bash
go test -race
```

## Built With
* **Go (Golang)**
* `net/http` for REST APIs
* `sync.Mutex` & `sync.WaitGroup` for concurrency controls
