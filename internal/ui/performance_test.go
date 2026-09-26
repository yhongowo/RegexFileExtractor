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
	installTestDispatcher(b, c)
	defer a.Quit()
	defer func() { awaitBackground(b, c) }()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Window.Resize(fyne.NewSize(float32(1000+i%400), float32(720+i%180)))
	}
}

// Resizing height while keeping width is common when snapping or dragging a
// window edge. The settings panel should not remeasure wrapped content then.
func BenchmarkWindowVerticalResize(b *testing.B) {
	a := test.NewApp()
	a.Settings().SetTheme(Theme())
	c := New(a, config.Default(), filepath.Join(b.TempDir(), "config.json"), nil)
	c.Window.Show()
	installTestDispatcher(b, c)
	defer a.Quit()
	defer func() { awaitBackground(b, c) }()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Window.Resize(fyne.NewSize(1000, float32(600+i%180)))
	}
}

func BenchmarkPreviewSelection(b *testing.B) {
	a := test.NewApp()
	a.Settings().SetTheme(Theme())
	c := New(a, config.Default(), filepath.Join(b.TempDir(), "config.json"), nil)
	installTestDispatcher(b, c)
	defer a.Quit()
	defer func() { awaitBackground(b, c) }()
	c.files = make([]core.File, 10000)
	c.selected = make([]bool, len(c.files))
	for i := range c.files {
		c.files[i] = core.File{Path: fmt.Sprintf("batch-%05d/X%04d.csv", i, i%100), Name: fmt.Sprintf("X%04d.csv", i%100), Size: 1024, Rule: "XY CSV"}
		c.selected[i] = true
	}
	c.refreshPlan()
	awaitBackground(b, c)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.selected[0] = !c.selected[0]
		c.refreshPlan()
		awaitBackground(b, c)
	}
}

func BenchmarkWindowResizeWithResults(b *testing.B) {
	a := test.NewApp()
	a.Settings().SetTheme(Theme())
	c := New(a, config.Default(), filepath.Join(b.TempDir(), "config.json"), nil)
	installTestDispatcher(b, c)
	defer a.Quit()
	defer func() { awaitBackground(b, c) }()
	c.files = make([]core.File, 10000)
	c.selected = make([]bool, len(c.files))
	for i := range c.files {
		c.files[i] = core.File{Path: fmt.Sprintf("D:/measurements/production-line/batch-%05d/channel-01/X%04d.csv", i, i%100), Name: fmt.Sprintf("X%04d.csv", i%100), Size: 1024, Rule: "XY CSV"}
		c.selected[i] = true
	}
	c.refreshPlan()
	awaitBackground(b, c)
	c.Window.Show()
	c.Window.Resize(fyne.NewSize(1000, 720))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Window.Resize(fyne.NewSize(float32(800+i%600), float32(600+i%180)))
	}
}

// Measures the event handler only; completion latency is measured separately by
// BenchmarkPreviewSelection. Keep the UI dispatcher paused to model rapid input.
func BenchmarkPreviewSelectionDispatch(b *testing.B) {
	for _, count := range []int{10000, 100000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			a := test.NewApp()
			a.Settings().SetTheme(Theme())
			c := New(a, config.Default(), filepath.Join(b.TempDir(), "config.json"), nil)
			installTestDispatcher(b, c)
			defer a.Quit()
			defer func() { awaitBackground(b, c) }()
			seedLargeResults(c, count)
			awaitBackground(b, c)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				c.selected[0] = !c.selected[0]
				c.refreshPlan()
			}
			b.StopTimer()
		})
	}
}
