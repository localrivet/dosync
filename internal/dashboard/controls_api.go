package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"

	"dosync/internal/controls"
)

var deploymentController *controls.DeploymentController

// SetDeploymentController sets the deployment controller for the API handlers
func SetDeploymentController(controller *controls.DeploymentController) {
	deploymentController = controller
}

// pauseServiceHandler handles POST /api/controls/pause/{service}
func pauseServiceHandler(w http.ResponseWriter, r *http.Request) {
	if deploymentController == nil {
		http.Error(w, "Deployment controller not initialized", http.StatusInternalServerError)
		return
	}

	serviceName := r.Context().Value("service").(string)
	if serviceName == "" {
		http.Error(w, "Service name required", http.StatusBadRequest)
		return
	}

	deploymentController.PauseService(serviceName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Service %s paused", serviceName),
	})
}

// resumeServiceHandler handles POST /api/controls/resume/{service}
func resumeServiceHandler(w http.ResponseWriter, r *http.Request) {
	if deploymentController == nil {
		http.Error(w, "Deployment controller not initialized", http.StatusInternalServerError)
		return
	}

	serviceName := r.Context().Value("service").(string)
	if serviceName == "" {
		http.Error(w, "Service name required", http.StatusBadRequest)
		return
	}

	deploymentController.ResumeService(serviceName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Service %s resumed", serviceName),
	})
}

// approveDeploymentHandler handles POST /api/controls/approve/{service}
func approveDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	if deploymentController == nil {
		http.Error(w, "Deployment controller not initialized", http.StatusInternalServerError)
		return
	}

	serviceName := r.Context().Value("service").(string)
	if serviceName == "" {
		http.Error(w, "Service name required", http.StatusBadRequest)
		return
	}

	deploymentController.ApproveDeployment(serviceName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Deployment approved for service %s", serviceName),
	})
}

// statusHandler handles GET /api/controls/status
func controlsStatusHandler(w http.ResponseWriter, r *http.Request) {
	if deploymentController == nil {
		http.Error(w, "Deployment controller not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dry_run": deploymentController.IsDryRun(),
	})
}
