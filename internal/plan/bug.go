package plan

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/gmmorgan96/handle/internal/workitem"
)

// BugDoc is the parsed contents of a .handle/plan/bug markdown file.
type BugDoc struct {
	ID           workitem.WorkItemID
	Title        string
	Description  string
	Requirements []string
	Steps        []string
	Expected     string
	Actual       string
	Introduced   *Commit
	Dependencies []workitem.WorkItemID
	Acceptance   []Check
	Phase        workitem.WorkItemID
	Feature      *DocLink
	Commit       *Commit
	Summary      string
}

var (
	stepRE     = regexp.MustCompile(`^(?:\d+\.|[-*])\s+(.+?)\s*$`)
	expectedRE = regexp.MustCompile(`(?i)^[-*]\s+\*\*expected\*\*:\s*(.+?)\s*$`)
	actualRE   = regexp.MustCompile(`(?i)^[-*]\s+\*\*actual\*\*:\s*(.+?)\s*$`)
)

// ParseBug reads a bug document and extracts its metadata, requirements,
// reproduction steps, expected/actual behaviour, the commit that may have
// introduced it, dependencies, acceptance criteria, phase, related feature,
// implementation commit, and summary.
func ParseBug(r io.Reader) (*BugDoc, error) {
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
	steps, err := parseSteps(doc.section("Steps to Reproduce"))
	if err != nil {
		return nil, err
	}
	expected, actual, err := parseExpectedActual(doc.section("Expected/Actual"))
	if err != nil {
		return nil, err
	}
	introduced, err := parseIntroduced(doc.section("Potentially Introduced"))
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
	phase, err := parsePhase(doc.section("Phase"))
	if err != nil {
		return nil, err
	}
	commit, err := parseCommit(doc.section("Implementation"))
	if err != nil {
		return nil, err
	}

	return &BugDoc{
		ID:           id,
		Title:        doc.Title,
		Description:  doc.description(),
		Requirements: requirements,
		Steps:        steps,
		Expected:     expected,
		Actual:       actual,
		Introduced:   introduced,
		Dependencies: dependencies,
		Acceptance:   acceptance,
		Phase:        phase,
		Feature:      feature,
		Commit:       commit,
		Summary:      doc.summary(),
	}, nil
}

func parseSteps(lines []string) ([]string, error) {
	steps := make([]string, 0, len(lines))
	for _, line := range lines {
		match := stepRE.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("malformed reproduction step %q", line)
		}
		steps = append(steps, match[1])
	}
	return steps, nil
}

func parseExpectedActual(lines []string) (expected, actual string, err error) {
	for _, line := range lines {
		switch {
		case expectedRE.MatchString(line):
			expected = expectedRE.FindStringSubmatch(line)[1]
		case actualRE.MatchString(line):
			actual = actualRE.FindStringSubmatch(line)[1]
		default:
			return "", "", fmt.Errorf("malformed expected/actual entry %q", line)
		}
	}
	return expected, actual, nil
}

// parsePhase reads a bug's optional, singular phase reference. Bug files may
// refer to a phase either with a markdown link ("- [P001](../plan.md)") or as a
// plain bullet ("- P001"). Because BugDoc.Phase holds one id, more than one
// entry is an error rather than a value to silently ignore.
func parsePhase(lines []string) (workitem.WorkItemID, error) {
	switch len(lines) {
	case 0:
		return "", nil
	case 1:
		return parsePhaseEntry(lines[0])
	default:
		return "", fmt.Errorf("multiple phase entries: %q", strings.Join(lines, ", "))
	}
}

func parsePhaseEntry(line string) (workitem.WorkItemID, error) {
	var raw string
	if match := docLinkRE.FindStringSubmatch(line); match != nil {
		raw = strings.TrimSpace(match[1])
	} else {
		match := bulletRE.FindStringSubmatch(line)
		if match == nil {
			return "", fmt.Errorf("malformed phase entry %q", line)
		}
		raw = match[1]
	}

	id, err := workitem.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("malformed phase entry %q: %w", line, err)
	}
	return id, nil
}
