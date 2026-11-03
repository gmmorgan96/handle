package ui

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gmmorgan96/handle/internal/plan"
	"github.com/gmmorgan96/handle/internal/project"
	"github.com/gmmorgan96/handle/internal/workitem"
)

// crumb is one breadcrumb link. The last crumb of a page has no URL.
type crumb struct {
	Label string
	URL   string
}

// itemStatus is completion derived from one document's acceptance criteria.
type itemStatus struct {
	Key     string
	Label   string
	Done    int
	Total   int
	Percent float64
}

// checkView is one acceptance criterion ready for rendering.
type checkView struct {
	Text string
	Done bool
}

// itemLink is a compact link to a task, bug, or feature.
type itemLink struct {
	LocalID string
	Title   string
	URL     string
	Status  itemStatus
}

// featureCard is a feature with its nested tasks and bugs.
type featureCard struct {
	LocalID      string
	Title        string
	Description  string
	URL          string
	Status       itemStatus
	Acceptance   []checkView
	Requirements []string
	Dependencies []string
	Tasks        []itemLink
	Bugs         []itemLink
	Commit       *plan.Commit
	Summary      string
}

type servicePage struct {
	Title           string
	ProjectName     string
	Crumbs          []crumb
	Service         serviceView
	Features        []featureCard
	UnassignedTasks []itemLink
	UnassignedBugs  []itemLink
	HasItems        bool
}

type featurePage struct {
	Title       string
	ProjectName string
	Crumbs      []crumb
	ServiceName string
	Feature     featureCard
}

type taskDetail struct {
	LocalID      string
	Title        string
	Description  string
	Status       itemStatus
	Acceptance   []checkView
	Requirements []string
	Dependencies []string
	Feature      *itemLink
	Commit       *plan.Commit
	Summary      string
}

type taskPage struct {
	Title       string
	ProjectName string
	Crumbs      []crumb
	ServiceName string
	Task        taskDetail
}

type bugDetail struct {
	LocalID      string
	Title        string
	Description  string
	Status       itemStatus
	Acceptance   []checkView
	Requirements []string
	Steps        []string
	Expected     string
	Actual       string
	Introduced   *plan.Commit
	Dependencies []string
	Feature      *itemLink
	Commit       *plan.Commit
	Summary      string
}

type bugPage struct {
	Title       string
	ProjectName string
	Crumbs      []crumb
	ServiceName string
	Bug         bugDetail
}

func (h *Handler) serveServiceRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	segments := splitPath(r.URL.Path)
	if len(segments) < 2 || segments[0] != "services" || segments[1] == "" {
		http.NotFound(w, r)
		return
	}

	p, err := project.Load(h.projectRoot)
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	service, ok := findService(p, segments[1])
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch len(segments) {
	case 2:
		render(w, serviceTemplate, "layout", buildServicePage(p, service))
	case 4:
		localID := segments[3]
		switch segments[2] {
		case "features":
			feature := findFeature(service.Result, localID)
			if feature == nil {
				http.NotFound(w, r)
				return
			}
			render(w, featureTemplate, "layout", buildFeaturePage(p, service, feature))
		case "tasks":
			task := findTask(service.Result, localID)
			if task == nil {
				http.NotFound(w, r)
				return
			}
			render(w, taskTemplate, "layout", buildTaskPage(p, service, task))
		case "bugs":
			bug := findBug(service.Result, localID)
			if bug == nil {
				http.NotFound(w, r)
				return
			}
			render(w, bugTemplate, "layout", buildBugPage(p, service, bug))
		default:
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func findService(p *project.Project, id string) (project.Service, bool) {
	for _, service := range p.Services {
		if service.ID == id {
			return service, true
		}
	}
	return project.Service{}, false
}

func findFeature(result *plan.Result, localID string) *plan.FeatureDoc {
	if result == nil || localID == "" {
		return nil
	}
	for _, feature := range result.Features {
		if feature != nil && feature.ID.Local() == localID {
			return feature
		}
	}
	return nil
}

func findTask(result *plan.Result, localID string) *plan.TaskDoc {
	if result == nil || localID == "" {
		return nil
	}
	for _, task := range result.Tasks {
		if task != nil && task.ID.Local() == localID {
			return task
		}
	}
	return nil
}

func findBug(result *plan.Result, localID string) *plan.BugDoc {
	if result == nil || localID == "" {
		return nil
	}
	for _, bug := range result.Bugs {
		if bug != nil && bug.ID.Local() == localID {
			return bug
		}
	}
	return nil
}

func serviceURL(serviceID string) string {
	return "/services/" + url.PathEscape(serviceID)
}

func featureURL(serviceID string, featureID workitem.WorkItemID) string {
	return serviceURL(serviceID) + "/features/" + url.PathEscape(featureID.Local())
}

func taskURL(serviceID string, taskID workitem.WorkItemID) string {
	return serviceURL(serviceID) + "/tasks/" + url.PathEscape(taskID.Local())
}

func bugURL(serviceID string, bugID workitem.WorkItemID) string {
	return serviceURL(serviceID) + "/bugs/" + url.PathEscape(bugID.Local())
}

func buildServicePage(p *project.Project, service project.Service) servicePage {
	view := buildServiceView(service)
	page := servicePage{
		Title:       pageTitle(view.Name, p.Name),
		ProjectName: p.Name,
		Crumbs: []crumb{
			{Label: "Project overview", URL: "/"},
			{Label: view.Name},
		},
		Service: view,
	}
	if service.Result == nil {
		return page
	}

	groups := groupItems(service.Result)
	for _, feature := range service.Result.Features {
		if feature == nil {
			continue
		}
		page.Features = append(page.Features, buildFeatureCard(
			service.ID,
			feature,
			groups.tasks[feature.ID],
			groups.bugs[feature.ID],
		))
	}

	page.UnassignedTasks = buildTaskLinks(service.ID, groups.unassignedTasks)
	page.UnassignedBugs = buildBugLinks(service.ID, groups.unassignedBugs)
	page.HasItems = len(page.Features) > 0 || len(page.UnassignedTasks) > 0 || len(page.UnassignedBugs) > 0
	return page
}

func buildFeaturePage(p *project.Project, service project.Service, feature *plan.FeatureDoc) featurePage {
	groups := groupItems(service.Result)
	serviceName := displayName(service)
	return featurePage{
		Title:       pageTitle(feature.Title, serviceName),
		ProjectName: p.Name,
		Crumbs: []crumb{
			{Label: "Project overview", URL: "/"},
			{Label: serviceName, URL: serviceURL(service.ID)},
			{Label: feature.ID.Local()},
		},
		ServiceName: serviceName,
		Feature: buildFeatureCard(
			service.ID,
			feature,
			groups.tasks[feature.ID],
			groups.bugs[feature.ID],
		),
	}
}

func buildTaskPage(p *project.Project, service project.Service, task *plan.TaskDoc) taskPage {
	serviceName := displayName(service)
	detail := taskDetail{
		LocalID:      task.ID.Local(),
		Title:        task.Title,
		Description:  task.Description,
		Status:       statusFromChecks(task.Acceptance),
		Acceptance:   checksView(task.Acceptance),
		Requirements: task.Requirements,
		Dependencies: dependencyStrings(task.Dependencies),
		Commit:       task.Commit,
		Summary:      task.Summary,
	}

	crumbs := []crumb{
		{Label: "Project overview", URL: "/"},
		{Label: serviceName, URL: serviceURL(service.ID)},
	}
	if task.Feature != nil {
		if feature := findFeature(service.Result, task.Feature.ID.Local()); feature != nil {
			detail.Feature = featureLink(service.ID, feature)
			crumbs = append(crumbs, crumb{Label: feature.ID.Local(), URL: featureURL(service.ID, feature.ID)})
		}
	}
	crumbs = append(crumbs, crumb{Label: task.ID.Local()})

	return taskPage{
		Title:       pageTitle(task.Title, serviceName),
		ProjectName: p.Name,
		Crumbs:      crumbs,
		ServiceName: serviceName,
		Task:        detail,
	}
}

func buildBugPage(p *project.Project, service project.Service, bug *plan.BugDoc) bugPage {
	serviceName := displayName(service)
	detail := bugDetail{
		LocalID:      bug.ID.Local(),
		Title:        bug.Title,
		Description:  bug.Description,
		Status:       statusFromChecks(bug.Acceptance),
		Acceptance:   checksView(bug.Acceptance),
		Requirements: bug.Requirements,
		Steps:        bug.Steps,
		Expected:     bug.Expected,
		Actual:       bug.Actual,
		Introduced:   bug.Introduced,
		Dependencies: dependencyStrings(bug.Dependencies),
		Commit:       bug.Commit,
		Summary:      bug.Summary,
	}

	crumbs := []crumb{
		{Label: "Project overview", URL: "/"},
		{Label: serviceName, URL: serviceURL(service.ID)},
	}
	if bug.Feature != nil {
		if feature := findFeature(service.Result, bug.Feature.ID.Local()); feature != nil {
			detail.Feature = featureLink(service.ID, feature)
			crumbs = append(crumbs, crumb{Label: feature.ID.Local(), URL: featureURL(service.ID, feature.ID)})
		}
	}
	crumbs = append(crumbs, crumb{Label: bug.ID.Local()})

	return bugPage{
		Title:       pageTitle(bug.Title, serviceName),
		ProjectName: p.Name,
		Crumbs:      crumbs,
		ServiceName: serviceName,
		Bug:         detail,
	}
}

func buildFeatureCard(serviceID string, feature *plan.FeatureDoc, tasks []*plan.TaskDoc, bugs []*plan.BugDoc) featureCard {
	return featureCard{
		LocalID:      feature.ID.Local(),
		Title:        feature.Title,
		Description:  feature.Description,
		URL:          featureURL(serviceID, feature.ID),
		Status:       statusFromChecks(feature.Acceptance),
		Acceptance:   checksView(feature.Acceptance),
		Requirements: feature.Requirements,
		Dependencies: dependencyStrings(feature.Dependencies),
		Tasks:        buildTaskLinks(serviceID, tasks),
		Bugs:         buildBugLinks(serviceID, bugs),
		Commit:       feature.Commit,
		Summary:      feature.Summary,
	}
}

func featureLink(serviceID string, feature *plan.FeatureDoc) *itemLink {
	return &itemLink{
		LocalID: feature.ID.Local(),
		Title:   feature.Title,
		URL:     featureURL(serviceID, feature.ID),
		Status:  statusFromChecks(feature.Acceptance),
	}
}

func buildTaskLinks(serviceID string, tasks []*plan.TaskDoc) []itemLink {
	links := make([]itemLink, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		links = append(links, itemLink{
			LocalID: task.ID.Local(),
			Title:   task.Title,
			URL:     taskURL(serviceID, task.ID),
			Status:  statusFromChecks(task.Acceptance),
		})
	}
	return links
}

func buildBugLinks(serviceID string, bugs []*plan.BugDoc) []itemLink {
	links := make([]itemLink, 0, len(bugs))
	for _, bug := range bugs {
		if bug == nil {
			continue
		}
		links = append(links, itemLink{
			LocalID: bug.ID.Local(),
			Title:   bug.Title,
			URL:     bugURL(serviceID, bug.ID),
			Status:  statusFromChecks(bug.Acceptance),
		})
	}
	return links
}

func statusFromChecks(checks []plan.Check) itemStatus {
	status := itemStatus{Total: len(checks)}
	for _, check := range checks {
		if check.Done {
			status.Done++
		}
	}

	switch {
	case status.Total == 0:
		status.Key, status.Label = statusEmpty, "No criteria"
	case status.Done == status.Total:
		status.Key, status.Label, status.Percent = statusComplete, "Complete", 100
	case status.Done == 0:
		status.Key, status.Label = statusNotStarted, "Not started"
	default:
		status.Key, status.Label = statusActive, "In progress"
		status.Percent = float64(status.Done) / float64(status.Total) * 100
	}
	return status
}

func checksView(checks []plan.Check) []checkView {
	views := make([]checkView, len(checks))
	for i, check := range checks {
		views[i] = checkView{Text: check.Text, Done: check.Done}
	}
	return views
}

func dependencyStrings(dependencies []workitem.WorkItemID) []string {
	values := make([]string, len(dependencies))
	for i, dependency := range dependencies {
		values[i] = string(dependency)
	}
	return values
}

func pageTitle(title, context string) string {
	if context == "" {
		return title
	}
	return title + " · " + context
}

type featureGroups struct {
	tasks           map[workitem.WorkItemID][]*plan.TaskDoc
	bugs            map[workitem.WorkItemID][]*plan.BugDoc
	unassignedTasks []*plan.TaskDoc
	unassignedBugs  []*plan.BugDoc
}

// groupItems associates tasks and bugs with features using both the feature's
// task/bug links and each item's Related Feature. Anything unreferenced is
// returned separately so it still appears on the service page.
func groupItems(result *plan.Result) featureGroups {
	groups := featureGroups{
		tasks: make(map[workitem.WorkItemID][]*plan.TaskDoc),
		bugs:  make(map[workitem.WorkItemID][]*plan.BugDoc),
	}
	if result == nil {
		return groups
	}

	taskByID := make(map[workitem.WorkItemID]*plan.TaskDoc, len(result.Tasks))
	for _, task := range result.Tasks {
		if task != nil {
			taskByID[task.ID] = task
		}
	}
	bugByID := make(map[workitem.WorkItemID]*plan.BugDoc, len(result.Bugs))
	for _, bug := range result.Bugs {
		if bug != nil {
			bugByID[bug.ID] = bug
		}
	}
	featureByID := make(map[workitem.WorkItemID]*plan.FeatureDoc, len(result.Features))
	for _, feature := range result.Features {
		if feature != nil {
			featureByID[feature.ID] = feature
		}
	}

	assignedTasks := make(map[workitem.WorkItemID]bool, len(result.Tasks))
	assignedBugs := make(map[workitem.WorkItemID]bool, len(result.Bugs))

	for _, feature := range result.Features {
		if feature == nil {
			continue
		}
		for _, link := range feature.Tasks {
			task := taskByID[link.ID]
			if task == nil || assignedTasks[task.ID] {
				continue
			}
			groups.tasks[feature.ID] = append(groups.tasks[feature.ID], task)
			assignedTasks[task.ID] = true
		}
		for _, link := range feature.Bugs {
			bug := bugByID[link.ID]
			if bug == nil || assignedBugs[bug.ID] {
				continue
			}
			groups.bugs[feature.ID] = append(groups.bugs[feature.ID], bug)
			assignedBugs[bug.ID] = true
		}
	}

	for _, task := range result.Tasks {
		if task == nil || assignedTasks[task.ID] || task.Feature == nil {
			continue
		}
		feature := featureByID[task.Feature.ID]
		if feature == nil {
			continue
		}
		groups.tasks[feature.ID] = append(groups.tasks[feature.ID], task)
		assignedTasks[task.ID] = true
	}
	for _, bug := range result.Bugs {
		if bug == nil || assignedBugs[bug.ID] || bug.Feature == nil {
			continue
		}
		feature := featureByID[bug.Feature.ID]
		if feature == nil {
			continue
		}
		groups.bugs[feature.ID] = append(groups.bugs[feature.ID], bug)
		assignedBugs[bug.ID] = true
	}

	for _, task := range result.Tasks {
		if task != nil && !assignedTasks[task.ID] {
			groups.unassignedTasks = append(groups.unassignedTasks, task)
		}
	}
	for _, bug := range result.Bugs {
		if bug != nil && !assignedBugs[bug.ID] {
			groups.unassignedBugs = append(groups.unassignedBugs, bug)
		}
	}

	return groups
}
