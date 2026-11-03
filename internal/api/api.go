// Package api provides HTTP handlers for the project viewer.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/gmmorgan96/handle/internal/progress"
	"github.com/gmmorgan96/handle/internal/project"
)

// ProjectResponse is the JSON response for GET /api/project. It aggregates
// every service with its progress, counts, and any load error.
//
// The response shape is:
//
//	{
//	  "name": "Example Project",
//	  "root": "/absolute/path/to/project",
//	  "progress": {"done": 1, "total": 2, "percent": 50},
//	  "services": [
//	    {
//	      "id": "api",
//	      "name": "API",
//	      "description": "Public-facing API",
//	      "path": "./api",
//	      "depends_on": ["db"],
//	      "progress": {"done": 1, "total": 2, "percent": 50},
//	      "counts": {"features": 1, "tasks": 2, "bugs": 0},
//	      "error": "parse .../feature-001.md: ..."
//	    }
//	  ]
//	}
//
// Services keep the order declared in .project/services.yaml. The service
// "error" field is omitted when the service loaded successfully, and a failed
// service reports zeroed progress and counts.
type ProjectResponse struct {
	Name     string           `json:"name"`
	Root     string           `json:"root"`
	Progress ProgressStats    `json:"progress"`
	Services []ServiceSummary `json:"services"`
}

// ServiceSummary represents one service in the project response.
type ServiceSummary struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Path        string        `json:"path"`
	DependsOn   []string      `json:"depends_on"`
	Progress    ProgressStats `json:"progress"`
	Counts      Counts        `json:"counts"`
	Error       string        `json:"error,omitempty"`
}

// ProgressStats represents completion stats at any level.
type ProgressStats struct {
	Done    int     `json:"done"`
	Total   int     `json:"total"`
	Percent float64 `json:"percent"`
}

// Complete reports whether the level has work and all of it is done.
// A level with no work is not complete.
func (s ProgressStats) Complete() bool {
	return s.Total > 0 && s.Done == s.Total
}

// Counts holds the number of planning items in a service.
type Counts struct {
	Features int `json:"features"`
	Tasks    int `json:"tasks"`
	Bugs     int `json:"bugs"`
}

// Handler handles HTTP requests for the project API.
type Handler struct {
	projectRoot string
}

// NewHandler creates a new API handler for the given project root.
func NewHandler(projectRoot string) *Handler {
	return &Handler{projectRoot: projectRoot}
}

// ProjectHandler returns the aggregated project as JSON.
// GET /api/project
//
// Responses:
//
//	200: successful response with ProjectResponse body
//	500: fatal manifest error (malformed or missing manifest)
func (h *Handler) ProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p, err := project.Load(h.projectRoot)
	if err != nil {
		// Fatal manifest error - return 500 with error message
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := buildProjectResponse(p)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// buildProjectResponse constructs the API response from a loaded project.
func buildProjectResponse(p *project.Project) ProjectResponse {
	services := make([]ServiceSummary, len(p.Services))
	for i, svc := range p.Services {
		services[i] = buildServiceSummary(svc)
	}

	return ProjectResponse{
		Name:     p.Name,
		Root:     p.Root,
		Progress: toProgressStats(progress.Project(p)),
		Services: services,
	}
}

// buildServiceSummary constructs a service summary from a project service.
func buildServiceSummary(svc project.Service) ServiceSummary {
	var progressStats ProgressStats
	var counts Counts
	var errStr string

	if svc.Result != nil {
		progressStats = toProgressStats(progress.Service(svc.Result))
		counts = Counts{
			Features: len(svc.Result.Features),
			Tasks:    len(svc.Result.Tasks),
			Bugs:     len(svc.Result.Bugs),
		}
	} else {
		progressStats = ProgressStats{}
		counts = Counts{}
	}

	if svc.Err != nil {
		errStr = svc.Err.Error()
	}

	dependsOn := make([]string, len(svc.DependsOn))
	for i, dep := range svc.DependsOn {
		dependsOn[i] = string(dep)
	}

	return ServiceSummary{
		ID:          svc.ID,
		Name:        svc.Name,
		Description: svc.Description,
		Path:        svc.Path,
		DependsOn:   dependsOn,
		Progress:    progressStats,
		Counts:      counts,
		Error:       errStr,
	}
}

// toProgressStats converts internal progress.Stats to API ProgressStats.
func toProgressStats(s progress.Stats) ProgressStats {
	return ProgressStats{
		Done:    s.Done,
		Total:   s.Total,
		Percent: s.Percent,
	}
}

// ServeHTTP implements http.Handler for the API.
// Routes: GET /api/project
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/project":
		h.ProjectHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}
