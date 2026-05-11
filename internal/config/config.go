package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Group is a named collection of project directories (each containing a Makefile).
type Group struct {
	Name     string   `json:"name"`
	Projects []string `json:"projects"` // absolute paths to dirs with a Makefile
}

// Config is the persistent application configuration.
type Config struct {
	Groups  []Group `json:"groups"`
	cfgPath string
}

func cfgFilePath() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "mkm", "config.json"), nil
}

// Load reads config from disk. Returns an empty config on first run.
func Load() (*Config, error) {
	p, err := cfgFilePath()
	if err != nil {
		return &Config{}, nil
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Config{cfgPath: p}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	c.cfgPath = p
	return &c, nil
}

// Save writes config atomically to disk.
func (c *Config) Save() error {
	if c.cfgPath == "" {
		p, err := cfgFilePath()
		if err != nil {
			return err
		}
		c.cfgPath = p
	}
	if err := os.MkdirAll(filepath.Dir(c.cfgPath), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.cfgPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, c.cfgPath)
}

// AddGroup appends a new group (deduplicates by name).
func (c *Config) AddGroup(name string) {
	for _, g := range c.Groups {
		if g.Name == name {
			return
		}
	}
	c.Groups = append(c.Groups, Group{Name: name})
}

// DeleteGroup removes the group at index idx.
func (c *Config) DeleteGroup(idx int) {
	if idx < 0 || idx >= len(c.Groups) {
		return
	}
	c.Groups = append(c.Groups[:idx], c.Groups[idx+1:]...)
}

// AddProject appends an absolute project path to the group (deduplicates).
func (c *Config) AddProject(groupIdx int, absPath string) {
	if groupIdx < 0 || groupIdx >= len(c.Groups) {
		return
	}
	for _, p := range c.Groups[groupIdx].Projects {
		if p == absPath {
			return
		}
	}
	c.Groups[groupIdx].Projects = append(c.Groups[groupIdx].Projects, absPath)
}

// RemoveProject removes the project at projIdx from the group at groupIdx.
func (c *Config) RemoveProject(groupIdx, projIdx int) {
	if groupIdx < 0 || groupIdx >= len(c.Groups) {
		return
	}
	g := &c.Groups[groupIdx]
	if projIdx < 0 || projIdx >= len(g.Projects) {
		return
	}
	g.Projects = append(g.Projects[:projIdx], g.Projects[projIdx+1:]...)
}
