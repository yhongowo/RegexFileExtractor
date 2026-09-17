package ui

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"regexfileextractor/internal/config"
	"regexfileextractor/internal/core"
)

func testController(t *testing.T) *Controller {
	t.Helper()
	a := test.NewApp()
	a.Settings().SetTheme(Theme())
	c := New(a, config.Default(), filepath.Join(t.TempDir(), "config.json"), nil)
	c.Window.Resize(fyne.NewSize(1280, 820))
	c.Window.Show()
	t.Cleanup(func() { c.Window.SetCloseIntercept(nil); c.Window.Close(); a.Quit() })
	return c
}
func seedResults(c *Controller) {
	c.files = []core.File{
		{Path: `D:\TestData\Batch_A\X01Y01.csv`, Name: "X01Y01.csv", Size: 18243, Rule: "XY CSV"},
		{Path: `D:\TestData\Batch_B\X01Y01.csv`, Name: "X01Y01.csv", Size: 20481, Rule: "XY CSV"},
		{Path: `D:\TestData\Batch_B\X02Y02.csv`, Name: "X02Y02.csv", Size: 15920, Rule: "XY CSV"},
		{Path: `D:\TestData\Batch_C\X03Y01.csv`, Name: "X03Y01.csv", Size: 29041, Rule: "XY CSV"},
	}
	c.selected = []bool{true, true, true, true}
	c.refreshPlan()
}
func TestSelectionAndInvalidation(t *testing.T) {
	c := testController(t)
	if !c.copy.Disabled() {
		t.Fatal("copy enabled without scan results")
	}
	seedResults(c)
	if c.copy.Disabled() || len(c.plan) != 4 {
		t.Fatal("copy not available for selected files")
	}
	if c.outputs[c.files[0].Path] != filepath.Join("X01Y01", "X01Y01_001.csv") {
		t.Fatal("group output not previewed")
	}
	c.selected[1] = false
	c.refreshPlan()
	if c.outputs[c.files[0].Path] != "X01Y01.csv" {
		t.Fatal("auto did not recompute for selection")
	}
	if _, exists := c.outputs[c.files[1].Path]; exists {
		t.Fatal("unselected file still in plan")
	}
	test.Tap(c.scan) // Empty source fails validation; no background scan is started.
	if c.busy {
		t.Fatal("started scanning without a source")
	}
	c.selectAll(false)
	if !c.copy.Disabled() {
		t.Fatal("copy enabled with no selection")
	}
	c.selectAll(true)
	c.source.SetText("new source")
	if len(c.files) != 0 || !c.copy.Disabled() {
		t.Fatal("source change retained stale results")
	}
	seedResults(c)
	c.target.SetText("new destination")
	if len(c.files) != 0 {
		t.Fatal("destination change retained stale results")
	}
	seedResults(c)
	c.rule.SetSelected("Daily logs")
	if c.cfg.SelectedRule != "logs" || len(c.files) != 0 {
		t.Fatal("rule selection not exclusive or results stale")
	}
	stored, err := config.Load(c.configPath)
	if err != nil || stored.SelectedRule != "logs" {
		t.Fatal("selection not saved")
	}
}
func TestLanguageCoverage(t *testing.T) {
	c := testController(t)
	for key, value := range translations {
		if value[0] == "" || value[1] == "" {
			t.Errorf("missing language for %s", key)
		}
	}
	seedResults(c)
	c.cfg.Language = "en"
	c.build()
	if c.scan.Text != "Scan files" || len(c.files) != 4 {
		t.Fatal("language switch lost state")
	}
	ctx := c.begin(false)
	if !c.source.Disabled() || !c.copy.Disabled() || c.cancelButton.Disabled() {
		t.Fatal("busy controls inconsistent")
	}
	c.stop()
	if ctx.Err() == nil {
		t.Fatal("cancel did not reach worker context")
	}
	c.finish()
	if c.source.Disabled() || c.copy.Disabled() {
		t.Fatal("controls did not recover")
	}
}

// Set RFE_SCREENSHOTS to an output directory to render both languages during QA.
func TestRenderScreenshots(t *testing.T) {
	dir := os.Getenv("RFE_SCREENSHOTS")
	if dir == "" {
		t.Skip("optional visual QA export")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	c := testController(t)
	c.source.SetText(`D:\TestData`)
	c.target.SetText(`D:\Extracted`)
	seedResults(c)
	for _, lang := range []string{"zh", "en"} {
		c.cfg.Language = lang
		c.build()
		c.status.SetText(c.tr("results") + ": 4")
		c.Window.Resize(fyne.NewSize(1280, 820))
		file, err := os.Create(filepath.Join(dir, "ui-"+lang+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, c.Window.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		for _, width := range []int{800, 1600} {
			c.Window.Resize(fyne.NewSize(float32(width), 720))
			file, err := os.Create(filepath.Join(dir, fmt.Sprintf("ui-%s-%d.png", lang, width)))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, c.Window.Canvas().Capture()); err != nil {
				t.Fatal(err)
			}
			file.Close()
		}
	}
}
