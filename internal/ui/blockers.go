package ui

import (
	"net/http"

	"github.com/gmmorgan96/handle/internal/blockers"
	"github.com/gmmorgan96/handle/internal/project"
)

// blockersPage is the template data for the global "what's blocking me" view.
type blockersPage struct {
	Title                  string
	ProjectName            string
	Crumbs                 []crumb
	Groups                 []serviceBlockers
	Unresolved             []unresolvedReferenceView
	BlockedItems           int
	CrossServiceReferences int
	UnresolvedReferences   int
	HasItems               bool
}

// serviceBlockers is one service's blocked and blocking work items.
type serviceBlockers struct {
	ID    string
	Name  string
	URL   string
	Items []blockerItemView
}

// blockerItemView is one work item together with the items that block it and
// the items it blocks.
type blockerItemView struct {
	LocalID         string
	DisplayID       string
	Title           string
	Kind            string
	KindLabel       string
	URL             string
	BlockedBy       []referenceView
	Blocks          []referenceView
	HasCrossService bool
}

// referenceView is one resolved or unresolved dependency edge.
type referenceView struct {
	ID           string
	LocalID      string
	ServiceID    string
	ServiceName  string
	Title        string
	Kind         string
	KindLabel    string
	URL          string
	Resolved     bool
	CrossService bool
}

// unresolvedReferenceView is one dependency that points at a missing work item.
type unresolvedReferenceView struct {
	SourceID          string
	SourceTitle       string
	SourceServiceName string
	SourceURL         string
	Target            string
	TargetService     string
	CrossService      bool
}

// serveBlockers renders the global view of blocked work items and their
// blockers across every loaded service.
func (h *Handler) serveBlockers(w http.ResponseWriter, r *http.Request) {
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

	render(w, blockersTemplate, "layout", buildBlockersPage(p))
}

func buildBlockersPage(p *project.Project) blockersPage {
	page := blockersPage{
		Title:       pageTitle("What's blocking me", projectName(p)),
		ProjectName: projectName(p),
		Crumbs: []crumb{
			{Label: "Project overview", URL: "/"},
			{Label: "What's blocking me"},
		},
	}
	if p == nil {
		return page
	}

	resolution := blockers.Resolve(p)
	names := make(map[string]string, len(p.Services))
	order := make([]string, 0, len(p.Services))
	groups := make(map[string]*serviceBlockers, len(p.Services))
	for _, service := range p.Services {
		name := displayName(service)
		names[service.ID] = name
		groups[service.ID] = &serviceBlockers{
			ID:   service.ID,
			Name: name,
			URL:  serviceURL(service.ID),
		}
		order = append(order, service.ID)
	}

	for _, item := range resolution.Items {
		if len(item.Dependencies) == 0 && len(item.Dependents) == 0 {
			continue
		}
		group := groups[item.ServiceID]
		if group == nil {
			continue
		}

		view := buildBlockerItemView(item, names)
		group.Items = append(group.Items, view)
		if len(item.Dependencies) > 0 {
			page.BlockedItems++
		}
		for _, reference := range item.Dependencies {
			if reference.CrossService {
				page.CrossServiceReferences++
			}
		}
	}

	for _, id := range order {
		if group := groups[id]; len(group.Items) > 0 {
			page.Groups = append(page.Groups, *group)
		}
	}

	for _, unresolved := range resolution.Unresolved {
		page.Unresolved = append(page.Unresolved, buildUnresolvedReferenceView(unresolved, names))
	}
	page.UnresolvedReferences = len(resolution.Unresolved)
	page.HasItems = len(page.Groups) > 0 || len(page.Unresolved) > 0
	return page
}

func buildBlockerItemView(item *blockers.Item, names map[string]string) blockerItemView {
	view := blockerItemView{
		LocalID:   item.ID.Local(),
		DisplayID: string(item.ID),
		Title:     item.Title,
		Kind:      string(item.Kind),
		KindLabel: kindLabel(item.Kind),
		URL:       blockerItemURL(item),
	}

	for _, reference := range item.Dependencies {
		view.BlockedBy = append(view.BlockedBy, buildReferenceView(reference, names))
		if reference.CrossService {
			view.HasCrossService = true
		}
	}
	for _, dependent := range item.Dependents {
		if dependent == nil {
			continue
		}
		reference := itemReferenceView(dependent, item.ServiceID, names)
		view.Blocks = append(view.Blocks, reference)
		if reference.CrossService {
			view.HasCrossService = true
		}
	}
	return view
}

func buildReferenceView(reference blockers.Reference, names map[string]string) referenceView {
	view := referenceView{
		ID:           string(reference.ID),
		LocalID:      reference.ID.Local(),
		ServiceID:    reference.ID.Service(),
		ServiceName:  serviceName(names, reference.ID.Service()),
		Resolved:     reference.Resolved,
		CrossService: reference.CrossService,
	}
	if reference.Item == nil {
		return view
	}

	view.LocalID = reference.Item.ID.Local()
	view.ServiceID = reference.Item.ServiceID
	view.ServiceName = serviceName(names, reference.Item.ServiceID)
	view.Title = reference.Item.Title
	view.Kind = string(reference.Item.Kind)
	view.KindLabel = kindLabel(reference.Item.Kind)
	view.URL = blockerItemURL(reference.Item)
	return view
}

func itemReferenceView(target *blockers.Item, ownerServiceID string, names map[string]string) referenceView {
	return referenceView{
		ID:           string(target.ID),
		LocalID:      target.ID.Local(),
		ServiceID:    target.ServiceID,
		ServiceName:  serviceName(names, target.ServiceID),
		Title:        target.Title,
		Kind:         string(target.Kind),
		KindLabel:    kindLabel(target.Kind),
		URL:          blockerItemURL(target),
		Resolved:     true,
		CrossService: target.ServiceID != ownerServiceID,
	}
}

func buildUnresolvedReferenceView(unresolved blockers.Unresolved, names map[string]string) unresolvedReferenceView {
	view := unresolvedReferenceView{
		Target:        string(unresolved.Target),
		TargetService: serviceName(names, unresolved.Target.Service()),
		CrossService:  unresolved.CrossService,
	}
	if unresolved.Source != nil {
		view.SourceID = string(unresolved.Source.ID)
		view.SourceTitle = unresolved.Source.Title
		view.SourceServiceName = serviceName(names, unresolved.Source.ServiceID)
		view.SourceURL = blockerItemURL(unresolved.Source)
	}
	return view
}

func blockerItemURL(item *blockers.Item) string {
	if item == nil {
		return ""
	}
	switch item.Kind {
	case blockers.KindFeature:
		return featureURL(item.ServiceID, item.ID)
	case blockers.KindTask:
		return taskURL(item.ServiceID, item.ID)
	case blockers.KindBug:
		return bugURL(item.ServiceID, item.ID)
	default:
		// Phases have no drill-down page.
		return ""
	}
}

func kindLabel(kind blockers.Kind) string {
	switch kind {
	case blockers.KindPhase:
		return "Phase"
	case blockers.KindFeature:
		return "Feature"
	case blockers.KindTask:
		return "Task"
	case blockers.KindBug:
		return "Bug"
	default:
		return string(kind)
	}
}

func serviceName(names map[string]string, serviceID string) string {
	if serviceID == "" {
		return ""
	}
	if name := names[serviceID]; name != "" {
		return name
	}
	return serviceID
}

func projectName(p *project.Project) string {
	if p == nil {
		return ""
	}
	return p.Name
}
