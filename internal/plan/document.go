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

// Check is a single acceptance criterion and whether it is checked.
type Check struct {
	Text string
	Done bool
}

// Commit is the implementation commit recorded in a planning document.
type Commit struct {
	ID      string
	Message string
}

// DocLink is a reference from one planning document to another.
type DocLink struct {
	ID   workitem.WorkItemID
	Path string
}

var (
	docTitleRE   = regexp.MustCompile(`^#\s+([A-Za-z]\d{3})\s+-\s+(.+?)\s*$`)
	sectionRE    = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	bulletRE     = regexp.MustCompile(`^[-*]\s+(.+?)\s*$`)
	checkRE      = regexp.MustCompile(`^[-*]\s+\[([ xX])\]\s+(.+?)\s*$`)
	docLinkRE    = regexp.MustCompile(`\[([^\[\]]+)\]\(([^()]+)\)`)
	commitHashRE = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

	checkboxPrefixRE = regexp.MustCompile(`^\[[ xX]\]\s+(.+)$`)
)

// rawDocument is the common structure shared by feature, task, and bug files.
// Section names are stored lowercased; lines keep their original order.
type rawDocument struct {
	ID       string
	Title    string
	Sections map[string][]string
}

// parseDocument reads a planning document with a "# ID - Title" heading and
// "## Section" bodies. Content outside known sections is ignored.
func parseDocument(r io.Reader) (*rawDocument, error) {
	scanner := bufio.NewScanner(r)
	doc := &rawDocument{Sections: map[string][]string{}}
	var (
		section    string
		lineNumber int
		titleFound bool
	)

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.TrimSuffix(scanner.Text(), "\r"))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "## ") {
			match := sectionRE.FindStringSubmatch(line)
			if match == nil {
				return nil, fmt.Errorf("line %d: malformed section heading %q", lineNumber, line)
			}
			section = strings.ToLower(match[1])
			continue
		}

		if strings.HasPrefix(line, "# ") {
			if titleFound {
				return nil, fmt.Errorf("line %d: duplicate document title %q", lineNumber, line)
			}
			match := docTitleRE.FindStringSubmatch(line)
			if match == nil {
				return nil, fmt.Errorf("line %d: malformed document title %q", lineNumber, line)
			}
			doc.ID = match[1]
			doc.Title = match[2]
			titleFound = true
			section = ""
			continue
		}

		if section == "" {
			continue
		}
		doc.Sections[section] = append(doc.Sections[section], line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read document: %w", err)
	}
	if !titleFound {
		return nil, errors.New("document title not found")
	}
	return doc, nil
}

func (d *rawDocument) section(name string) []string {
	return d.Sections[strings.ToLower(name)]
}

func (d *rawDocument) description() string {
	return strings.Join(d.section("Description"), " ")
}

func (d *rawDocument) summary() string {
	return strings.Join(d.section("Summary"), "\n")
}

func parseBullets(lines []string, name string) ([]string, error) {
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		match := bulletRE.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("malformed %s entry %q", name, line)
		}
		items = append(items, match[1])
	}
	return items, nil
}

func parseChecks(lines []string) ([]Check, error) {
	checks := make([]Check, 0, len(lines))
	for _, line := range lines {
		match := checkRE.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("malformed acceptance criterion %q", line)
		}
		checks = append(checks, Check{Text: match[2], Done: strings.EqualFold(match[1], "x")})
	}
	return checks, nil
}

func parseDocLinks(lines []string, name string) ([]DocLink, error) {
	links := make([]DocLink, 0, len(lines))
	for _, line := range lines {
		match := docLinkRE.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("malformed %s link %q", name, line)
		}
		id, err := workitem.Parse(strings.TrimSpace(match[1]))
		if err != nil {
			return nil, fmt.Errorf("malformed %s link %q: %w", name, line, err)
		}
		links = append(links, DocLink{ID: id, Path: match[2]})
	}
	return links, nil
}

// parseFirstDocLink parses a section that should reference at most one
// document, such as "Related Feature".
func parseFirstDocLink(lines []string, name string) (*DocLink, error) {
	links, err := parseDocLinks(lines, name)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}
	return &links[0], nil
}

// parseDependencies reads a "## Dependencies" section. "- None" means no
// dependencies; every other entry must be a local or namespaced work-item id.
func parseDependencies(lines []string) ([]workitem.WorkItemID, error) {
	deps := make([]workitem.WorkItemID, 0, len(lines))
	for _, line := range lines {
		match := bulletRE.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("malformed dependency entry %q", line)
		}

		value := strings.TrimSpace(match[1])
		if sub := checkboxPrefixRE.FindStringSubmatch(value); sub != nil {
			value = strings.TrimSpace(sub[1])
		}
		if strings.EqualFold(value, "None") {
			continue
		}

		id, err := workitem.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("malformed dependency entry %q: %w", line, err)
		}
		deps = append(deps, id)
	}
	return deps, nil
}

// parseCommit parses an "## Implementation" section. Both fields are
// placeholders until the work lands, so two placeholders mean no commit yet.
func parseCommit(lines []string) (*Commit, error) {
	return parseCommitFields(lines, false)
}

// parseIntroduced parses a bug's "## Potentially Introduced" section, where
// either field may be the literal "(unknown)".
func parseIntroduced(lines []string) (*Commit, error) {
	return parseCommitFields(lines, true)
}

func parseCommitFields(lines []string, allowUnknown bool) (*Commit, error) {
	var (
		commit         Commit
		seen           bool
		idPlaceholder  bool
		msgPlaceholder bool
	)

	for _, line := range lines {
		value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		lower := strings.ToLower(value)

		switch {
		case strings.HasPrefix(lower, "commit:"):
			seen = true
			hash := strings.TrimSpace(value[len("commit:"):])
			if isPlaceholder(hash) {
				idPlaceholder = true
				continue
			}
			if !commitHashRE.MatchString(hash) {
				return nil, fmt.Errorf("malformed commit hash %q", hash)
			}
			commit.ID = hash
		case strings.HasPrefix(lower, "message:"):
			seen = true
			message := strings.Trim(strings.TrimSpace(value[len("message:"):]), `"`)
			if isPlaceholder(message) {
				msgPlaceholder = true
				continue
			}
			if message == "" {
				return nil, errors.New("malformed commit message")
			}
			commit.Message = message
		}
	}

	if !seen {
		return nil, nil
	}
	if allowUnknown {
		if commit.ID == "" && commit.Message == "" {
			return nil, nil
		}
		return &commit, nil
	}
	if idPlaceholder || msgPlaceholder {
		if idPlaceholder && msgPlaceholder && commit.ID == "" && commit.Message == "" {
			return nil, nil
		}
		return nil, errors.New("incomplete commit metadata")
	}
	if commit.ID == "" || commit.Message == "" {
		return nil, errors.New("incomplete commit metadata")
	}
	return &commit, nil
}

func isPlaceholder(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "(pending)", "(unknown)", "(none)":
		return true
	}
	return false
}
