package plan

import (
	"io"

	"github.com/gmmorgan96/handle/internal/workitem"
)

// FeatureDoc is the parsed contents of a .handle/plan/feature markdown file.
type FeatureDoc struct {
	ID           workitem.WorkItemID
	Title        string
	Description  string
	Requirements []string
	Tasks        []DocLink
	Bugs         []DocLink
	Dependencies []workitem.WorkItemID
	Acceptance   []Check
	Commit       *Commit
	Summary      string
}

// ParseFeature reads a feature document and extracts its metadata,
// requirements, task and bug links, dependencies, acceptance criteria,
// implementation commit, and summary.
func ParseFeature(r io.Reader) (*FeatureDoc, error) {
	doc, err := parseDocument(r)
	if err != nil {
		return nil, err
	}
	id, err := workitem.Parse(doc.ID)
	if err != nil {
		return nil, err
	}

	requirements, err := parseBullets(doc.section("Requirements"), "requirement")
	if err != nil {
		return nil, err
	}
	dependencies, err := parseDependencies(doc.section("Dependencies"))
	if err != nil {
		return nil, err
	}
	acceptance, err := parseChecks(doc.section("Acceptance Criteria"))
	if err != nil {
		return nil, err
	}
	commit, err := parseCommit(doc.section("Implementation"))
	if err != nil {
		return nil, err
	}
	tasks, err := parseDocLinks(doc.section("Tasks"), "task")
	if err != nil {
		return nil, err
	}
	bugs, err := parseDocLinks(doc.section("Bugs"), "bug")
	if err != nil {
		return nil, err
	}

	return &FeatureDoc{
		ID:           id,
		Title:        doc.Title,
		Description:  doc.description(),
		Requirements: requirements,
		Tasks:        tasks,
		Bugs:         bugs,
		Dependencies: dependencies,
		Acceptance:   acceptance,
		Commit:       commit,
		Summary:      doc.summary(),
	}, nil
}
