package plan

import (
	"io"

	"github.com/gmmorgan96/handle/internal/workitem"
)

// TaskDoc is the parsed contents of a .handle/plan/task markdown file.
type TaskDoc struct {
	ID           workitem.WorkItemID
	Title        string
	Description  string
	Requirements []string
	Dependencies []workitem.WorkItemID
	Acceptance   []Check
	Feature      *DocLink
	Commit       *Commit
	Summary      string
}

// ParseTask reads a task document and extracts its metadata, requirements,
// dependencies, acceptance criteria, parent feature link, implementation
// commit, and summary.
func ParseTask(r io.Reader) (*TaskDoc, error) {
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
	feature, err := parseFirstDocLink(doc.section("Related Feature"), "feature")
	if err != nil {
		return nil, err
	}
	commit, err := parseCommit(doc.section("Implementation"))
	if err != nil {
		return nil, err
	}

	return &TaskDoc{
		ID:           id,
		Title:        doc.Title,
		Description:  doc.description(),
		Requirements: requirements,
		Dependencies: dependencies,
		Acceptance:   acceptance,
		Feature:      feature,
		Commit:       commit,
		Summary:      doc.summary(),
	}, nil
}
