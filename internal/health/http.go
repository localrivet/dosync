/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"context"
	"fmt"
	"net/http"

	"dosync/internal/replica"
)

// HTTPHealthChecker implements the HealthChecker interface using HTTP requests.
type HTTPHealthChecker struct {
	*BaseChecker
	httpClient *http.Client
}

// NewHTTPHealthChecker creates a new health checker that uses HTTP requests.
func NewHTTPHealthChecker(config HealthCheckConfig) (*HTTPHealthChecker, error) {
	// Set the correct type if it's not already set
	if config.Type == "" {
		config.Type = HTTPHealthCheck
	} else if config.Type != HTTPHealthCheck {
		return nil, fmt.Errorf("invalid health check type for HTTPHealthChecker: %s", config.Type)
	}

	// Validate and apply defaults
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid HTTP configuration: %w", err)
	}

	// Create the base checker
	baseChecker, err := NewBaseChecker(HTTPHealthCheck, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base checker: %w", err)
	}

	// Create an HTTP client with the specified timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &HTTPHealthChecker{
		BaseChecker: baseChecker,
		httpClient:  httpClient,
	}, nil
}

// Check performs a health check on the specified replica using an HTTP request.
func (h *HTTPHealthChecker) Check(replica replica.Replica) (bool, error) {
	// Check if it's time to perform a health check
	if !h.ShouldCheck() {
		// Return the current status if it's not time to check again
		healthy, _, _ := h.GetStatus()
		return healthy, nil
	}

	result, err := h.CheckWithDetails(replica)
	return result.Healthy, err
}

// CheckWithDetails performs an HTTP health check and returns detailed information.
func (h *HTTPHealthChecker) CheckWithDetails(replica replica.Replica) (HealthCheckResult, error) {
	// TODO: Enhance replica.Replica to include IP address or network information
	// For now, we still use localhost as a placeholder until replica provides the actual IP.
	// We need a way to map the replica to its accessible IP address within the Docker network.
	containerIP := "localhost" // Placeholder - Needs actual IP from replica

	// Construct the URL using the placeholder IP, configured port, and endpoint
	targetURL := fmt.Sprintf("http://%s:%d%s", containerIP, h.Config.Port, h.Config.Endpoint)
	if h.Config.Port <= 0 {
		// If port is not configured, maybe default to 80?
		// Or return error? For now, using default port 80 as placeholder behaviour.
		targetURL = fmt.Sprintf("http://%s%s", containerIP, h.Config.Endpoint)
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.Config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		message := fmt.Sprintf("Failed to create HTTP request for %s: %v", targetURL, err)
		h.UpdateStatus(false, message)
		return h.CreateHealthCheckResult(), fmt.Errorf(message)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		message := fmt.Sprintf("HTTP request failed for %s: %v", targetURL, err)
		h.UpdateStatus(false, message)
		return h.CreateHealthCheckResult(), fmt.Errorf(message)
	}
	defer resp.Body.Close()

	// Check if the status code is in the 2xx range
	var healthy bool
	var message string
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		healthy = true
		message = fmt.Sprintf("HTTP check successful for %s: Status %d", targetURL, resp.StatusCode)
	} else {
		healthy = false
		message = fmt.Sprintf("HTTP check failed for %s: Status %d", targetURL, resp.StatusCode)
	}

	// Update the status
	h.UpdateStatus(healthy, message)

	// Return the health check result
	return h.CreateHealthCheckResult(), nil
}
