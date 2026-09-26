package ui

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

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
	installTestDispatcher(t, c)
	c.Window.Resize(fyne.NewSize(1280, 820))
	c.Window.Show()
	t.Cleanup(func() { awaitBackground(t, c); c.closeWindow(); a.Quit() })
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
	awaitBackground(t, c)
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
	awaitBackground(t, c)
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

func TestDefaultOutputPlaceholder(t *testing.T) {
	c := testController(t)
	if c.target.PlaceHolder != c.tr("targetHint") || c.effectiveDestination() != "" {
		t.Fatal("empty source should retain the generic output hint")
	}
	source := t.TempDir()
	c.source.SetText(source)
	assertDefault := func() {
		t.Helper()
		want := filepath.Join(c.source.Text, "extracted")
		if c.target.Text != "" || c.cfg.Destination != "" || c.target.PlaceHolder != want || c.effectiveDestination() != want {
			t.Fatalf("default output mismatch: value=%q config=%q placeholder=%q resolved=%q", c.target.Text, c.cfg.Destination, c.target.PlaceHolder, c.effectiveDestination())
		}
	}
	assertDefault()
	c.source.SetText(filepath.Join(source, "next"))
	assertDefault()
	custom := filepath.Join(source, "custom")
	c.target.SetText(custom) // Browse also applies its selection through SetText.
	c.source.SetText(source)
	if c.target.Text != custom || c.effectiveDestination() != custom {
		t.Fatal("source change replaced custom output")
	}
	c.target.SetText("")
	assertDefault()
	c.setLanguage("en")
	assertDefault()
	awaitBackground(t, c)
	stored, err := config.Load(c.configPath)
	if err != nil || stored.Destination != "" {
		t.Fatalf("default was saved as an explicit destination: %+v, %v", stored, err)
	}
	c.cfg = stored
	c.build()
	assertDefault()
	c.source.SetText("")
	if c.target.PlaceHolder != c.tr("targetHint") || c.effectiveDestination() != "" {
		t.Fatal("clearing source retained a stale default")
	}
}

func TestDefaultOutputCopyAndRescan(t *testing.T) {
	c := testController(t)
	source := t.TempDir()
	c.source.SetText(source)
	name := "X01Y01.csv"
	if err := os.WriteFile(filepath.Join(source, name), []byte("sample"), 0644); err != nil {
		t.Fatal(err)
	}
	rule := core.Rule{ID: "test", Name: "test", Pattern: `.*\.csv`}
	opts := core.ScanOptions{Source: c.cfg.Source, Exclude: c.effectiveDestination(), Rule: rule}
	result, err := core.Scan(context.Background(), opts, nil)
	if err != nil || len(result.Files) != 1 {
		t.Fatalf("scan: %+v, %v", result, err)
	}
	plan, err := core.Plan(result.Files, core.Flat)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := core.Copy(context.Background(), c.effectiveDestination(), plan, core.Skip, nil)
	if err != nil || copied.Copied != 1 {
		t.Fatalf("copy: %+v, %v", copied, err)
	}
	data, err := os.ReadFile(filepath.Join(source, "extracted", name))
	if err != nil || string(data) != "sample" {
		t.Fatalf("default output missing or incorrect: %q, %v", data, err)
	}
	result, err = core.Scan(context.Background(), opts, nil)
	if err != nil || len(result.Files) != 1 {
		t.Fatalf("rescan included extracted output: %+v, %v", result, err)
	}
	if c.target.Text != "" || c.cfg.Destination != "" {
		t.Fatal("copy populated the output value")
	}
}

// Fyne's test driver executes Do immediately. Explicitly pump callbacks on the
// test goroutine to model the native event loop and avoid concurrent widget access.
var testDispatchers sync.Map

func installTestDispatcher(t testing.TB, c *Controller) {
	events := make(chan func(), 100)
	c.dispatch = func(f func()) { events <- f }
	testDispatchers.Store(c, events)
	t.Cleanup(func() { testDispatchers.Delete(c) })
}
func awaitBackground(t testing.TB, c *Controller) {
	t.Helper()
	value, ok := testDispatchers.Load(c)
	if !ok {
		t.Fatal("missing test dispatcher")
	}
	events := value.(chan func())
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	for c.planRunning || c.saveRunning || c.busy {
		select {
		case f := <-events:
			f()
		case <-timeout.C:
			t.Fatal("background operation timed out")
		}
	}
}
