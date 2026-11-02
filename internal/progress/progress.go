// Package progress computes completion from acceptance-criteria checkboxes and
// rolls it up task -> feature -> service -> project. Every percentage is 0-100,
// and an empty set is 0% rather than a division by zero.
package progress

import (
	"github.com/gmmorgan96/handle/internal/plan"
	"github.com/gmmorgan96/handle/internal/project"
	"github.com/gmmorgan96/handle/internal/workitem"
)

// Stats is the completion state at one rollup level. Done is the number of
// checked acceptance criteria for a task, or the number of complete tasks for a
// feature, service, or project. Total is the corresponding denominator.
type Stats struct {
	Done    int
	Total   int
	Percent float64
}

// Complete reports whether the level has work and all of it is done. A level
// with no work is not complete.
func (s Stats) Complete() bool {
	return s.Total > 0 && s.Done == s.Total
}

// Task returns completion for one task's acceptance criteria. A nil task or a
// task without criteria is an incomplete zero-total task.
func Task(task *plan.TaskDoc) Stats {
	if task == nil {
		return Stats{}
	}

	done := 0
	for _, check := range task.Acceptance {
		if check.Done {
			done++
		}
	}
	return newStats(done, len(task.Acceptance))
}

// Feature returns completion for the tasks that belong to feature. A task
// belongs to a feature when the feature lists it under "## Tasks" or the task's
// "## Related Feature" points at the feature. A nil feature has no tasks.
func Feature(feature *plan.FeatureDoc, tasks []*plan.TaskDoc) Stats {
	if feature == nil {
		return Stats{}
	}

	listed := make(map[workitem.WorkItemID]struct{}, len(feature.Tasks))
	for _, link := range feature.Tasks {
		listed[link.ID] = struct{}{}
	}

	done, total := 0, 0
	for _, task := range tasks {
		if task == nil {
			continue
		}
		_, isListed := listed[task.ID]
		isRelated := task.Feature != nil && task.Feature.ID == feature.ID
		if !isListed && !isRelated {
			continue
		}

		total++
		if Task(task).Complete() {
			done++
		}
	}
	return newStats(done, total)
}

// Service returns completion across every task in a parsed service. A nil or
// empty result is an empty service.
func Service(result *plan.Result) Stats {
	if result == nil {
		return Stats{}
	}

	done, total := 0, 0
	for _, task := range result.Tasks {
		if task == nil {
			continue
		}
		total++
		if Task(task).Complete() {
			done++
		}
	}
	return newStats(done, total)
}

// Project returns completion across every loaded service. Services without a
// parsed result, such as those that failed to load, contribute no tasks.
func Project(p *project.Project) Stats {
	if p == nil {
		return Stats{}
	}

	done, total := 0, 0
	for _, service := range p.Services {
		stats := Service(service.Result)
		done += stats.Done
		total += stats.Total
	}
	return newStats(done, total)
}

func newStats(done, total int) Stats {
	if total <= 0 {
		return Stats{}
	}
	return Stats{
		Done:    done,
		Total:   total,
		Percent: float64(done) / float64(total) * 100,
	}
}
