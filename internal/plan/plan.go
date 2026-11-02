// Package plan parses a single service's .handle/plan directory into a
// per-service result. Cross-service loading and error isolation stay in the
// aggregating layer so this package can stay focused on one repository.
package plan

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/gmmorgan96/handle/internal/workitem"
)

// Plan is the parsed contents of a .handle/plan/plan.md file.
type Plan struct {
	Title   string
	Summary string
	Phases  []Phase
}

// Phase is a single phase section from plan.md.
type Phase struct {
	ID           workitem.WorkItemID
	Name         string
	Purpose      string
	Features     []FeatureRef
	Dependencies []workitem.WorkItemID
}

// FeatureRef links a phase to a feature document.
type FeatureRef struct {
	ID    workitem.WorkItemID
	Title string
	Path  string
}

var (
	planTitleRE  = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	phaseRE      = regexp.MustCompile(`^###\s+(.+?)\s+\(\[([^\[\]]+)\]`)
	featureRE    = regexp.MustCompile(`^-\s+(?:\[[ xX]\]\s+)?\(\[([^\[\]]+)\]\(([^()]+)\)\)\s+(.+?)\s*$`)
	dependencyRE = regexp.MustCompile(`^-\s+(?:\[[ xX]\]\s+)?(\S+)\s*$`)
)

const (
	sectionNone = iota
	sectionFeatures
	sectionDependencies
)

// ParsePlan reads a plan.md document and returns its phases, features, and
// dependencies. An empty document is a valid, empty plan. Prose outside the
// known sections is ignored; malformed structural entries are reported with
// their line number.
func ParsePlan(r io.Reader) (*Plan, error) {
	scanner := bufio.NewScanner(r)
	plan := &Plan{}
	var (
		phase      *Phase
		section    = sectionNone
		summary    []string
		lineNumber int
		sawContent bool
	)

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.TrimSuffix(scanner.Text(), "\r"))
		if line == "" {
			continue
		}
		sawContent = true

		if strings.HasPrefix(line, "### ") {
			match := phaseRE.FindStringSubmatch(line)
			if match == nil {
				return nil, fmt.Errorf("line %d: malformed phase heading %q", lineNumber, line)
			}
			id, err := workitem.Parse(match[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: malformed phase id: %w", lineNumber, err)
			}
			plan.Phases = append(plan.Phases, Phase{
				ID:   id,
				Name: match[1],
			})
			phase = &plan.Phases[len(plan.Phases)-1]
			section = sectionNone
			continue
		}

		if strings.HasPrefix(line, "## ") {
			if plan.Title != "" {
				continue
			}
			match := planTitleRE.FindStringSubmatch(line)
			if match == nil {
				return nil, fmt.Errorf("line %d: malformed plan title %q", lineNumber, line)
			}
			plan.Title = match[1]
			continue
		}

		if plan.Title == "" {
			continue
		}

		if phase == nil {
			summary = append(summary, line)
			continue
		}

		switch {
		case strings.HasPrefix(line, "**Purpose**:"):
			phase.Purpose = strings.TrimSpace(strings.TrimPrefix(line, "**Purpose**:"))
			section = sectionNone
		case line == "**Features**":
			section = sectionFeatures
		case line == "**Dependencies**":
			section = sectionDependencies
		case strings.HasPrefix(line, "**"):
			section = sectionNone
		case section == sectionFeatures && strings.HasPrefix(line, "-"):
			ref, err := parseFeatureRef(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			phase.Features = append(phase.Features, ref)
		case section == sectionDependencies && strings.HasPrefix(line, "-"):
			dependency, err := parsePlanDependency(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			if dependency != "" {
				phase.Dependencies = append(phase.Dependencies, dependency)
			}
		case section == sectionFeatures || section == sectionDependencies:
			return nil, fmt.Errorf("line %d: unexpected entry %q in %s", lineNumber, line, sectionName(section))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read plan: %w", err)
	}
	if !sawContent {
		return plan, nil
	}
	if plan.Title == "" {
		return nil, errors.New("plan title not found")
	}

	plan.Summary = strings.Join(summary, " ")
	return plan, nil
}

func parseFeatureRef(line string) (FeatureRef, error) {
	match := featureRE.FindStringSubmatch(line)
	if match == nil {
		return FeatureRef{}, fmt.Errorf("malformed feature entry %q", line)
	}
	id, err := workitem.Parse(match[1])
	if err != nil {
		return FeatureRef{}, fmt.Errorf("malformed feature entry %q: %w", line, err)
	}
	return FeatureRef{ID: id, Path: match[2], Title: match[3]}, nil
}

func parsePlanDependency(line string) (workitem.WorkItemID, error) {
	match := dependencyRE.FindStringSubmatch(line)
	if match == nil {
		return "", fmt.Errorf("malformed dependency entry %q", line)
	}
	value := match[1]
	if strings.EqualFold(value, "None") {
		return "", nil
	}
	id, err := workitem.Parse(value)
	if err != nil {
		return "", fmt.Errorf("malformed dependency entry %q: %w", line, err)
	}
	return id, nil
}

func sectionName(section int) string {
	if section == sectionFeatures {
		return "Features"
	}
	return "Dependencies"
}
