package ui

import (
	"fmt"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"regexfileextractor/internal/config"
	"regexfileextractor/internal/core"
)

// These measure Go layout work with Fyne's test driver, not GPU/Windows frame rates.
func BenchmarkWindowResize(b *testing.B) {
	a := test.NewApp()
	a.Settings().SetTheme(Theme())
	c := New(a, config.Default(), filepath.Join(b.TempDir(), "config.json"), nil)
	c.Window.Show()
	defer a.Quit()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Window.Resize(fyne.NewSize(float32(1000+i%400), float32(720+i%180)))
	}
}

func BenchmarkPreviewSelection(b *testing.B) {
	a := test.NewApp()
	a.Settings().SetTheme(Theme())
	c := New(a, config.Default(), filepath.Join(b.TempDir(), "config.json"), nil)
	defer a.Quit()
	c.files = make([]core.File, 10000)
	c.selected = make([]bool, len(c.files))
	for i := range c.files {
		c.files[i] = core.File{Path: fmt.Sprintf("batch-%05d/X%04d.csv", i, i%100), Name: fmt.Sprintf("X%04d.csv", i%100), Size: 1024, Rule: "XY CSV"}
		c.selected[i] = true
	}
	c.refreshPlan()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.selected[0] = !c.selected[0]
		c.refreshPlan()
	}
}
