// Package workitem defines work-item identifiers that stay unique across the
// services in a project.
package workitem

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// WorkItemID is a work-item identifier. A local id names an item within one
// service ("feature-001" or "T001"); a namespaced id adds the service id
// ("api-a/feature-001").
type WorkItemID string

var (
	serviceIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	localIDPattern   = regexp.MustCompile(`^([a-z]+-[0-9]+|[A-Z][0-9]+)$`)
)

// NewID returns the namespaced id for localID within serviceID. It does not
// validate its inputs; use Parse for untrusted strings.
func NewID(serviceID, localID string) WorkItemID {
	return WorkItemID(serviceID + "/" + localID)
}

// Parse validates raw and returns it as a WorkItemID. Both local ids
// ("feature-001", "T001") and namespaced ids ("api-a/feature-001") are
// accepted.
func Parse(raw string) (WorkItemID, error) {
	if raw == "" {
		return "", fmt.Errorf("work item id must not be empty")
	}

	parts := strings.Split(raw, "/")
	switch len(parts) {
	case 1:
		if !localIDPattern.MatchString(parts[0]) {
			return "", fmt.Errorf("invalid work item id %q: local id must look like feature-001 or T001", raw)
		}
	case 2:
		if !serviceIDPattern.MatchString(parts[0]) {
			return "", fmt.Errorf("invalid work item id %q: service id %q must match %s", raw, parts[0], serviceIDPattern)
		}
		if !localIDPattern.MatchString(parts[1]) {
			return "", fmt.Errorf("invalid work item id %q: local id %q must look like feature-001 or T001", raw, parts[1])
		}
	default:
		return "", fmt.Errorf("invalid work item id %q: want feature-001 or api-a/feature-001", raw)
	}

	return WorkItemID(raw), nil
}

// String returns the id in its canonical on-disk form.
func (id WorkItemID) String() string {
	return string(id)
}

// IsNamespaced reports whether id includes a service prefix.
func (id WorkItemID) IsNamespaced() bool {
	return strings.Contains(string(id), "/")
}

// IsLocal reports whether id is a non-empty id without a service prefix.
func (id WorkItemID) IsLocal() bool {
	return id != "" && !id.IsNamespaced()
}

// Service returns the service part of a namespaced id, or "" for a local id.
func (id WorkItemID) Service() string {
	service, _, ok := strings.Cut(string(id), "/")
	if !ok {
		return ""
	}
	return service
}

// Local returns the id without its service prefix.
func (id WorkItemID) Local() string {
	_, local, ok := strings.Cut(string(id), "/")
	if !ok {
		return string(id)
	}
	return local
}

// Namespaced returns id prefixed with serviceID when id is still local. An
// already namespaced id is returned unchanged.
func (id WorkItemID) Namespaced(serviceID string) WorkItemID {
	if id == "" || id.IsNamespaced() {
		return id
	}
	return NewID(serviceID, string(id))
}

// MarshalJSON encodes the id as a JSON string.
func (id WorkItemID) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(id))
}

// UnmarshalJSON decodes and validates a JSON string.
func (id *WorkItemID) UnmarshalJSON(data []byte) error {
	if id == nil {
		return fmt.Errorf("work item id: UnmarshalJSON on nil pointer")
	}

	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("work item id: %w", err)
	}

	parsed, err := Parse(raw)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
