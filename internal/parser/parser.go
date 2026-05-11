package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Target represents a Makefile target with an optional description.
type Target struct {
	Name        string
	Description string
}

// Project represents a directory containing a Makefile and its parsed targets.
type Project struct {
	Name    string
	Dir     string
	Targets []Target
}

// ScanProjects walks root recursively looking for Makefiles and returns parsed projects.
func ScanProjects(root string) ([]Project, error) {
	var projects []Project

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == "Makefile" {
			dir := filepath.Dir(path)
			targets, parseErr := ParseTargets(path)
			if parseErr != nil {
				return nil // skip unparseable Makefiles
			}
			if len(targets) == 0 {
				return nil
			}
			name := filepath.Base(dir)
			if name == "." {
				name = root
			}
			projects = append(projects, Project{
				Name:    name,
				Dir:     dir,
				Targets: targets,
			})
		}
		return nil
	})

	return projects, err
}

// ParseTargets reads a Makefile and extracts targets with optional descriptions.
func ParseTargets(path string) ([]Target, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var targets []Target
	var lastComment string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Capture comment lines above targets
		if strings.HasPrefix(line, "##") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(line, "##"))
			continue
		}
		if strings.HasPrefix(line, "#") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}

		// Skip lines starting with whitespace (recipe lines)
		if line == "" || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") {
			lastComment = ""
			continue
		}

		// Match target lines: "targetname:" or "targetname: deps"
		if idx := strings.Index(line, ":"); idx > 0 {
			name := strings.TrimSpace(line[:idx])
			// Skip special variables and phony-like declarations
			if strings.ContainsAny(name, "=$()") || name == ".PHONY" || name == ".DEFAULT_GOAL" {
				lastComment = ""
				continue
			}
			// Skip targets that look like variable assignments or multi-word keys
			if strings.Contains(name, " ") {
				lastComment = ""
				continue
			}
			targets = append(targets, Target{
				Name:        name,
				Description: lastComment,
			})
			lastComment = ""
			continue
		}

		lastComment = ""
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}
