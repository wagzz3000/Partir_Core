package omega

import (
	"encoding/json"
	"sync"
)

// RetryController implements the bounded retry strategy
// Same defect class twice = stop, escalate
type RetryController struct {
	mu            sync.RWMutex
	defectHistory map[string]map[string]int // ticketID -> defectClass -> count
}

// NewRetryController creates a new retry controller
func NewRetryController() *RetryController {
	return &RetryController{
		defectHistory: make(map[string]map[string]int),
	}
}

// ShouldRetry determines if a ticket should be retried
// Returns: shouldRetry, deltaFields (fields to fix)
func (r *RetryController) ShouldRetry(ticketID string, defects []Defect) (bool, []string) {
	if len(defects) == 0 {
		return false, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize history for this ticket
	if r.defectHistory[ticketID] == nil {
		r.defectHistory[ticketID] = make(map[string]int)
	}

	// Check each defect class
	shouldRetry := true
	deltaFields := []string{}
	for _, defect := range defects {
		// Increment count
		r.defectHistory[ticketID][defect.DefectClass]++
		count := r.defectHistory[ticketID][defect.DefectClass]

		// Same defect class twice = don't retry
		if count >= 2 {
			shouldRetry = false
		}

		// Collect offending fields for delta retry
		if len(defect.OffendingFields) > 0 {
			var fields map[string]interface{}
			if err := json.Unmarshal(defect.OffendingFields, &fields); err == nil {
				for field := range fields {
					deltaFields = append(deltaFields, field)
				}
			}
		}
	}

	return shouldRetry, deltaFields
}

// GetDefectHistory returns the defect history for a ticket
func (r *RetryController) GetDefectHistory(ticketID string) map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if history, ok := r.defectHistory[ticketID]; ok {
		// Return a copy
		result := make(map[string]int)
		for k, v := range history {
			result[k] = v
		}
		return result
	}
	return nil
}

// ClearTicket removes history for a ticket (on success)
func (r *RetryController) ClearTicket(ticketID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.defectHistory, ticketID)
}

// GetRepeatedDefects returns classes that have occurred more than once
func (r *RetryController) GetRepeatedDefects(ticketID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var repeated []string
	if history, ok := r.defectHistory[ticketID]; ok {
		for class, count := range history {
			if count >= 2 {
				repeated = append(repeated, class)
			}
		}
	}
	return repeated
}
