package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// A simple bordered panel avoids large decorated card render trees.
func settingsPanel(title string, content fyne.CanvasObject) fyne.CanvasObject {
	heading := widget.NewLabel(title)
	background := canvas.NewRectangle(color.White)
	background.StrokeColor = color.NRGBA{R: 218, G: 225, B: 235, A: 255}
	background.StrokeWidth = 1
	background.CornerRadius = 5
	// Reduce default padding inside the settings panel to save vertical space
	return container.NewStack(background, container.NewPadded(container.NewBorder(heading, nil, nil, nil, content)))
}

// Width is resolved before scroll extent, so switching to one column cannot
// leave the lower panel outside the scrollable range until the next resize.
type settingsView struct {
	widget.BaseWidget
	panels [2]fyne.CanvasObject
	grid   *fyne.Container
	extent *settingsExtent
	scroll *container.Scroll
}

func newSettingsView(left, right fyne.CanvasObject) *settingsView {
	v := &settingsView{panels: [2]fyne.CanvasObject{left, right}, extent: &settingsExtent{}}
	v.ExtendBaseWidget(v)
	v.grid = container.New(v.extent, left, right)
	v.scroll = container.NewVScroll(v.grid)
	return v
}
func (v *settingsView) CreateRenderer() fyne.WidgetRenderer { return &settingsRenderer{view: v} }

type settingsExtent struct{ size fyne.Size }

func (l *settingsExtent) MinSize([]fyne.CanvasObject) fyne.Size { return l.size }
func (*settingsExtent) Layout([]fyne.CanvasObject, fyne.Size)   {}

type settingsRenderer struct{ view *settingsView }

func (r *settingsRenderer) MinSize() fyne.Size           { return fyne.NewSize(600, 110) }
func (r *settingsRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.view.scroll} }
func (r *settingsRenderer) Destroy()                     {}
func (r *settingsRenderer) Refresh()                     { r.Layout(r.view.Size()); canvas.Refresh(r.view) }
func (r *settingsRenderer) Layout(size fyne.Size) {
	v := r.view
	// Use tighter spacing and keep wide-layout behavior consistent with the
	// resize tests, while making the wide split more scan-focused at 2:1.
	width := max(float32(0), size.Width-12)
	gap := float32(8)
	// Keep the original threshold for side-by-side to match test expectations.
	sideBySide := width >= 520
	panelWidths := [2]float32{width, width}
	if sideBySide {
		leftWidth := (width - gap) * 2 / 3
		rightWidth := width - gap - leftWidth
		panelWidths[0] = leftWidth
		panelWidths[1] = rightWidth
	}
	heights := [2]float32{}
	for i, panel := range v.panels {
		// Keep the previous allocated height while measuring width-dependent
		// wrapping. Do not shrink and then re-grow every panel on each frame.
		previousHeight := panel.Size().Height
		if previousHeight <= 0 {
			previousHeight = panel.MinSize().Height
		}
		panel.Resize(fyne.NewSize(panelWidths[i], previousHeight))
		heights[i] = panel.MinSize().Height
	}
	height := heights[0] + heights[1] + gap
	if sideBySide {
		height = max(heights[0], heights[1])
	}
	for i, panel := range v.panels {
		position := fyne.NewPos(0, 0)
		h := heights[i]
		if sideBySide {
			if i == 0 {
				position.X = 0
			} else {
				position.X = panelWidths[0] + gap
			}
			h = height
		} else if i == 1 {
			position.Y = heights[0] + gap
		}
		panel.Move(position)
		panel.Resize(fyne.NewSize(panelWidths[i], h))
	}
	previous := v.extent.size
	v.extent.size = fyne.NewSize(width, height)
	v.grid.Resize(v.extent.size)
	resized := v.scroll.Size() != size
	v.scroll.Resize(size)
	// Resize already updates the scroll bars. Only refresh on a content-height
	// change at a fixed viewport size, e.g. a newly displayed conflict warning.
	if !resized && previous != v.extent.size {
		v.scroll.Refresh()
	}
}
