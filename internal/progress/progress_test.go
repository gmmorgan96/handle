package progress

import (
	"errors"
	"math"
	"testing"

	"github.com/gmmorgan96/handle/internal/plan"
	"github.com/gmmorgan96/handle/internal/project"
	"github.com/gmmorgan96/handle/internal/workitem"
)

func TestTask(t *testing.T) {
	tests := []struct {
		name     string
		task     *plan.TaskDoc
		want     Stats
		complete bool
	}{
		{name: "nil task", task: nil, want: Stats{}},
		{name: "no criteria", task: &plan.TaskDoc{}, want: Stats{}},
		{name: "none checked", task: taskWithChecks(0, 3), want: Stats{Total: 3}},
		{name: "partial", task: taskWithChecks(1, 3), want: Stats{Done: 1, Total: 3, Percent: 100.0 / 3}},
		{name: "mostly complete", task: taskWithChecks(2, 3), want: Stats{Done: 2, Total: 3, Percent: 200.0 / 3}},
		{name: "complete", task: taskWithChecks(3, 3), want: Stats{Done: 3, Total: 3, Percent: 100}, complete: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Task(tt.task)
			assertStats(t, got, tt.want)
			if got.Complete() != tt.complete {
				t.Errorf("Complete() = %t, want %t", got.Complete(), tt.complete)
			}
		})
	}
}

func TestFeatureUsesEitherLink(t *testing.T) {
	feature := &plan.FeatureDoc{
		ID: "api-a/F001",
		Tasks: []plan.DocLink{
			{ID: "api-a/T001"},
			{ID: "api-a/T002"},
		},
	}
	tasks := []*plan.TaskDoc{
		newTask("api-a/T001", 2, 2, "api-a/F001"), // listed and related, complete
		newTask("api-a/T002", 1, 2, ""),           // listed only, partial
		newTask("api-a/T003", 1, 1, "api-a/F002"), // related to another feature
		newTask("api-a/T004", 1, 1, ""),           // unrelated
		nil,
	}

	assertStats(t, Feature(feature, tasks), Stats{Done: 1, Total: 2, Percent: 50})

	relatedOnly := &plan.FeatureDoc{ID: "api-a/F002"}
	assertStats(t, Feature(relatedOnly, tasks), Stats{Done: 1, Total: 1, Percent: 100})

	assertStats(t, Feature(nil, tasks), Stats{})
}

func TestServiceRollup(t *testing.T) {
	result := &plan.Result{
		Tasks: []*plan.TaskDoc{
			newTask("api-a/T001", 2, 2, ""),
			newTask("api-a/T002", 1, 2, ""),
			newTask("api-a/T003", 0, 0, ""),
			nil,
		},
	}

	assertStats(t, Service(result), Stats{Done: 1, Total: 3, Percent: 100.0 / 3})
	assertStats(t, Service(nil), Stats{})
	assertStats(t, Service(&plan.Result{}), Stats{})
}

func TestProjectRollup(t *testing.T) {
	aggregated := &project.Project{
		Services: []project.Service{
			{
				ID: "api-a",
				Result: &plan.Result{
					Tasks: []*plan.TaskDoc{
						newTask("api-a/T001", 1, 1, ""),
						newTask("api-a/T002", 0, 2, ""),
					},
				},
			},
			{
				ID: "api-b",
				Result: &plan.Result{
					Tasks: []*plan.TaskDoc{
						newTask("api-b/T001", 1, 1, ""),
						newTask("api-b/T002", 1, 1, ""),
						newTask("api-b/T003", 0, 1, ""),
					},
				},
			},
			{
				ID:  "broken",
				Err: errors.New("service failed to load"),
			},
		},
	}

	assertStats(t, Project(aggregated), Stats{Done: 3, Total: 5, Percent: 60})
	assertStats(t, Project(nil), Stats{})
	assertStats(t, Project(&project.Project{}), Stats{})
}

func TestEmptyRollupsReturnZero(t *testing.T) {
	tests := []struct {
		name string
		got  Stats
	}{
		{name: "nil project", got: Project(nil)},
		{name: "empty project", got: Project(&project.Project{})},
		{
			name: "project with failed services",
			got: Project(&project.Project{
				Services: []project.Service{{}, {Result: nil}},
			}),
		},
		{name: "nil service", got: Service(nil)},
		{name: "empty service", got: Service(&plan.Result{})},
		{name: "nil feature", got: Feature(nil, nil)},
		{name: "empty feature", got: Feature(&plan.FeatureDoc{}, nil)},
		{name: "nil task", got: Task(nil)},
		{name: "empty task", got: Task(&plan.TaskDoc{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStats(t, tt.got, Stats{})
			if tt.got.Percent != 0 {
				t.Errorf("Percent = %v, want 0", tt.got.Percent)
			}
			if tt.got.Complete() {
				t.Error("Complete() = true, want false")
			}
		})
	}
}

func taskWithChecks(done, total int) *plan.TaskDoc {
	checks := make([]plan.Check, total)
	for i := range checks {
		checks[i] = plan.Check{Done: i < done}
	}
	return &plan.TaskDoc{Acceptance: checks}
}

func newTask(id workitem.WorkItemID, done, total int, feature workitem.WorkItemID) *plan.TaskDoc {
	task := taskWithChecks(done, total)
	task.ID = id
	if feature != "" {
		task.Feature = &plan.DocLink{ID: feature}
	}
	return task
}

func assertStats(t *testing.T, got, want Stats) {
	t.Helper()

	if got.Done != want.Done {
		t.Errorf("Done = %d, want %d", got.Done, want.Done)
	}
	if got.Total != want.Total {
		t.Errorf("Total = %d, want %d", got.Total, want.Total)
	}
	if math.Abs(got.Percent-want.Percent) > 1e-9 {
		t.Errorf("Percent = %v, want %v", got.Percent, want.Percent)
	}
}
