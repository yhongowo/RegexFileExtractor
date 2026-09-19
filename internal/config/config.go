package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"regexfileextractor/internal/core"
)

type Config struct {
	Version      int           `json:"version"`
	Language     string        `json:"language"`
	Rules        []core.Rule   `json:"rules"`
	SelectedRule string        `json:"selected_rule"`
	Source       string        `json:"source"`
	Destination  string        `json:"destination"`
	Layout       core.Layout   `json:"layout"`
	Conflict     core.Conflict `json:"conflict"`
}

func Default() Config {
	return Config{Version: 1, Language: "zh", Layout: core.Auto, Conflict: core.Skip,
		Rules: []core.Rule{
			{ID: "spectrum", Name: "Spectrum_Data", Pattern: `(?:Spectrum_Data_)?X\d+Y\d+\.csv`},
			{ID: "datalog", Name: "Datalog", Pattern: `X\d+Y\d+\.csv`}},
		SelectedRule: "datalog"}
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "RegexFileExtractor", "config.json"), nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), err
	}
	if err := cfg.Validate(); err != nil {
		return Default(), err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version: %d", c.Version)
	}
	if c.Language != "zh" && c.Language != "en" {
		return fmt.Errorf("unsupported language: %s", c.Language)
	}
	if c.Layout != core.Flat && c.Layout != core.Auto && c.Layout != core.Group {
		return fmt.Errorf("invalid layout")
	}
	if c.Conflict != core.Skip && c.Conflict != core.Overwrite {
		return fmt.Errorf("invalid conflict policy")
	}
	ids := map[string]bool{}
	names := make([]string, 0, len(c.Rules))
	for _, rule := range c.Rules {
		if rule.ID == "" || ids[rule.ID] {
			return fmt.Errorf("empty or duplicate rule ID")
		}
		if _, err := rule.Compile(); err != nil {
			return fmt.Errorf("%s: %w", rule.Name, err)
		}
		name := strings.TrimSpace(rule.Name)
		for _, existing := range names {
			if strings.EqualFold(name, existing) {
				return fmt.Errorf("duplicate rule name: %s", rule.Name)
			}
		}
		ids[rule.ID] = true
		names = append(names, name)
	}
	if c.SelectedRule != "" && !ids[c.SelectedRule] {
		return fmt.Errorf("selected rule does not exist")
	}
	return nil
}

func Save(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// Recover preserves the exact invalid file before resetting it, only on explicit request.
func Recover(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	backup := path + ".backup-" + time.Now().Format("20060102-150405.000000000")
	if err := os.WriteFile(backup, data, 0600); err != nil {
		return "", err
	}
	if err := Save(path, Default()); err != nil {
		return backup, err
	}
	return backup, nil
}
