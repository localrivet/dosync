/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"context"
	"fmt"
	"net"

	"dosync/internal/replica"
)

// TCPHealthChecker implements the HealthChecker interface using TCP socket connections.
type TCPHealthChecker struct {
	*BaseChecker
}

// NewTCPHealthChecker creates a new health checker that uses TCP socket connections.
func NewTCPHealthChecker(config HealthCheckConfig) (*TCPHealthChecker, error) {
	// Set the correct type if it's not already set
	if config.Type == "" {
		config.Type = TCPHealthCheck
	} else if config.Type != TCPHealthCheck {
		return nil, fmt.Errorf("invalid health check type for TCPHealthChecker: %s", config.Type)
	}

	// Validate and apply defaults
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid TCP configuration: %w", err)
	}

	// Create the base checker
	baseChecker, err := NewBaseChecker(TCPHealthCheck, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base checker: %w", err)
	}

	return &TCPHealthChecker{
		BaseChecker: baseChecker,
	}, nil
}

// Check performs a health check on the specified replica using a TCP socket connection.
func (t *TCPHealthChecker) Check(replica replica.Replica) (bool, error) {
	// Check if it's time to perform a health check
	if !t.ShouldCheck() {
		// Return the current status if it's not time to check again
		healthy, _, _ := t.GetStatus()
		return healthy, nil
	}

	result, err := t.CheckWithDetails(replica)
	return result.Healthy, err
}

// CheckWithDetails performs a TCP socket health check and returns detailed information.
func (t *TCPHealthChecker) CheckWithDetails(replica replica.Replica) (HealthCheckResult, error) {
	// TODO: Enhance replica.Replica to include IP address or network information
	// For now, we still use localhost as a placeholder until replica provides the actual IP.
	// We need a way to map the replica to its accessible IP address within the Docker network.
	hostIP := "localhost" // Placeholder - Needs actual IP from replica

	// Validate port
	if t.Config.Port <= 0 {
		message := fmt.Sprintf("Invalid port configured for TCP health check: %d", t.Config.Port)
		t.UpdateStatus(false, message)
		return t.CreateHealthCheckResult(), fmt.Errorf(message)
	}

	// Construct the address
	address := fmt.Sprintf("%s:%d", hostIP, t.Config.Port)

	// Create a dialer with timeout
	dialer := &net.Dialer{
		Timeout: t.Config.Timeout,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), t.Config.Timeout)
	defer cancel()

	// Try to establish a TCP connection
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		message := fmt.Sprintf("TCP connection failed to %s: %v", address, err)
		t.UpdateStatus(false, message)
		return t.CreateHealthCheckResult(), fmt.Errorf(message)
	}
	defer conn.Close()

	// Connection succeeded, service is healthy
	message := fmt.Sprintf("TCP connection successful to %s", address)
	t.UpdateStatus(true, message)

	// Return the health check result
	return t.CreateHealthCheckResult(), nil
}
