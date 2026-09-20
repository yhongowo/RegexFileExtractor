package ui

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"regexfileextractor/internal/core"
)

func TestResizeAndVirtualization(t *testing.T) {
	c := testController(t)
	c.files = make([]core.File, 10000)
	c.selected = make([]bool, len(c.files))
	for i := range c.files {
		c.files[i] = core.File{Path: fmt.Sprintf("source/%05d/X.csv", i), Name: "X.csv", Size: 12, Rule: "XY"}
		c.selected[i] = true
	}
	created := 0
	create := c.resultList.CreateItem
	c.resultList.CreateItem = func() fyne.CanvasObject { created++; return create() }
	c.refreshPlan()
	row := newResultRow(c)
	row.update(0)
	iconCommands := 0
	for _, control := range c.controls {
		if button, ok := control.(*labeledIconButton); ok {
			if button.Text != "" || button.AccessibilityLabel() == "" {
				t.Fatal("icon commands must retain an accessible name")
			}
			if button != c.languageButton {
				iconCommands++
			}
		}
	}
	if iconCommands != 6 {
		t.Fatalf("found %d icon commands, want 6", iconCommands)
	}
	near := func(a, b float32) bool { return a >= b-1 && a <= b+1 }
	var narrowWidth float32
	for _, size := range []fyne.Size{fyne.NewSize(800, 600), fyne.NewSize(600, 500), fyne.NewSize(640, 520), fyne.NewSize(700, 560), fyne.NewSize(1200, 760), fyne.NewSize(1000, 720)} {
		// The test driver allows sizes below the native window minimum.
		c.Window.Resize(size.Max(c.Window.Content().MinSize()))
		view := c.settings
		right := view.panels[1]
		if right.Size().Width != extractionSettingsWidth {
			t.Fatal("extraction settings width changed during resize")
		}
		if right.Position().X <= 0 || right.Position().Y != 0 {
			t.Fatal("settings must remain side by side")
		}
		if right.Size().Height > view.extent.size.Height+1 {
			t.Fatal("settings outside scroll extent")
		}
		for _, p := range view.panels {
			if p.Position().X+p.Size().Width > view.Size().Width {
				t.Fatal("settings overflow viewport")
			}
		}
		leftPos := c.app.Driver().AbsolutePositionForObject(view.panels[0])
		rightPos := c.app.Driver().AbsolutePositionForObject(view.panels[1])
		resultPos := c.app.Driver().AbsolutePositionForObject(c.resultsCard)
		if !near(leftPos.X, resultPos.X) || !near(rightPos.X+right.Size().Width, resultPos.X+c.resultsCard.Size().Width) {
			t.Fatalf("cards not horizontally aligned at %.0f width: left=%v right=%v results=%v", size.Width, leftPos, rightPos, resultPos)
		}
		if !near(resultPos.Y, leftPos.Y+view.panels[0].Size().Height+8) {
			t.Fatalf("card row gap inconsistent at %.0f width: settings=%v size=%v results=%v", size.Width, leftPos, view.panels[0].Size(), resultPos)
		}
		if c.resultList.Size().Height < 50 {
			t.Fatal("results collapsed during resize")
		}
		row.Resize(fyne.NewSize(c.resultList.Size().Width, 32))
		end := row.labels[4].Position().X + row.labels[4].Size().Width
		if end > row.Size().Width || row.labels[4].Size().Width < 60 {
			t.Fatal("result columns do not fit")
		}
		if size.Width == 600 {
			narrowWidth = row.labels[1].Size().Width
			if c.rule.Size().Width < 90 {
				t.Fatal("rule selector lost space at 600 width")
			}
		}
		if size.Width == 1200 && row.labels[1].Size().Width <= narrowWidth {
			t.Fatal("path column did not expand")
		}
	}
	if created > 100 {
		t.Fatalf("created %d row widgets for 10,000 files; virtualization failed", created)
	}
}

func TestVerticalResizeOnlyChangesResultsHeight(t *testing.T) {
	c := testController(t)
	var settingsHeight, resultsHeight float32
	for i, height := range []float32{600, 720, 840, 660} {
		c.Window.Resize(fyne.NewSize(800, height).Max(c.Window.Content().MinSize()))
		if i == 0 {
			settingsHeight = c.settings.Size().Height
			resultsHeight = c.resultsCard.Size().Height
			continue
		}
		if c.settings.Size().Height != settingsHeight {
			t.Fatalf("settings height changed during vertical resize: got %.0f, want %.0f", c.settings.Size().Height, settingsHeight)
		}
		if height > 600 && c.resultsCard.Size().Height <= resultsHeight {
			t.Fatalf("results did not receive added vertical space at height %.0f", height)
		}
	}
	if _, ok := c.settings.CreateRenderer().Objects()[0].(*container.Scroll); ok {
		t.Fatal("settings must not be scrollable")
	}
}

func TestRecycledRowChangesOnlyItsCurrentSelection(t *testing.T) {
	c := testController(t)
	seedResults(c)
	row := newResultRow(c)
	row.update(0)
	row.update(1)
	test.Tap(row.check)
	if !c.selected[0] || c.selected[1] {
		t.Fatal("recycled checkbox changed the wrong file")
	}
	if c.selectAllCheck.Checked {
		t.Fatal("header checkbox did not reflect the row selection")
	}
	plan := &c.plan[0]
	row.update(1)
	if &c.plan[0] != plan {
		t.Fatal("painting a row rebuilt the copy plan")
	}
	if row.labels[4].Text != "" {
		t.Fatal("unselected row retained stale output")
	}
}

func TestLabeledIconButtonActivatesWithoutVisibleText(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	pressed := 0
	button := newLabeledIconButton("Browse source folder", folderIcon, func() { pressed++ })
	if button.Text != "" || button.AccessibilityLabel() != "Browse source folder" {
		t.Fatal("icon button lost its accessible action name")
	}
	test.Tap(button)
	if pressed != 1 {
		t.Fatal("icon button did not activate")
	}
}

func TestThemeUsesWindowsUIFontAndCompactSizes(t *testing.T) {
	custom, defaults := Theme(), theme.DefaultTheme()
	for _, style := range []fyne.TextStyle{{}, {Bold: true}, {Monospace: true}, {Symbol: true}} {
		want := defaults.Font(style)
		if !style.Monospace && !style.Symbol {
			if system := uiFont(style.Bold); system != nil {
				want = system
			}
		}
		if custom.Font(style).Name() != want.Name() {
			t.Fatalf("font for %+v is %q, want %q", style, custom.Font(style).Name(), want.Name())
		}
	}
	for _, name := range []fyne.ThemeSizeName{theme.SizeNamePadding, theme.SizeNameText, theme.SizeNameInlineIcon} {
		if custom.Size(name) >= 20 {
			t.Fatalf("theme size %s is not compact", name)
		}
	}
	if custom.Size(theme.SizeNameText) < 12 || custom.Size(theme.SizeNameCaptionText) < 12 {
		t.Fatal("compact typography dropped below the legibility floor")
	}
}

// Track expensive work below a container, not only the outer widget's resize.
type measuredPanel struct {
	fyne.CanvasObject
	refreshes, measurements int
	wrap                    bool
}

func (p *measuredPanel) Refresh() { p.refreshes++; p.CanvasObject.Refresh() }
func (p *measuredPanel) MinSize() fyne.Size {
	p.measurements++
	if p.wrap && p.Size().Width < 500 {
		return fyne.NewSize(100, 300)
	}
	return fyne.NewSize(100, 200)
}

func TestSettingsResizeAvoidsRecursiveRefresh(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	left := &measuredPanel{CanvasObject: container.NewWithoutLayout(), wrap: true}
	right := &measuredPanel{CanvasObject: container.NewWithoutLayout()}
	view := newSettingsView(left, right)
	results := container.NewWithoutLayout()
	objects := []fyne.CanvasObject{view, results}
	layout := workspaceLayout{gap: 8}
	layout.Layout(objects, fyne.NewSize(1000, 700))
	left.refreshes, right.refreshes = 0, 0
	right.measurements = 0
	for _, width := range []float32{700, 1000, 680, 900} {
		layout.Layout(objects, fyne.NewSize(width, 700))
		wantHeight := float32(200)
		if width-8-extractionSettingsWidth < 500 {
			wantHeight = 300
		}
		if view.Size().Height != wantHeight || results.Position().Y != wantHeight+8 {
			t.Fatalf("settings/results did not follow wrapped height at width %v: %v, %v", width, view.Size(), results.Position())
		}
	}
	if left.refreshes != 0 || right.refreshes != 0 {
		t.Fatalf("resize recursively refreshed panels: %d / %d", left.refreshes, right.refreshes)
	}
	if right.measurements != 0 {
		t.Fatalf("fixed-width panel was remeasured %d times", right.measurements)
	}
	view.Refresh()
	if left.refreshes == 0 || right.refreshes == 0 || right.measurements == 0 {
		t.Fatal("explicit refresh must still update content and measurements")
	}
}
