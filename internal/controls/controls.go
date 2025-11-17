package controls

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"dosync/internal/config"
)

// DeploymentController manages deployment controls (windows, approval, pause/resume)
type DeploymentController struct {
	config         *config.DeploymentControlConfig
	approvals      map[string]bool // service -> approved
	approvalsMutex sync.RWMutex
}

// NewDeploymentController creates a new deployment controller
func NewDeploymentController(cfg *config.DeploymentControlConfig) *DeploymentController {
	if cfg == nil {
		cfg = &config.DeploymentControlConfig{}
	}
	return &DeploymentController{
		config:    cfg,
		approvals: make(map[string]bool),
	}
}

// CanDeploy checks if a deployment is allowed for the given service
func (dc *DeploymentController) CanDeploy(ctx context.Context, serviceName string) (bool, string) {
	// Check if dry-run mode is enabled
	if dc.config.DryRun {
		return false, "Dry-run mode is enabled - no actual deployments will be performed"
	}

	// Check if service is paused
	if dc.isServicePaused(serviceName) {
		return false, fmt.Sprintf("Service %s is paused", serviceName)
	}

	// Check deployment windows
	if !dc.isWithinDeploymentWindow() {
		return false, "Outside of allowed deployment window"
	}

	// Check if approval is required
	if dc.config.RequireApproval {
		if !dc.isApproved(serviceName) {
			return false, fmt.Sprintf("Deployment approval required for service %s", serviceName)
		}
	}

	return true, ""
}

// isServicePaused checks if a service is in the paused list
func (dc *DeploymentController) isServicePaused(serviceName string) bool {
	for _, paused := range dc.config.PausedServices {
		if paused == serviceName {
			return true
		}
	}
	return false
}

// isWithinDeploymentWindow checks if the current time is within an allowed deployment window
func (dc *DeploymentController) isWithinDeploymentWindow() bool {
	// If no windows are configured, allow all times
	if len(dc.config.DeploymentWindows) == 0 {
		return true
	}

	now := time.Now()
	currentDay := now.Weekday().String()[:3] // Mon, Tue, Wed, etc.

	for _, window := range dc.config.DeploymentWindows {
		// Check if today is in the allowed days
		dayAllowed := false
		for _, day := range window.Days {
			if strings.EqualFold(day, currentDay) || strings.EqualFold(day, now.Weekday().String()) {
				dayAllowed = true
				break
			}
		}

		if !dayAllowed {
			continue
		}

		// Parse start and end times
		startTime, err1 := parseTimeOfDay(window.StartTime)
		endTime, err2 := parseTimeOfDay(window.EndTime)
		if err1 != nil || err2 != nil {
			continue
		}

		// Check if current time is within the window
		currentTime := now.Hour()*60 + now.Minute()
		if currentTime >= startTime && currentTime <= endTime {
			return true
		}
	}

	return false
}

// parseTimeOfDay parses a time string (HH:MM) and returns minutes since midnight
func parseTimeOfDay(timeStr string) (int, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format: %s (expected HH:MM)", timeStr)
	}

	var hour, minute int
	_, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil {
		return 0, err
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("invalid time values: %s", timeStr)
	}

	return hour*60 + minute, nil
}

// isApproved checks if a service deployment has been approved
func (dc *DeploymentController) isApproved(serviceName string) bool {
	dc.approvalsMutex.RLock()
	defer dc.approvalsMutex.RUnlock()
	return dc.approvals[serviceName]
}

// ApproveDeployment approves a deployment for a service
func (dc *DeploymentController) ApproveDeployment(serviceName string) {
	dc.approvalsMutex.Lock()
	defer dc.approvalsMutex.Unlock()
	dc.approvals[serviceName] = true
}

// RevokeApproval revokes approval for a service deployment
func (dc *DeploymentController) RevokeApproval(serviceName string) {
	dc.approvalsMutex.Lock()
	defer dc.approvalsMutex.Unlock()
	delete(dc.approvals, serviceName)
}

// PauseService pauses deployments for a service
func (dc *DeploymentController) PauseService(serviceName string) {
	for _, paused := range dc.config.PausedServices {
		if paused == serviceName {
			return // Already paused
		}
	}
	dc.config.PausedServices = append(dc.config.PausedServices, serviceName)
}

// ResumeService resumes deployments for a service
func (dc *DeploymentController) ResumeService(serviceName string) {
	newPaused := make([]string, 0)
	for _, paused := range dc.config.PausedServices {
		if paused != serviceName {
			newPaused = append(newPaused, paused)
		}
	}
	dc.config.PausedServices = newPaused
}

// IsPaused checks if a service is paused
func (dc *DeploymentController) IsPaused(serviceName string) bool {
	return dc.isServicePaused(serviceName)
}

// IsDryRun returns whether dry-run mode is enabled
func (dc *DeploymentController) IsDryRun() bool {
	return dc.config.DryRun
}
