package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// One pooled row owns one checkbox and five labels, instead of twelve widgets
// plus six stacked containers in each row of the old two-dimensional Table.
type resultRow struct {
	widget.BaseWidget
	owner    *Controller
	id       int
	check    *widget.Check
	labels   [5]*singleLineLabel
	content  *fyne.Container
	fileSize int64
	filePath string
}

func newResultRow(c *Controller) *resultRow {
	r := &resultRow{owner: c, id: -1}
	r.ExtendBaseWidget(r)
	r.check = widget.NewCheck("", func(value bool) {
		if c.busy || r.id < 0 || r.id >= len(c.selected) || c.selected[r.id] == value {
			return
		}
		c.selected[r.id] = value
		c.refreshPlan()
	})
	objects := []fyne.CanvasObject{r.check}
	for i := range r.labels {
		r.labels[i] = newSingleLineLabel("")
		objects = append(objects, r.labels[i])
	}
	r.content = container.New(resultColumns{}, objects...)
	return r
}
func (r *resultRow) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(r.content) }
func (r *resultRow) update(id int) {
	c := r.owner
	if id < 0 || id >= len(c.files) {
		return
	}
	r.id = id
	// Suppress callbacks while applying model state, including recycled rows.
	changed := r.check.OnChanged
	r.check.OnChanged = nil
	if r.check.Checked != c.selected[id] {
		r.check.SetChecked(c.selected[id])
	}
	r.check.OnChanged = changed
	setDisabled(r.check, c.busy)
	f := c.files[id]
	size := r.labels[2].Text
	if r.filePath != f.Path || r.fileSize != f.Size {
		size = formatSize(f.Size)
		r.filePath = f.Path
		r.fileSize = f.Size
	}
	values := [5]string{f.Name, f.Path, size, f.Rule, c.outputs[f.Path]}
	for i, value := range values {
		r.labels[i].SetText(value)
	}
}

// All columns share the available width; full values remain available in details.
// Header and rows use the same geometry and reserve space for the vertical bar.
type resultColumns struct{}

func (resultColumns) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(520, 32) }
func (resultColumns) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 6 {
		return
	}
	width := max(float32(0), size.Width-theme.Size(theme.SizeNameScrollBar)-6)
	widths := resultColumnWidths(width)
	x := float32(0)
	for i, obj := range objects {
		obj.Move(fyne.NewPos(x, 0))
		obj.Resize(fyne.NewSize(widths[i], size.Height))
		x += widths[i]
	}
}
func resultColumnWidths(width float32) [6]float32 {
	check, size := float32(32), float32(68)
	remaining := max(float32(0), width-check-size)
	name, path, rule := remaining*.20, remaining*.38, remaining*.13
	return [6]float32{check, name, path, size, rule, remaining - name - path - rule}
}

type selectionHeaderCheck struct {
	widget.Check
	label string
}

func newSelectionHeaderCheck(label string, changed func(bool)) *selectionHeaderCheck {
	check := &selectionHeaderCheck{label: label}
	check.OnChanged = changed
	check.ExtendBaseWidget(check)
	return check
}

func (c *selectionHeaderCheck) AccessibilityLabel() string { return c.label }

func (c *Controller) makeResults() (*widget.List, fyne.CanvasObject) {
	list := widget.NewList(func() int { return len(c.files) }, func() fyne.CanvasObject { return newResultRow(c) }, func(id widget.ListItemID, obj fyne.CanvasObject) { obj.(*resultRow).update(id) })
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(c.files) {
			return
		}
		f := c.files[id]
		c.showText(c.tr("fileDetails"), fmt.Sprintf("%s: %s\n%s: %s\n%s: %s (%d B)\n%s: %s\n%s: %s", c.tr("filename"), f.Name, c.tr("path"), f.Path, c.tr("size"), formatSize(f.Size), f.Size, c.tr("matchedRule"), f.Rule, c.tr("outputPath"), c.outputs[f.Path]))
		list.UnselectAll()
	}
	c.selectAllCheck = newSelectionHeaderCheck(c.tr("all"), func(checked bool) { c.selectAll(checked) })
	headers := []fyne.CanvasObject{c.selectAllCheck}
	for _, key := range []string{"filename", "path", "size", "matchedRule", "outputPath"} {
		label := newSingleLineLabel(c.tr(key))
		headers = append(headers, label)
	}
	return list, container.New(resultColumns{}, headers...)
}

func setLabel(label *widget.Label, text string) {
	if label.Text != text {
		label.SetText(text)
	}
}
func setDisabled(w fyne.Disableable, disabled bool) {
	if w.Disabled() == disabled {
		return
	}
	if disabled {
		w.Disable()
	} else {
		w.Enable()
	}
}
