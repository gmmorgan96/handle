// Package ui renders the browser overview for an aggregated project.
//
// The overview is server-rendered from the same project and progress packages
// that back the JSON API, so the cards and the API always report the same
// values. Filters and sorting are plain query parameters, which keeps the
// viewer usable without JavaScript and makes deep links reload-safe.
package ui

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/gmmorgan96/handle/internal/progress"
	"github.com/gmmorgan96/handle/internal/project"
)

//go:embed templates/*.html static/*
var assets embed.FS

var (
	overviewTemplate = template.Must(template.ParseFS(assets, "templates/layout.html", "templates/overview.html"))
	errorTemplate    = template.Must(template.ParseFS(assets, "templates/layout.html", "templates/error.html"))
	staticHandler    = http.FileServer(http.FS(assets))
)

const (
	statusAll      = "all"
	statusActive   = "active"
	statusComplete = "complete"
	statusEmpty    = "empty"
	statusError    = "error"

	sortName   = "name"
	sortStatus = "status"
)

// Handler renders the project overview. It loads the project on every request,
// so a browser refresh reflects edits on disk.
type Handler struct {
	projectRoot string
}

// NewHandler creates an overview handler for the project rooted at
// projectRoot.
func NewHandler(projectRoot string) *Handler {
	return &Handler{projectRoot: projectRoot}
}

// ServeHTTP routes the overview, static assets, and unknown paths.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/":
		h.serveOverview(w, r)
	case strings.HasPrefix(r.URL.Path, "/static/"):
		staticHandler.ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) serveOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p, err := project.Load(h.projectRoot)
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	query := r.URL.Query()
	data := buildOverview(
		p,
		strings.TrimSpace(query.Get("q")),
		normalizeStatus(query.Get("status")),
		normalizeSort(query.Get("sort")),
	)
	render(w, overviewTemplate, "layout", data)
}

// overviewPage is the template data for the project overview.
type overviewPage struct {
	Title           string
	ProjectName     string
	Progress        progress.Stats
	Services        []serviceView
	TotalServices   int
	VisibleServices int
	Query           string
	StatusOptions   []option
	SortOptions     []option
	Filtered        bool
}

// serviceView is one service card.
type serviceView struct {
	ID          string
	Name        string
	Description string
	DependsOn   []string
	Progress    progress.Stats
	Features    int
	Tasks       int
	Bugs        int
	Error       string
	Status      string
	StatusLabel string
}

// errorPage is the template data for a fatal project-loading error.
type errorPage struct {
	Title       string
	ProjectName string
	Error       string
}

// option is one select option for a filter or sort control.
type option struct {
	Value    string
	Label    string
	Selected bool
}

func buildOverview(p *project.Project, search, status, sortBy string) overviewPage {
	all := make([]serviceView, 0, len(p.Services))
	for _, service := range p.Services {
		all = append(all, buildServiceView(service))
	}

	visible := make([]serviceView, 0, len(all))
	for _, service := range all {
		if matchesSearch(service, search) && matchesStatus(service, status) {
			visible = append(visible, service)
		}
	}
	sortServices(visible, sortBy)

	return overviewPage{
		Title:           projectTitle(p),
		ProjectName:     p.Name,
		Progress:        progress.Project(p),
		Services:        visible,
		TotalServices:   len(all),
		VisibleServices: len(visible),
		Query:           search,
		StatusOptions:   statusOptions(status),
		SortOptions:     sortOptions(sortBy),
		Filtered:        search != "" || status != statusAll,
	}
}

func buildServiceView(service project.Service) serviceView {
	view := serviceView{
		ID:          service.ID,
		Name:        displayName(service),
		Description: service.Description,
	}
	for _, dependency := range service.DependsOn {
		view.DependsOn = append(view.DependsOn, string(dependency))
	}

	if service.Result != nil {
		view.Progress = progress.Service(service.Result)
		view.Features = len(service.Result.Features)
		view.Tasks = len(service.Result.Tasks)
		view.Bugs = len(service.Result.Bugs)
	}
	if service.Err != nil {
		view.Error = service.Err.Error()
	}

	view.Status, view.StatusLabel = serviceStatus(view)
	return view
}

func displayName(service project.Service) string {
	switch {
	case strings.TrimSpace(service.Name) != "":
		return service.Name
	case strings.TrimSpace(service.ID) != "":
		return service.ID
	default:
		return "(unnamed service)"
	}
}

func serviceStatus(view serviceView) (string, string) {
	switch {
	case view.Error != "":
		return statusError, "Error"
	case view.Progress.Total == 0:
		return statusEmpty, "No work"
	case view.Progress.Complete():
		return statusComplete, "Complete"
	default:
		return statusActive, "In progress"
	}
}

func matchesSearch(view serviceView, search string) bool {
	if search == "" {
		return true
	}

	needle := strings.ToLower(search)
	return strings.Contains(strings.ToLower(view.ID), needle) ||
		strings.Contains(strings.ToLower(view.Name), needle)
}

func matchesStatus(view serviceView, status string) bool {
	return status == statusAll || view.Status == status
}

var statusOrder = map[string]int{
	statusError:    0,
	statusActive:   1,
	statusComplete: 2,
	statusEmpty:    3,
}

func sortServices(services []serviceView, sortBy string) {
	sort.SliceStable(services, func(i, j int) bool {
		if sortBy == sortStatus {
			left, right := statusOrder[services[i].Status], statusOrder[services[j].Status]
			if left != right {
				return left < right
			}
		}
		return lessByName(services[i], services[j])
	})
}

func lessByName(left, right serviceView) bool {
	leftName := strings.ToLower(left.Name)
	rightName := strings.ToLower(right.Name)
	if leftName != rightName {
		return leftName < rightName
	}
	return left.ID < right.ID
}

func normalizeStatus(value string) string {
	switch status := strings.ToLower(strings.TrimSpace(value)); status {
	case statusActive, statusComplete, statusEmpty, statusError:
		return status
	default:
		return statusAll
	}
}

func normalizeSort(value string) string {
	switch sortBy := strings.ToLower(strings.TrimSpace(value)); sortBy {
	case sortStatus:
		return sortStatus
	default:
		return sortName
	}
}

func statusOptions(selected string) []option {
	options := []struct {
		value string
		label string
	}{
		{statusAll, "All statuses"},
		{statusActive, "In progress"},
		{statusComplete, "Complete"},
		{statusEmpty, "No work"},
		{statusError, "Error"},
	}

	values := make([]option, len(options))
	for i, item := range options {
		values[i] = option{Value: item.value, Label: item.label, Selected: item.value == selected}
	}
	return values
}

func sortOptions(selected string) []option {
	options := []struct {
		value string
		label string
	}{
		{sortName, "Name"},
		{sortStatus, "Status"},
	}

	values := make([]option, len(options))
	for i, item := range options {
		values[i] = option{Value: item.value, Label: item.label, Selected: item.value == selected}
	}
	return values
}

func projectTitle(p *project.Project) string {
	if p != nil && strings.TrimSpace(p.Name) != "" {
		return p.Name
	}
	return "Project overview"
}

func render(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	var body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&body, name, data); err != nil {
		http.Error(w, fmt.Sprintf("render page: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = body.WriteTo(w)
}

func renderErrorPage(w http.ResponseWriter, err error) {
	var body bytes.Buffer
	data := errorPage{Title: "Could not load project", Error: err.Error()}
	if renderErr := errorTemplate.ExecuteTemplate(&body, "layout", data); renderErr != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = body.WriteTo(w)
}
