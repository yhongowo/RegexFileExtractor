package ui

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
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
	var narrowWidth float32
	for _, size := range []fyne.Size{fyne.NewSize(1280, 820), fyne.NewSize(800, 620), fyne.NewSize(1600, 960), fyne.NewSize(1000, 720)} {
		c.Window.Resize(size)
		view := c.settings
		right := view.panels[1]
		if view.Size().Width-12 < 940 {
			if right.Position().X != 0 || right.Position().Y <= 0 {
				t.Fatal("narrow settings did not stack")
			}
		} else if right.Position().X <= 0 || right.Position().Y != 0 {
			t.Fatal("wide settings did not return to two columns")
		}
		if right.Position().Y+right.Size().Height > view.extent.size.Height+1 {
			t.Fatal("lower settings outside scroll extent")
		}
		for _, p := range view.panels {
			if p.Position().X+p.Size().Width > view.Size().Width {
				t.Fatal("settings overflow viewport")
			}
		}
		if c.resultList.Size().Height < 50 {
			t.Fatal("results collapsed during resize")
		}
		row.Resize(fyne.NewSize(c.resultList.Size().Width, 32))
		end := row.labels[4].Position().X + row.labels[4].Size().Width
		if end > row.Size().Width || row.labels[4].Size().Width < 60 {
			t.Fatal("result columns do not fit")
		}
		if size.Width == 800 {
			narrowWidth = row.labels[1].Size().Width
		}
		if size.Width == 1600 && row.labels[1].Size().Width <= narrowWidth {
			t.Fatal("path column did not expand")
		}
	}
	if created > 100 {
		t.Fatalf("created %d row widgets for 10,000 files; virtualization failed", created)
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
	plan := &c.plan[0]
	row.update(1)
	if &c.plan[0] != plan {
		t.Fatal("painting a row rebuilt the copy plan")
	}
	if row.labels[4].Text != "" {
		t.Fatal("unselected row retained stale output")
	}
}

func TestThemeUsesDefaultFontsAndCompactSizes(t *testing.T) {
	custom, defaults := Theme(), theme.DefaultTheme()
	for _, style := range []fyne.TextStyle{{}, {Bold: true}, {Monospace: true}, {Symbol: true}} {
		if custom.Font(style).Name() != defaults.Font(style).Name() {
			t.Fatalf("font for %+v differs from the Fyne default", style)
		}
	}
	for _, name := range []fyne.ThemeSizeName{theme.SizeNamePadding, theme.SizeNameText, theme.SizeNameInlineIcon} {
		if custom.Size(name) >= 20 {
			t.Fatalf("theme size %s is not compact", name)
		}
	}
}
