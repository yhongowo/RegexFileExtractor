package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRoundTripAndReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings", "config.json")
	cfg, err := Load(path)
	if err != nil || !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("%+v %v", cfg, err)
	}
	cfg.Source = `C:\测试数据`
	cfg.Language = "en"
	cfg.SelectedRule = "logs"
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, cfg) {
		t.Fatalf("%+v %v", got, err)
	}
	cfg.Language = "zh"
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil || got.Language != "zh" {
		t.Fatalf("%+v %v", got, err)
	}
}
func TestInvalidConfigIsPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte("{ broken json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("ignored corrupt configuration")
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(data) {
		t.Fatal("load modified invalid configuration")
	}
	backup, err := Recover(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(backup)
	if err != nil || string(got) != string(data) {
		t.Fatal("backup lost original")
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
}
func TestValidationBeforeSaving(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Rules[0].Pattern = "[" }, func(c *Config) { c.SelectedRule = "missing" },
		func(c *Config) { c.Rules[1].Name = c.Rules[0].Name }, func(c *Config) { c.Rules[1].ID = c.Rules[0].ID },
		func(c *Config) { c.Layout = "unknown" }, func(c *Config) { c.Version = 2 },
	} {
		cfg = Default()
		mutate(&cfg)
		if err := Save(path, cfg); err == nil {
			t.Fatal("accepted invalid config")
		}
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid save changed disk")
	}
}
