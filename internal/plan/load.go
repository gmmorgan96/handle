package plan

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gmmorgan96/handle/internal/workitem"
)

const (
	handleDir = ".handle"
	planDir   = "plan"
)

// Result is the parsed planning tree for one service. Item ids are namespaced
// with ServiceID so callers can combine results from several services without
// collisions.
type Result struct {
	ServiceID string
	Root      string
	PlanDir   string
	Plan      *Plan
	Features  []*FeatureDoc
	Tasks     []*TaskDoc
	Bugs      []*BugDoc
}

// Load parses <root>/.handle/plan for serviceID. It never panics: missing or
// malformed files are returned as errors that name the offending path.
//
// A completely empty .handle/plan directory is a valid empty service and
// produces an empty Result. If the directory has content, plan.md is required.
func Load(serviceID, root string) (*Result, error) {
	if serviceID == "" {
		return nil, errors.New("service id is empty")
	}
	if _, err := workitem.Parse(serviceID + "/T001"); err != nil {
		return nil, fmt.Errorf("invalid service id %q: %w", serviceID, err)
	}
	if root == "" {
		return nil, errors.New("service root is empty")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve service root: %w", err)
	}

	directory := filepath.Join(absoluteRoot, handleDir, planDir)
	info, err := os.Stat(directory)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("%s not found", directory)
	case err != nil:
		return nil, fmt.Errorf("locate %s: %w", directory, err)
	case !info.IsDir():
		return nil, fmt.Errorf("%s is not a directory", directory)
	}

	result := &Result{
		ServiceID: serviceID,
		Root:      absoluteRoot,
		PlanDir:   directory,
		Plan:      &Plan{},
		Features:  []*FeatureDoc{},
		Tasks:     []*TaskDoc{},
		Bugs:      []*BugDoc{},
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", directory, err)
	}
	if len(entries) == 0 {
		return result, nil
	}

	result.Plan, err = parseFile(filepath.Join(directory, "plan.md"), ParsePlan)
	if err != nil {
		return nil, err
	}
	if result.Features, err = loadDocuments(filepath.Join(directory, "feature"), ParseFeature); err != nil {
		return nil, err
	}
	if result.Tasks, err = loadDocuments(filepath.Join(directory, "task"), ParseTask); err != nil {
		return nil, err
	}
	if result.Bugs, err = loadDocuments(filepath.Join(directory, "bug"), ParseBug); err != nil {
		return nil, err
	}

	namespace(result, serviceID)
	return result, nil
}

// loadDocuments parses every markdown file directly inside dir. A missing
// directory yields no documents; other filesystem and parse errors are
// reported with the file path.
func loadDocuments[T any](dir string, parse func(io.Reader) (*T, error)) ([]*T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []*T{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	docs := make([]*T, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		doc, err := parseFile(filepath.Join(dir, entry.Name()), parse)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// parseFile opens a planning document and reports parse failures with its path.
func parseFile[T any](path string, parse func(io.Reader) (*T, error)) (*T, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	doc, err := parse(file)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return doc, nil
}

// namespace prefixes every local id in result with serviceID. Ids that are
// already namespaced (cross-service references) are left unchanged.
func namespace(result *Result, serviceID string) {
	if result.Plan != nil {
		for i := range result.Plan.Phases {
			phase := &result.Plan.Phases[i]
			phase.ID = phase.ID.Namespaced(serviceID)
			for j := range phase.Features {
				phase.Features[j].ID = phase.Features[j].ID.Namespaced(serviceID)
			}
			for j := range phase.Dependencies {
				phase.Dependencies[j] = phase.Dependencies[j].Namespaced(serviceID)
			}
		}
	}

	for _, feature := range result.Features {
		feature.ID = feature.ID.Namespaced(serviceID)
		for i := range feature.Tasks {
			feature.Tasks[i].ID = feature.Tasks[i].ID.Namespaced(serviceID)
		}
		for i := range feature.Bugs {
			feature.Bugs[i].ID = feature.Bugs[i].ID.Namespaced(serviceID)
		}
		for i := range feature.Dependencies {
			feature.Dependencies[i] = feature.Dependencies[i].Namespaced(serviceID)
		}
	}

	for _, task := range result.Tasks {
		task.ID = task.ID.Namespaced(serviceID)
		if task.Feature != nil {
			task.Feature.ID = task.Feature.ID.Namespaced(serviceID)
		}
		for i := range task.Dependencies {
			task.Dependencies[i] = task.Dependencies[i].Namespaced(serviceID)
		}
	}

	for _, bug := range result.Bugs {
		bug.ID = bug.ID.Namespaced(serviceID)
		if bug.Feature != nil {
			bug.Feature.ID = bug.Feature.ID.Namespaced(serviceID)
		}
		bug.Phase = bug.Phase.Namespaced(serviceID)
		for i := range bug.Dependencies {
			bug.Dependencies[i] = bug.Dependencies[i].Namespaced(serviceID)
		}
	}
}
