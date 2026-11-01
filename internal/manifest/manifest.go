// Package manifest loads and validates the project manifest at
// .project/services.yaml.
package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ServiceDependency is the id of another service that a service depends on.
type ServiceDependency string

// Project is a decoded .project/services.yaml manifest.
type Project struct {
	// Name is the human-readable project name (the YAML `project` field).
	Name string `yaml:"project"`

	// Services are the registered service repositories.
	Services []Service `yaml:"services"`

	// Root is the project root: the directory containing .project/.
	Root string `yaml:"-"`
}

// Service is one service repository registered in the manifest.
type Service struct {
	ID          string              `yaml:"id"`
	Name        string              `yaml:"name"`
	Path        string              `yaml:"path"`
	Description string              `yaml:"description,omitempty"`
	DependsOn   []ServiceDependency `yaml:"depends_on,omitempty"`

	// Dir is the service repository root, resolved relative to Project.Root.
	Dir string `yaml:"-"`
}

// ValidationError is one manifest problem. Service is empty for problems that
// are not tied to a particular service.
type ValidationError struct {
	Service string
	Problem string
}

// Error implements error.
func (e ValidationError) Error() string {
	if e.Service == "" {
		return e.Problem
	}
	return fmt.Sprintf("service %q: %s", e.Service, e.Problem)
}

// ValidationErrors is the collection of problems found in a manifest.
type ValidationErrors []ValidationError

// Error implements error.
func (e ValidationErrors) Error() string {
	lines := make([]string, len(e))
	for i := range e {
		lines[i] = e[i].Error()
	}
	return strings.Join(lines, "\n")
}

// Unwrap exposes each problem to errors.Is and errors.As.
func (e ValidationErrors) Unwrap() []error {
	errs := make([]error, len(e))
	for i := range e {
		errs[i] = e[i]
	}
	return errs
}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Load reads and validates the manifest at manifestPath. Service paths are
// resolved relative to the project root, which is the parent of the manifest's
// .project directory.
//
// A malformed manifest returns a nil project. A well-formed manifest that
// fails validation returns the decoded project together with ValidationErrors,
// so callers can still inspect the services that did validate.
func Load(manifestPath string) (*Project, error) {
	absPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("resolve manifest path %q: %w", manifestPath, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", manifestPath, err)
	}

	project := new(Project)
	if err := yaml.Unmarshal(data, project); err != nil {
		return nil, fmt.Errorf("parse manifest %q: %w", manifestPath, err)
	}

	manifestDir := filepath.Dir(absPath)
	if filepath.Base(manifestDir) != ".project" {
		return nil, fmt.Errorf("manifest %q must live in a .project directory", manifestPath)
	}
	project.Root = filepath.Dir(manifestDir)

	if errs := project.validate(); len(errs) > 0 {
		return project, ValidationErrors(errs)
	}
	return project, nil
}

// LoadProject loads <projectRoot>/.project/services.yaml.
func LoadProject(projectRoot string) (*Project, error) {
	return Load(filepath.Join(projectRoot, ".project", "services.yaml"))
}

func (p *Project) validate() []ValidationError {
	var errs []ValidationError

	if strings.TrimSpace(p.Name) == "" {
		errs = append(errs, ValidationError{Problem: "project name must not be empty"})
	}
	if len(p.Services) == 0 {
		errs = append(errs, ValidationError{Problem: "manifest must declare at least one service"})
	}

	seen := make(map[string]int, len(p.Services))
	for i := range p.Services {
		service := &p.Services[i]
		label := serviceLabel(service, i)

		switch {
		case service.ID == "":
			errs = append(errs, ValidationError{Service: label, Problem: "id must not be empty"})
		case !idPattern.MatchString(service.ID):
			errs = append(errs, ValidationError{
				Service: label,
				Problem: `id must match ^[a-z0-9]+(-[a-z0-9]+)*$`,
			})
		}

		if service.ID != "" {
			if first, ok := seen[service.ID]; ok {
				errs = append(errs, ValidationError{
					Service: service.ID,
					Problem: fmt.Sprintf("duplicate id %q (first declared at service #%d)", service.ID, first+1),
				})
			} else {
				seen[service.ID] = i
			}
		}

		if strings.TrimSpace(service.Name) == "" {
			errs = append(errs, ValidationError{Service: label, Problem: "name must not be empty"})
		}
		if strings.TrimSpace(service.Path) == "" {
			errs = append(errs, ValidationError{Service: label, Problem: "path must not be empty"})
		}
	}

	declared := make(map[string]bool, len(seen))
	for id := range seen {
		declared[id] = true
	}

	for i := range p.Services {
		service := &p.Services[i]
		label := serviceLabel(service, i)
		for _, dependency := range service.DependsOn {
			id := string(dependency)
			if id == "" {
				errs = append(errs, ValidationError{Service: label, Problem: "dependency id must not be empty"})
				continue
			}
			if service.ID != "" && id == service.ID {
				errs = append(errs, ValidationError{Service: label, Problem: "must not depend on itself"})
				continue
			}
			if !declared[id] {
				errs = append(errs, ValidationError{
					Service: label,
					Problem: fmt.Sprintf("depends on unknown service %q", id),
				})
			}
		}
	}

	for _, cycle := range dependencyCycles(p.Services) {
		errs = append(errs, ValidationError{
			Service: cycle[0],
			Problem: "dependency cycle: " + strings.Join(cycle, " -> "),
		})
	}

	return append(errs, p.resolvePaths()...)
}

func (p *Project) resolvePaths() []ValidationError {
	var errs []ValidationError

	for i := range p.Services {
		service := &p.Services[i]
		label := serviceLabel(service, i)
		if strings.TrimSpace(service.Path) == "" {
			continue
		}
		if filepath.IsAbs(service.Path) {
			errs = append(errs, ValidationError{
				Service: label,
				Problem: fmt.Sprintf("path %q must be relative to the project root", service.Path),
			})
			continue
		}

		dir := filepath.Clean(filepath.Join(p.Root, service.Path))
		info, err := os.Stat(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				errs = append(errs, ValidationError{
					Service: label,
					Problem: fmt.Sprintf("path %q does not exist (resolved to %q)", service.Path, dir),
				})
			} else {
				errs = append(errs, ValidationError{
					Service: label,
					Problem: fmt.Sprintf("path %q cannot be read: %v", service.Path, err),
				})
			}
			continue
		}
		if !info.IsDir() {
			errs = append(errs, ValidationError{
				Service: label,
				Problem: fmt.Sprintf("path %q is not a directory", service.Path),
			})
			continue
		}

		service.Dir = dir

		planDir := filepath.Join(dir, ".handle", "plan")
		planInfo, err := os.Stat(planDir)
		if err != nil || !planInfo.IsDir() {
			errs = append(errs, ValidationError{
				Service: label,
				Problem: fmt.Sprintf("path %q is missing .handle/plan", service.Path),
			})
		}
	}

	return errs
}

func serviceLabel(service *Service, index int) string {
	if service.ID != "" {
		return service.ID
	}
	return fmt.Sprintf("#%d", index+1)
}

func dependencyCycles(services []Service) [][]string {
	graph := make(map[string][]string, len(services))
	for _, service := range services {
		if service.ID == "" {
			continue
		}
		for _, dependency := range service.DependsOn {
			id := string(dependency)
			if id == "" || id == service.ID {
				continue
			}
			graph[service.ID] = append(graph[service.ID], id)
		}
	}
	for id := range graph {
		sort.Strings(graph[id])
	}

	ids := make([]string, 0, len(graph))
	for id := range graph {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[string]int, len(graph))
	var stack []string
	var cycles [][]string
	seenCycles := make(map[string]bool)

	var visit func(string)
	visit = func(id string) {
		state[id] = visiting
		stack = append(stack, id)

		for _, next := range graph[id] {
			switch state[next] {
			case unvisited:
				visit(next)
			case visiting:
				start := indexOf(stack, next)
				if start < 0 {
					continue
				}
				cycle := canonicalCycle(stack[start:])
				key := strings.Join(cycle, "\x00")
				if !seenCycles[key] {
					seenCycles[key] = true
					cycles = append(cycles, cycle)
				}
			}
		}

		stack = stack[:len(stack)-1]
		state[id] = visited
	}

	for _, id := range ids {
		if state[id] == unvisited {
			visit(id)
		}
	}

	sort.Slice(cycles, func(i, j int) bool {
		return strings.Join(cycles[i], "\x00") < strings.Join(cycles[j], "\x00")
	})
	return cycles
}

func canonicalCycle(path []string) []string {
	nodes := append([]string(nil), path...)
	if len(nodes) == 0 {
		return nil
	}

	min := 0
	for i := range nodes {
		if nodes[i] < nodes[min] {
			min = i
		}
	}

	rotated := append([]string(nil), nodes[min:]...)
	rotated = append(rotated, nodes[:min]...)
	return append(rotated, rotated[0])
}

func indexOf(items []string, want string) int {
	for i, item := range items {
		if item == want {
			return i
		}
	}
	return -1
}
