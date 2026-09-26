package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"regexfileextractor/internal/config"
	"regexfileextractor/internal/core"
)

func seedLargeResults(c *Controller, n int) {
	c.files = make([]core.File, n)
	c.selected = make([]bool, n)
	for i := range c.files {
		c.files[i] = core.File{Path: fmt.Sprintf("source/%08d/X.csv", i), Name: "X.csv", Size: 1024}
		c.selected[i] = true
	}
	c.refreshPlan()
}

func TestLatestPlanWins(t *testing.T) {
	c := testController(t)
	seedLargeResults(c, 10000)
	if !c.planPending || !c.copy.Disabled() {
		t.Fatal("copy available before plan is ready")
	}
	c.selected[0] = false
	c.refreshPlan()
	c.cfg.Layout = core.Flat
	c.refreshPlan()
	if c.nextPlan == nil || c.nextPlan.layout != core.Flat {
		t.Fatal("latest edit was not coalesced")
	}
	// The old completion must neither enable copying nor publish old paths.
	events, _ := testDispatchers.Load(c)
	(<-events.(chan func()))()
	if !c.planPending || !c.copy.Disabled() || len(c.plan) != 0 {
		t.Fatal("stale plan was published")
	}
	awaitBackground(t, c)
	if len(c.plan) != 9999 || c.copy.Disabled() || c.outputs[c.files[0].Path] != "" {
		t.Fatal("latest selection was not applied")
	}
	if c.plan[0].Relative != "X_001.csv" {
		t.Fatalf("wrong layout: %s", c.plan[0].Relative)
	}
}

func TestInvalidationAndCloseDiscardBackgroundPlan(t *testing.T) {
	for _, closeWindow := range []bool{false, true} {
		t.Run(fmt.Sprint(closeWindow), func(t *testing.T) {
			c := testController(t)
			seedLargeResults(c, 10000)
			if closeWindow {
				c.closeWindow()
			} else {
				c.invalidate()
			}
			awaitBackground(t, c)
			if len(c.plan) != 0 {
				t.Fatal("background plan survived invalidation/close")
			}
		})
	}
}

func TestSaveQueueCoalescesAndSnapshots(t *testing.T) {
	c := testController(t)
	c.persist()
	c.cfg.Source = "superseded"
	c.persist()
	c.cfg.Source = "latest"
	c.persist()
	if len(c.saveQueue) != 1 {
		t.Fatalf("ordinary saves did not coalesce: %d", len(c.saveQueue))
	}
	// Changing the live slice must not change the already submitted snapshot.
	original := c.cfg.Rules[0].Name
	c.cfg.Rules[0].Name = "not submitted"
	awaitBackground(t, c)
	stored, err := config.Load(c.configPath)
	if err != nil || stored.Source != "latest" || stored.Rules[0].Name != original {
		t.Fatalf("incorrect saved snapshot: %+v, %v", stored, err)
	}
}

func TestCloseFlushesLatestSettings(t *testing.T) {
	c := testController(t)
	c.persist()
	c.cfg.Source = "latest before exit"
	c.close()
	if c.closed || !c.closing || !c.source.Disabled() {
		t.Fatal("close did not wait for save")
	}
	awaitBackground(t, c)
	stored, err := config.Load(c.configPath)
	if err != nil || stored.Source != "latest before exit" || !c.closed {
		t.Fatalf("exit did not flush: %+v %v", stored, err)
	}
}

func TestCloseSaveFailureKeepsWindowOpen(t *testing.T) {
	c := testController(t)
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	c.configPath = filepath.Join(blocked, "config.json")
	c.close()
	awaitBackground(t, c)
	if c.closed || c.closing || c.source.Disabled() {
		t.Fatal("failed save closed or locked window")
	}
	if c.Window.Canvas().Overlays().Top() == nil {
		t.Fatal("save error not presented")
	}
}

func TestAsyncScanAndCopy(t *testing.T) {
	c := testController(t)
	source := t.TempDir()
	c.source.SetText(source)
	if err := os.WriteFile(filepath.Join(source, "X01Y01.csv"), []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	c.startScan()
	if !c.busy {
		t.Fatal("scan did not start")
	}
	awaitBackground(t, c)
	if len(c.plan) != 1 || c.copy.Disabled() {
		t.Fatal("scan result did not become ready")
	}
	c.startCopy()
	if !c.busy || !c.activity.Visible() {
		t.Fatal("copy did not enter checking phase")
	}
	awaitBackground(t, c)
	data, err := os.ReadFile(filepath.Join(source, "extracted", "X01Y01.csv"))
	if err != nil || string(data) != "payload" {
		t.Fatalf("copy: %q %v", data, err)
	}
	if c.activity.Visible() || c.progress.Visible() || c.details == "" {
		t.Fatal("copy did not restore idle state")
	}
}

func TestSaveCallbacksRemainOrdered(t *testing.T) {
	c := testController(t)
	var completed []string
	for _, source := range []string{"first", "second", "third"} {
		cfg := c.cfg
		cfg.Source = source
		c.saveConfig(cfg, func(err error) {
			if err != nil {
				t.Fatal(err)
			}
			stored, err := config.Load(c.configPath)
			if err != nil || stored.Source != source {
				t.Fatalf("save callback out of order: %+v %v", stored, err)
			}
			completed = append(completed, source)
		})
	}
	awaitBackground(t, c)
	if len(completed) != 3 {
		t.Fatal("save callback was discarded")
	}
}
