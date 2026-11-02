// Package project aggregates a validated project manifest and every service's
// parsed .handle plan into one model. Loading stops for an individual service
// when that service cannot be resolved or parsed; the remaining services still
// load.
package project

import (
	"errors"
	"fmt"

	"github.com/gmmorgan96/handle/internal/manifest"
	"github.com/gmmorgan96/handle/internal/plan"
)

// Project is the aggregated, multi-service model. Services keep the order in
// which the manifest declares them.
type Project struct {
	Name     string
	Root     string
	Services []Service
}

// Service combines a manifest service with its parsed planning tree. Result is
// nil when Err is non-nil.
type Service struct {
	ID          string
	Name        string
	Description string
	Path        string
	Dir         string
	DependsOn   []manifest.ServiceDependency
	Result      *plan.Result
	Err         error
}

// Load reads <projectRoot>/.project/services.yaml and parses each registered
// service's .handle/plan directory. A manifest that cannot be read or decoded
// is fatal. Once a manifest decodes, validation problems and parse failures
// are recorded per service and Load returns a nil error, so one bad service
// never prevents the others from loading.
func Load(projectRoot string) (*Project, error) {
	decoded, err := manifest.LoadProject(projectRoot)
	if decoded == nil {
		return nil, err
	}

	var validation manifest.ValidationErrors
	if err != nil && !errors.As(err, &validation) {
		return nil, err
	}

	aggregated := &Project{
		Name:     decoded.Name,
		Root:     decoded.Root,
		Services: make([]Service, 0, len(decoded.Services)),
	}

	for i, service := range decoded.Services {
		entry := Service{
			ID:          service.ID,
			Name:        service.Name,
			Description: service.Description,
			Path:        service.Path,
			Dir:         service.Dir,
			DependsOn:   append([]manifest.ServiceDependency(nil), service.DependsOn...),
		}

		if problems := serviceProblems(service, i, validation); len(problems) > 0 {
			entry.Err = joinProblems(problems)
		} else if service.Dir == "" {
			entry.Err = fmt.Errorf("service %q: directory not resolved", service.ID)
		} else {
			entry.Result, entry.Err = plan.Load(service.ID, service.Dir)
		}

		aggregated.Services = append(aggregated.Services, entry)
	}

	return aggregated, nil
}

// serviceProblems returns the manifest validation problems attributed to one
// service. manifest validation labels a service by id, or by its one-based
// position when the id is empty.
func serviceProblems(service manifest.Service, index int, validation manifest.ValidationErrors) []manifest.ValidationError {
	label := service.ID
	if label == "" {
		label = fmt.Sprintf("#%d", index+1)
	}

	var problems []manifest.ValidationError
	for _, problem := range validation {
		if problem.Service == label {
			problems = append(problems, problem)
		}
	}
	return problems
}

func joinProblems(problems []manifest.ValidationError) error {
	errs := make([]error, len(problems))
	for i := range problems {
		errs[i] = problems[i]
	}
	return errors.Join(errs...)
}
