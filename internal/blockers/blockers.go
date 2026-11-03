// Package blockers resolves work-item dependency references across every
// service in a loaded project.
//
// Dependencies are declared in each planning document's "## Dependencies"
// section. plan.Load namespaces local references with the owning service and
// leaves cross-service references such as "api-b/T004" unchanged, so resolving
// a dependency is a lookup in a project-wide index of work items.
package blockers

import (
	"github.com/gmmorgan96/handle/internal/project"
	"github.com/gmmorgan96/handle/internal/workitem"
)

// Kind identifies the planning document a resolved item came from.
type Kind string

const (
	KindPhase   Kind = "phase"
	KindFeature Kind = "feature"
	KindTask    Kind = "task"
	KindBug     Kind = "bug"
)

// Item is one work item in the project-wide dependency graph. Dependencies are
// the item's blockers (declared under "## Dependencies"); Dependents are the
// items that this item blocks because they depend on it.
type Item struct {
	ID           workitem.WorkItemID
	ServiceID    string
	Kind         Kind
	Title        string
	Dependencies []Reference
	Dependents   []*Item
}

// Reference is one dependency declared by an Item. Item is nil and Resolved is
// false when the referenced work item does not exist in any loaded service.
type Reference struct {
	ID           workitem.WorkItemID
	Item         *Item
	Resolved     bool
	CrossService bool
}

// Unresolved is a dependency reference that could not be resolved. It is kept
// separately so callers can surface it instead of silently dropping it.
type Unresolved struct {
	Source       *Item
	Target       workitem.WorkItemID
	CrossService bool
}

// Resolution is the dependency graph for a project. Items keeps the order in
// which services and their documents were loaded: phases, then features,
// tasks, and bugs within each service.
type Resolution struct {
	Items      []*Item
	Unresolved []Unresolved

	byID    map[workitem.WorkItemID]*Item
	pending map[*Item][]workitem.WorkItemID
}

// Resolve builds the dependency graph for every loaded service in p. Services
// that failed to load are skipped; references to their work items remain
// unresolved. Resolve(nil) returns an empty resolution.
func Resolve(p *project.Project) *Resolution {
	resolution := &Resolution{
		Items:   []*Item{},
		byID:    make(map[workitem.WorkItemID]*Item),
		pending: make(map[*Item][]workitem.WorkItemID),
	}
	if p == nil {
		return resolution
	}

	for _, service := range p.Services {
		if service.Result == nil {
			continue
		}
		resolution.addService(service)
	}
	resolution.resolveReferences()
	return resolution
}

// Lookup returns the resolved item with id, or nil when no loaded service
// contains it.
func (r *Resolution) Lookup(id workitem.WorkItemID) *Item {
	if r == nil {
		return nil
	}
	return r.byID[id]
}

func (r *Resolution) addService(service project.Service) {
	result := service.Result
	if result.Plan != nil {
		for i := range result.Plan.Phases {
			phase := &result.Plan.Phases[i]
			r.addItem(service.ID, phase.ID, KindPhase, phase.Name, phase.Dependencies)
		}
	}
	for _, feature := range result.Features {
		if feature != nil {
			r.addItem(service.ID, feature.ID, KindFeature, feature.Title, feature.Dependencies)
		}
	}
	for _, task := range result.Tasks {
		if task != nil {
			r.addItem(service.ID, task.ID, KindTask, task.Title, task.Dependencies)
		}
	}
	for _, bug := range result.Bugs {
		if bug != nil {
			r.addItem(service.ID, bug.ID, KindBug, bug.Title, bug.Dependencies)
		}
	}
}

func (r *Resolution) addItem(serviceID string, id workitem.WorkItemID, kind Kind, title string, dependencies []workitem.WorkItemID) {
	if id == "" {
		return
	}

	item := &Item{
		ID:        id,
		ServiceID: serviceID,
		Kind:      kind,
		Title:     title,
	}
	r.Items = append(r.Items, item)
	// Keep the first item for an id so resolution stays deterministic if a
	// malformed service happens to contain duplicate ids.
	if _, exists := r.byID[id]; !exists {
		r.byID[id] = item
	}
	if len(dependencies) > 0 {
		r.pending[item] = append([]workitem.WorkItemID(nil), dependencies...)
	}
}

func (r *Resolution) resolveReferences() {
	for _, item := range r.Items {
		for _, target := range r.pending[item] {
			reference := r.reference(item, target)
			item.Dependencies = append(item.Dependencies, reference)

			if reference.Resolved {
				reference.Item.Dependents = append(reference.Item.Dependents, item)
				continue
			}
			r.Unresolved = append(r.Unresolved, Unresolved{
				Source:       item,
				Target:       target,
				CrossService: reference.CrossService,
			})
		}
	}
}

func (r *Resolution) reference(source *Item, target workitem.WorkItemID) Reference {
	return Reference{
		ID:           target,
		Item:         r.byID[target],
		Resolved:     r.byID[target] != nil,
		CrossService: isCrossService(source, target),
	}
}

func isCrossService(source *Item, target workitem.WorkItemID) bool {
	service := target.Service()
	return service != "" && service != source.ServiceID
}
