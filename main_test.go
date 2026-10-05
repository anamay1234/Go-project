package main

import (
	"fmt"
	"sync"
	"testing"
)

// TestRaceCondition intentionally blasts our SharedJobStore with hundreds 
// of concurrent reads and writes to force a race condition.
func TestRaceCondition(t *testing.T) {
	// A WaitGroup helps us wait for a bunch of goroutines to finish
	// before the test ends.
	var wg sync.WaitGroup

	// We are going to spawn 500 separate goroutines all at once!
	for i := 0; i < 500; i++ {
		wg.Add(1)
		
		go func(workerID int) {
			defer wg.Done() // Tell the WaitGroup we are done when this function exits
			
			jobID := fmt.Sprintf("%d", workerID)

			// 1. WRITE to the shared map
			store.Set(Job{ID: jobID, Status: "PENDING"})

			// 2. READ from the shared map
			store.Get(jobID)

			// 3. WRITE to the shared map again
			store.Set(Job{ID: jobID, Status: "COMPLETED"})

		}(i)
	}

	// Wait for all 500 goroutines to finish
	wg.Wait()
}
