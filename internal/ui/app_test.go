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
	c.rule.SetSelected("Datalog")
	if c.cfg.SelectedRule != "datalog" || len(c.files) != 0 {
		t.Fatal("rule selection not exclusive or results stale")
	}
	stored, err := config.Load(c.configPath)
	if err != nil || stored.SelectedRule != "datalog" {
		t.Fatal("selection not saved")
	}
}

func TestHeaderCheckboxSelection(t *testing.T) {
	c := testController(t)
	if !c.selectAllCheck.Disabled() || c.selectAllCheck.Checked {
		t.Fatal("header checkbox should be unavailable without results")
	}
	seedResults(c)
	if !c.selectAllCheck.Checked || c.selectAllCheck.AccessibilityLabel() != c.tr("none") {
		t.Fatal("header checkbox did not reflect all selected")
	}
	test.Tap(c.selectAllCheck)
	if c.selectAllCheck.Checked || len(c.plan) != 0 || !c.copy.Disabled() {
		t.Fatal("header checkbox did not clear selection")
	}
	test.Tap(c.selectAllCheck)
	if !c.selectAllCheck.Checked || len(c.plan) != len(c.files) {
		t.Fatal("header checkbox did not select all")
	}
	c.selected[0] = false
	c.refreshPlan()
	if c.selectAllCheck.Checked || c.selectAllCheck.AccessibilityLabel() != c.tr("all") {
		t.Fatal("header checkbox did not reflect partial selection")
	}
	test.Tap(c.selectAllCheck)
	if !c.selectAllCheck.Checked || len(c.plan) != len(c.files) {
		t.Fatal("header checkbox did not select all from partial state")
	}
	c.begin(false)
	if !c.selectAllCheck.Disabled() {
		t.Fatal("header checkbox should be unavailable while busy")
	}
	c.finish()
	if c.selectAllCheck.Disabled() {
		t.Fatal("header checkbox did not recover after busy state")
	}
	c.invalidate()
	if !c.selectAllCheck.Disabled() || c.selectAllCheck.Checked {
		t.Fatal("header checkbox retained stale selection")
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
	if c.languageButton.Text != "" || c.languageButton.Icon == nil {
		t.Fatal("language control is not icon-only")
	}
	test.Tap(c.languageButton)
	if c.cfg.Language != "en" || c.scan.Text != "Scan" || len(c.files) != 4 {
		t.Fatal("language switch lost state")
	}
	stored, err := config.Load(c.configPath)
	if err != nil || stored.Language != "en" {
		t.Fatal("language switch was not saved")
	}
	test.Tap(c.languageButton)
	if c.cfg.Language != "zh" || len(c.files) != 4 {
		t.Fatal("second tap did not switch back to Chinese")
	}
	ctx := c.begin(false)
	if !c.source.Disabled() || !c.copy.Disabled() || !c.languageButton.Disabled() || c.cancelButton.Disabled() {
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

func TestBrowseOpensFolderDialog(t *testing.T) {
	c := testController(t)
	c.source.SetText(t.TempDir())

	// This must not panic: FileDialog's popup is only initialized by Show.
	c.browse(c.source)
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
	c.cfg.Language = "en"
	c.build()
	c.Window.Resize(fyne.NewSize(800, 600))
	emptyFile, err := os.Create(filepath.Join(dir, "ui-empty-en.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(emptyFile, c.Window.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
	if err := emptyFile.Close(); err != nil {
		t.Fatal(err)
	}
	c.source.SetText(`D:\TestData`)
	c.target.SetText(`D:\Extracted`)
	seedResults(c)
	for _, lang := range []string{"zh", "en"} {
		c.cfg.Language = lang
		c.build()
		c.status.SetText(c.tr("results") + ": 4")
		c.Window.Resize(fyne.NewSize(800, 600))
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
		for _, width := range []int{600, 640, 1200} {
			height := float32(720)
			if width == 600 {
				height = 500
			} else if width == 640 {
				height = 520
			}
			c.Window.Resize(fyne.NewSize(float32(width), height))
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
