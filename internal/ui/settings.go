package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// White surfaces share a quiet border; controls use the tighter Win11 radius.
func borderedSurface(content fyne.CanvasObject, radius float32) fyne.CanvasObject {
	background := canvas.NewRectangle(color.White)
	background.StrokeColor = color.NRGBA{R: 218, G: 222, B: 228, A: 255}
	background.StrokeWidth = 1
	background.CornerRadius = radius
	return container.NewStack(background, content)
}
func cardSurface(content fyne.CanvasObject) fyne.CanvasObject { return borderedSurface(content, 8) }

func outlineButton(button *widget.Button) fyne.CanvasObject {
	button.Importance = widget.LowImportance
	return borderedSurface(button, 4)
}

// Fyne uses the icon filename as the accessible name of a textless Button.
// Keep a localized action name while rendering only the icon.
type labeledIconButton struct {
	widget.Button
	label string
}

func newLabeledIconButton(label string, icon fyne.Resource, tapped func()) *labeledIconButton {
	button := &labeledIconButton{label: label}
	button.Icon = icon
	button.OnTapped = tapped
	button.Importance = widget.LowImportance
	button.ExtendBaseWidget(button)
	return button
}

func (b *labeledIconButton) AccessibilityLabel() string { return b.label }

func outlineIconButton(button *labeledIconButton) fyne.CanvasObject {
	return sizedButton(borderedSurface(button, 4), 30, 28)
}

func sectionHeading(title string, icon fyne.Resource) fyne.CanvasObject {
	heading := canvas.NewText(title, color.NRGBA{R: 28, G: 35, B: 45, A: 255})
	heading.TextSize = 14
	heading.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewHBox(widget.NewIcon(icon), heading)
}

func settingsPanel(title string, icon fyne.Resource, content fyne.CanvasObject) fyne.CanvasObject {
	return cardSurface(container.NewPadded(container.NewVBox(
		sectionHeading(title, icon),
		content,
	)))
}

// Fixed label width keeps the three scan fields aligned without the tall
// spacing of a Form. The right-hand control uses all remaining width.
type fieldLayout struct{ labelWidth float32 }

func (l fieldLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 2 {
		return fyne.Size{}
	}
	control := objects[1].MinSize()
	return fyne.NewSize(l.labelWidth+control.Width+8,
		max(objects[0].MinSize().Height, control.Height))
}
func (l fieldLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 2 {
		return
	}
	labelHeight := objects[0].MinSize().Height
	objects[0].Move(fyne.NewPos(0, (size.Height-labelHeight)/2))
	objects[0].Resize(fyne.NewSize(l.labelWidth, labelHeight))
	x := l.labelWidth + 8
	objects[1].Move(fyne.NewPos(x, 0))
	objects[1].Resize(fyne.NewSize(max(float32(0), size.Width-x), size.Height))
}

func fieldRow(label string, control fyne.CanvasObject) fyne.CanvasObject {
	return container.New(fieldLayout{labelWidth: 96}, widget.NewLabel(label), control)
}

type sizedButtonLayout struct{ width, height float32 }

func (l sizedButtonLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 1 {
		return fyne.Size{}
	}
	min := objects[0].MinSize()
	return fyne.NewSize(max(l.width, min.Width), max(l.height, min.Height))
}
func (sizedButtonLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 1 {
		return
	}
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)
}

func sizedButton(button fyne.CanvasObject, width, height float32) fyne.CanvasObject {
	return container.New(sizedButtonLayout{width: width, height: height}, button)
}

type gapRowLayout struct{ gap float32 }

func (l gapRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var width, height float32
	for i, obj := range objects {
		if i > 0 {
			width += l.gap
		}
		min := obj.MinSize()
		width += min.Width
		height = max(height, min.Height)
	}
	return fyne.NewSize(width, height)
}
func (l gapRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	x := float32(0)
	for _, obj := range objects {
		width := obj.MinSize().Width
		obj.Move(fyne.NewPos(x, 0))
		obj.Resize(fyne.NewSize(width, size.Height))
		x += width + l.gap
	}
}
func gapRow(gap float32, objects ...fyne.CanvasObject) fyne.CanvasObject {
	return container.New(gapRowLayout{gap: gap}, objects...)
}

type emptyStateLayout struct{}

func (emptyStateLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 3 {
		return fyne.Size{}
	}
	icon, title, hint := objects[0].MinSize(), objects[1].MinSize(), objects[2].MinSize()
	return fyne.NewSize(max(icon.Width, title.Width, hint.Width), icon.Height+8+title.Height+2+hint.Height)
}
func (emptyStateLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 3 {
		return
	}
	y := float32(0)
	for i, obj := range objects {
		min := obj.MinSize()
		obj.Move(fyne.NewPos((size.Width-min.Width)/2, y))
		obj.Resize(min)
		y += min.Height
		if i == 0 {
			y += 8
		} else if i == 1 {
			y += 2
		}
	}
}

// Keep native radio behavior and styling. Each option gets an equal-width
// column, with content aligned from the leading to the trailing edge.
type equalWidthRadioGroup struct {
	widget.RadioGroup
}

func newEqualWidthRadioGroup(options []string) *equalWidthRadioGroup {
	r := &equalWidthRadioGroup{}
	r.Options = options
	r.Horizontal = true
	r.ExtendBaseWidget(r)
	return r
}

func (r *equalWidthRadioGroup) CreateRenderer() fyne.WidgetRenderer {
	return &equalWidthRadioRenderer{WidgetRenderer: r.RadioGroup.CreateRenderer(), radio: r}
}

type equalWidthRadioRenderer struct {
	fyne.WidgetRenderer
	radio *equalWidthRadioGroup
}

func (r *equalWidthRadioRenderer) Layout(size fyne.Size) {
	items := r.Objects()
	if len(items) == 0 {
		return
	}
	width := size.Width / float32(len(items))
	for i, item := range items {
		itemWidth := min(width, item.MinSize().Width)
		alignment := float32(0)
		if len(items) > 1 {
			alignment = float32(i) / float32(len(items)-1)
		}
		x := float32(i)*width + (width-itemWidth)*alignment
		item.Move(fyne.NewPos(x, 0))
		item.Resize(fyne.NewSize(itemWidth, size.Height))
	}
}

func (r *equalWidthRadioRenderer) Refresh() {
	r.WidgetRenderer.Refresh()
	r.Layout(r.radio.Size())
}

const extractionSettingsWidth float32 = 200

// The settings always stay in two columns and keep their natural height. The
// window minimum size protects this area from being compressed, so only the
// results area participates in vertical window resizing.
type settingsView struct {
	widget.BaseWidget
	panels [2]fyne.CanvasObject
	grid   *fyne.Container
	extent *settingsExtent

	measured      bool
	measureWidth  [2]float32
	measureHeight [2]float32
	fixedHeight   float32
}

func newSettingsView(left, right fyne.CanvasObject) *settingsView {
	v := &settingsView{panels: [2]fyne.CanvasObject{left, right}, extent: &settingsExtent{}}
	v.ExtendBaseWidget(v)
	v.grid = container.New(v.extent, left, right)
	return v
}
func (v *settingsView) CreateRenderer() fyne.WidgetRenderer { return &settingsRenderer{view: v} }

// invalidateMeasure is needed when width-dependent content (the warning text)
// changes. Pure height resizes reuse the last measurement.
func (v *settingsView) invalidateMeasure() { v.measured = false }

type settingsExtent struct{ size fyne.Size }

func (l *settingsExtent) MinSize([]fyne.CanvasObject) fyne.Size { return l.size }
func (*settingsExtent) Layout([]fyne.CanvasObject, fyne.Size)   {}

type settingsRenderer struct{ view *settingsView }

func (r *settingsRenderer) MinSize() fyne.Size {
	return fyne.NewSize(660, max(float32(200), r.view.fixedHeight))
}
func (r *settingsRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.view.grid} }
func (r *settingsRenderer) Destroy()                     {}
func (r *settingsRenderer) Refresh() {
	// Explicit content/theme changes refresh children and invalidate measurements.
	// A resize must only update geometry, not recursively refresh every control.
	r.view.grid.Refresh()
	r.view.invalidateMeasure()
	r.Layout(r.view.Size())
	canvas.Refresh(r.view)
}
func (r *settingsRenderer) Layout(size fyne.Size) {
	v := r.view
	width := max(float32(0), size.Width)
	gap := float32(8)
	leftWidth := max(float32(0), width-gap-extractionSettingsWidth)
	panelWidths := [2]float32{leftWidth, extractionSettingsWidth}
	if !v.measured || v.measureWidth != panelWidths {
		for i, panel := range v.panels {
			if v.measured && v.measureWidth[i] == panelWidths[i] {
				continue
			}
			// Keep the previous allocated height while measuring width-dependent
			// wrapping. Do not shrink and then re-grow every panel on each frame.
			previousHeight := panel.Size().Height
			if previousHeight <= 0 {
				previousHeight = panel.MinSize().Height
			}
			panel.Resize(fyne.NewSize(panelWidths[i], previousHeight))
			v.measureHeight[i] = panel.MinSize().Height
		}
		v.measureWidth = panelWidths
		v.measured = true
	}
	height := max(float32(200), v.measureHeight[0], v.measureHeight[1])
	v.fixedHeight = height
	for i, panel := range v.panels {
		x := float32(0)
		if i == 1 {
			x = panelWidths[0] + gap
		}
		panel.Move(fyne.NewPos(x, 0))
		panel.Resize(fyne.NewSize(panelWidths[i], height))
	}
	v.extent.size = fyne.NewSize(width, height)
	v.grid.Resize(v.extent.size)
}

// Settings and results use the same horizontal edges and a single gap.
type workspaceLayout struct{ gap float32 }

func (l workspaceLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 2 {
		return fyne.Size{}
	}
	settings, results := objects[0].MinSize(), objects[1].MinSize()
	return fyne.NewSize(max(settings.Width, results.Width), settings.Height+l.gap+results.Height)
}

func (l workspaceLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 2 {
		return
	}
	settingsHeight := objects[0].MinSize().Height
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(size.Width, settingsHeight))
	// Width-dependent wrapping may change the natural height during Resize.
	// Use the new measurement immediately, without recursively refreshing children.
	if measuredHeight := objects[0].MinSize().Height; measuredHeight != settingsHeight {
		settingsHeight = measuredHeight
		objects[0].Resize(fyne.NewSize(size.Width, settingsHeight))
	}
	objects[1].Move(fyne.NewPos(0, settingsHeight+l.gap))
	objects[1].Resize(fyne.NewSize(size.Width, max(float32(0), size.Height-settingsHeight-l.gap)))
}
