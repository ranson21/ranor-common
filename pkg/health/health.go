package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Status represents the health check response
type Status struct {
	Status    string            `json:"status"`
	Details   map[string]Detail `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Detail represents individual health check details
type Detail struct {
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthChecker interface for both services
type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
}

// NewHealthHandler returns a handler that both services can use
func NewHealthHandler(checkers []HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		status := &Status{
			Status:    "healthy",
			Details:   make(map[string]Detail),
			Timestamp: time.Now(),
		}

		for _, checker := range checkers {
			detail := Detail{
				Status:    "healthy",
				Timestamp: time.Now(),
			}

			if err := checker.Check(ctx); err != nil {
				status.Status = "unhealthy"
				detail.Status = "unhealthy"
				detail.Error = err.Error()
			}

			status.Details[checker.Name()] = detail
		}

		w.Header().Set("Content-Type", "application/json")
		if status.Status != "healthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		if err := json.NewEncoder(w).Encode(status); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to encode health check response",
			})
		}
	}
}

// Common checker implementations

// DBChecker implements database health checking
type DBChecker struct {
	name string
	ping func(context.Context) error
}

func NewDBChecker(name string, pingFn func(context.Context) error) HealthChecker {
	return &DBChecker{
		name: name,
		ping: pingFn,
	}
}

func (c *DBChecker) Check(ctx context.Context) error {
	return c.ping(ctx)
}

func (c *DBChecker) Name() string {
	return c.name
}

// ServiceChecker implements external service health checking
type ServiceChecker struct {
	name    string
	url     string
	timeout time.Duration
	client  *http.Client
}

func NewServiceChecker(name, url string, timeout time.Duration) HealthChecker {
	return &ServiceChecker{
		name:    name,
		url:     url,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *ServiceChecker) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service returned status: %d", resp.StatusCode)
	}

	return nil
}

func (c *ServiceChecker) Name() string {
	return c.name
}
