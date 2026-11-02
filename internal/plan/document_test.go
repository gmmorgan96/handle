package plan

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/gmmorgan96/handle/internal/workitem"
)

const sampleFeature = `# F021 - Sample feature

## Description
First line of the description
continues here.

## Requirements
- Parse feature files
- Report malformed input

## Tasks
- [x] [T001](../task/task-001.md) First task
- [ ] [T002](../task/task-002.md) Second task

## Dependencies
- F001
- api-b/feature-004

## Acceptance Criteria
- [x] Feature metadata extracted
- [ ] Related tasks linked

## Implementation
- Commit: 0123456789abcdef0123456789abcdef01234567
- Message: "F021: Sample feature"

## Summary
- Parsed the feature.
- Added tests.

## Bugs
- [B001](../bug/bug-001.md) Regression
`

const sampleTask = `# T001 - Unit tests for F002: Implement plan.md parser

## Description
Implement unit tests for the plan parser.

## Related Feature
- [F002](../feature/feature-002.md)

## Requirements
- Test phase headings
- Test dependencies

## Dependencies
- T002
- api-b/T004

## Acceptance Criteria
- [ ] All plan parser tests pass

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
`

const sampleBug = `# B001 - Parser drops empty sections

## Description
Empty sections are removed from the parsed output.

## Related Feature
- [F003](../feature/feature-003.md)

## Steps to Reproduce
1. Parse a document with an empty section.
2. Inspect the parsed sections.

## Expected/Actual
- **Expected**: the empty section is preserved
- **Actual**: the empty section is removed

## Potentially Introduced
- Commit: 0123456789abcdef0123456789abcdef01234567
- Message: "F003: Parse empty sections"

## Phase
- [P001](../plan.md)

## Requirements
- Preserve empty sections

## Dependencies
- T010

## Acceptance Criteria
- [x] Empty section preserved
- [ ] Regression test added

## Implementation
- Commit: 89abcdef0123456789abcdef0123456789abcdef
- Message: "B001: Preserve empty sections"

## Summary
- Fixed the parser.
`

func TestParseFeature(t *testing.T) {
	feature, err := ParseFeature(strings.NewReader(sampleFeature))
	if err != nil {
		t.Fatalf("ParseFeature returned error: %v", err)
	}

	if feature.ID != workitem.WorkItemID("F021") {
		t.Errorf("ID = %q, want F021", feature.ID)
	}
	if feature.Title != "Sample feature" {
		t.Errorf("Title = %q, want %q", feature.Title, "Sample feature")
	}
	if feature.Description != "First line of the description continues here." {
		t.Errorf("Description = %q", feature.Description)
	}
	if !reflect.DeepEqual(feature.Requirements, []string{"Parse feature files", "Report malformed input"}) {
		t.Errorf("Requirements = %v", feature.Requirements)
	}

	wantChecks := []Check{
		{Text: "Feature metadata extracted", Done: true},
		{Text: "Related tasks linked", Done: false},
	}
	if !reflect.DeepEqual(feature.Acceptance, wantChecks) {
		t.Errorf("Acceptance = %+v, want %+v", feature.Acceptance, wantChecks)
	}

	if feature.Commit == nil {
		t.Fatal("Commit = nil, want parsed commit")
	}
	if feature.Commit.ID != "0123456789abcdef0123456789abcdef01234567" {
		t.Errorf("Commit.ID = %q", feature.Commit.ID)
	}
	if feature.Commit.Message != "F021: Sample feature" {
		t.Errorf("Commit.Message = %q", feature.Commit.Message)
	}
	if feature.Summary != "- Parsed the feature.\n- Added tests." {
		t.Errorf("Summary = %q", feature.Summary)
	}

	wantTasks := []DocLink{
		{ID: "T001", Path: "../task/task-001.md"},
		{ID: "T002", Path: "../task/task-002.md"},
	}
	if !reflect.DeepEqual(feature.Tasks, wantTasks) {
		t.Errorf("Tasks = %+v, want %+v", feature.Tasks, wantTasks)
	}
	wantBugs := []DocLink{{ID: "B001", Path: "../bug/bug-001.md"}}
	if !reflect.DeepEqual(feature.Bugs, wantBugs) {
		t.Errorf("Bugs = %+v, want %+v", feature.Bugs, wantBugs)
	}
	wantDependencies := []workitem.WorkItemID{"F001", "api-b/feature-004"}
	if !reflect.DeepEqual(feature.Dependencies, wantDependencies) {
		t.Errorf("Dependencies = %v, want %v", feature.Dependencies, wantDependencies)
	}
}

func TestParseFeatureWithPendingImplementation(t *testing.T) {
	input := `# F001 - No implementation yet

## Acceptance Criteria
- [ ] Pending

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
`
	feature, err := ParseFeature(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseFeature returned error: %v", err)
	}
	if feature.Commit != nil {
		t.Errorf("Commit = %+v, want nil", feature.Commit)
	}
	if len(feature.Acceptance) != 1 || feature.Acceptance[0].Done {
		t.Errorf("Acceptance = %+v, want one unchecked item", feature.Acceptance)
	}
	if feature.Summary != "(pending)" {
		t.Errorf("Summary = %q, want (pending)", feature.Summary)
	}
}

func TestParseTask(t *testing.T) {
	task, err := ParseTask(strings.NewReader(sampleTask))
	if err != nil {
		t.Fatalf("ParseTask returned error: %v", err)
	}

	if task.ID != workitem.WorkItemID("T001") {
		t.Errorf("ID = %q, want T001", task.ID)
	}
	if task.Title != "Unit tests for F002: Implement plan.md parser" {
		t.Errorf("Title = %q", task.Title)
	}
	if task.Description != "Implement unit tests for the plan parser." {
		t.Errorf("Description = %q", task.Description)
	}
	if !reflect.DeepEqual(task.Requirements, []string{"Test phase headings", "Test dependencies"}) {
		t.Errorf("Requirements = %v", task.Requirements)
	}
	if !reflect.DeepEqual(task.Dependencies, []workitem.WorkItemID{"T002", "api-b/T004"}) {
		t.Errorf("Dependencies = %v", task.Dependencies)
	}
	if len(task.Acceptance) != 1 || task.Acceptance[0].Done {
		t.Errorf("Acceptance = %+v, want one unchecked item", task.Acceptance)
	}
	if task.Feature == nil {
		t.Fatal("Feature = nil, want related feature")
	}
	if *task.Feature != (DocLink{ID: "F002", Path: "../feature/feature-002.md"}) {
		t.Errorf("Feature = %+v", *task.Feature)
	}
	if task.Commit != nil {
		t.Errorf("Commit = %+v, want nil for pending implementation", task.Commit)
	}
	if task.Summary != "(pending)" {
		t.Errorf("Summary = %q, want (pending)", task.Summary)
	}
}

func TestParseTaskWithoutRelatedFeature(t *testing.T) {
	input := "# T002 - Standalone task\n\n## Acceptance Criteria\n- [x] Done\n"
	task, err := ParseTask(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTask returned error: %v", err)
	}
	if task.Feature != nil {
		t.Errorf("Feature = %+v, want nil", task.Feature)
	}
}

func TestParseBug(t *testing.T) {
	bug, err := ParseBug(strings.NewReader(sampleBug))
	if err != nil {
		t.Fatalf("ParseBug returned error: %v", err)
	}

	if bug.ID != workitem.WorkItemID("B001") {
		t.Errorf("ID = %q, want B001", bug.ID)
	}
	if bug.Title != "Parser drops empty sections" {
		t.Errorf("Title = %q", bug.Title)
	}
	if bug.Description != "Empty sections are removed from the parsed output." {
		t.Errorf("Description = %q", bug.Description)
	}
	if !reflect.DeepEqual(bug.Requirements, []string{"Preserve empty sections"}) {
		t.Errorf("Requirements = %v", bug.Requirements)
	}
	wantChecks := []Check{
		{Text: "Empty section preserved", Done: true},
		{Text: "Regression test added", Done: false},
	}
	if !reflect.DeepEqual(bug.Acceptance, wantChecks) {
		t.Errorf("Acceptance = %+v, want %+v", bug.Acceptance, wantChecks)
	}
	if !reflect.DeepEqual(bug.Steps, []string{
		"Parse a document with an empty section.",
		"Inspect the parsed sections.",
	}) {
		t.Errorf("Steps = %v", bug.Steps)
	}
	if bug.Expected != "the empty section is preserved" {
		t.Errorf("Expected = %q", bug.Expected)
	}
	if bug.Actual != "the empty section is removed" {
		t.Errorf("Actual = %q", bug.Actual)
	}
	if bug.Introduced == nil || bug.Introduced.ID != "0123456789abcdef0123456789abcdef01234567" {
		t.Errorf("Introduced = %+v", bug.Introduced)
	}
	if !reflect.DeepEqual(bug.Dependencies, []workitem.WorkItemID{"T010"}) {
		t.Errorf("Dependencies = %v", bug.Dependencies)
	}
	if bug.Phase != workitem.WorkItemID("P001") {
		t.Errorf("Phase = %q, want P001", bug.Phase)
	}
	if bug.Feature == nil || *bug.Feature != (DocLink{ID: "F003", Path: "../feature/feature-003.md"}) {
		t.Errorf("Feature = %+v", bug.Feature)
	}
	if bug.Commit == nil || bug.Commit.ID != "89abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf("Commit = %+v", bug.Commit)
	}
	if bug.Summary != "- Fixed the parser." {
		t.Errorf("Summary = %q", bug.Summary)
	}
}

func TestParseBugPlainPhase(t *testing.T) {
	input := "# B002 - Plain phase\n\n## Phase\n- P002\n"
	bug, err := ParseBug(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseBug returned error: %v", err)
	}
	if bug.Phase != workitem.WorkItemID("P002") {
		t.Errorf("Phase = %q, want P002", bug.Phase)
	}
}

func TestParseBugUnknownIntroduced(t *testing.T) {
	input := `# B003 - Unknown origin

## Potentially Introduced
- Commit: (unknown)
- Message: "(unknown)"
`
	bug, err := ParseBug(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseBug returned error: %v", err)
	}
	if bug.Introduced != nil {
		t.Errorf("Introduced = %+v, want nil", bug.Introduced)
	}
}

func TestParseDocumentErrors(t *testing.T) {
	featureParser := func(r io.Reader) error {
		_, err := ParseFeature(r)
		return err
	}
	bugParser := func(r io.Reader) error {
		_, err := ParseBug(r)
		return err
	}

	tests := []struct {
		name  string
		parse func(io.Reader) error
		input string
		want  string
	}{
		{name: "empty", parse: featureParser, input: "", want: "title"},
		{
			name:  "malformed title",
			parse: featureParser,
			input: "# Not an ID\n\n## Acceptance Criteria\n- [ ] Item\n",
			want:  "document title",
		},
		{
			name:  "duplicate title",
			parse: featureParser,
			input: "# F001 - One\n\n# F002 - Two\n",
			want:  "duplicate",
		},
		{
			name:  "malformed requirement",
			parse: featureParser,
			input: "# F001 - One\n\n## Requirements\nnot a bullet\n",
			want:  "requirement",
		},
		{
			name:  "malformed acceptance criterion",
			parse: featureParser,
			input: "# F001 - One\n\n## Acceptance Criteria\n- not a checkbox\n",
			want:  "acceptance",
		},
		{
			name:  "malformed task link",
			parse: featureParser,
			input: "# F001 - One\n\n## Tasks\n- not a link\n",
			want:  "task",
		},
		{
			name:  "malformed dependency",
			parse: featureParser,
			input: "# F001 - One\n\n## Dependencies\n- TBD\n",
			want:  "dependency",
		},
		{
			name:  "malformed commit hash",
			parse: featureParser,
			input: "# F001 - One\n\n## Implementation\n- Commit: nope\n- Message: \"F001: One\"\n",
			want:  "commit hash",
		},
		{
			name:  "incomplete commit",
			parse: featureParser,
			input: "# F001 - One\n\n## Implementation\n- Commit: 0123456789abcdef0123456789abcdef01234567\n",
			want:  "incomplete commit",
		},
		{
			name:  "half-pending commit",
			parse: featureParser,
			input: "# F001 - One\n\n## Implementation\n- Commit: 0123456789abcdef0123456789abcdef01234567\n- Message: (pending)\n",
			want:  "incomplete commit",
		},
		{
			name:  "malformed phase",
			parse: bugParser,
			input: "# B001 - One\n\n## Phase\nnot a bullet\n",
			want:  "phase",
		},
		{
			name:  "malformed reproduction step",
			parse: bugParser,
			input: "# B001 - One\n\n## Steps to Reproduce\nnot a step\n",
			want:  "reproduction",
		},
		{
			name:  "malformed expected actual",
			parse: bugParser,
			input: "# B001 - One\n\n## Expected/Actual\nnot expected or actual\n",
			want:  "expected/actual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("parser returned nil error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}
