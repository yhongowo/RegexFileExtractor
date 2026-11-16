package ui

import (
	"fmt"
	"image/color"

	"regexfileextractor/internal/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type datalogOverview struct {
	widget.BaseWidget
	content *fyne.Container
	title   *widget.Label
	stats   *widget.Label
	bar     *completenessBar
}

func newDatalogOverview() *datalogOverview {
	o := &datalogOverview{}
	o.title = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	o.stats = widget.NewLabel("")
	o.stats.Wrapping = fyne.TextWrapWord
	o.bar = newCompletenessBar()
	o.content = container.NewPadded(container.NewVBox(o.title, o.stats, o.bar))
	o.ExtendBaseWidget(o)
	o.Hide()
	return o
}

func (o *datalogOverview) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(o.content)
}

func (c *Controller) refreshDatalogOverview() {
	if c.overview == nil {
		return
	}
	if c.sizeControls != nil {
		datalog := c.isDatalog()
		changed := c.sizeControls.Visible() != datalog
		if datalog {
			c.sizeControls.Show()
		} else {
			c.sizeControls.Hide()
		}
		if changed && c.settings != nil {
			c.settings.invalidateMeasure()
			c.settings.Refresh()
		}
	}
	if c.layoutHint != nil {
		key := string(c.cfg.Layout) + "Hint"
		if c.isDatalog() {
			key = "datalogLayoutHint"
		}
		setLabel(c.layoutHint, c.tr(key))
	}
	if !c.isDatalog() || len(c.files) == 0 {
		c.overview.Hide()
		return
	}
	var complete, abnormal, selected int
	dies := make(map[string]int)
	for i, file := range c.files {
		if i >= len(c.selected) || !c.selected[i] {
			continue
		}
		selected++
		if core.IsDatalogComplete(file.Size, c.cfg.DatalogMinKB, c.cfg.DatalogMaxKB) {
			complete++
		} else {
			abnormal++
		}
		if key, _, err := core.DatalogCoordinate(file.Name); err == nil {
			dies[key]++
		}
	}
	repeatedDies := 0
	for _, count := range dies {
		if count > 1 {
			repeatedDies++
		}
	}
	percentage := float64(0)
	if selected > 0 {
		percentage = float64(complete) * 100 / float64(selected)
	}
	setLabel(c.overview.title, fmt.Sprintf(c.tr("datalogOverview"), selected, len(dies)))
	setLabel(c.overview.stats, fmt.Sprintf(c.tr("datalogCompleteness"), complete, abnormal, percentage, repeatedDies, c.cfg.DatalogMinKB, c.cfg.DatalogMaxKB))
	c.overview.bar.Set(complete, abnormal)
	c.overview.Show()
}

type completenessBar struct {
	widget.BaseWidget
	background *canvas.Rectangle
	complete   *canvas.Rectangle
	abnormal   *canvas.Rectangle
	good, bad  int
}

func newCompletenessBar() *completenessBar {
	b := &completenessBar{
		background: canvas.NewRectangle(color.NRGBA{R: 229, G: 231, B: 235, A: 255}),
		complete:   canvas.NewRectangle(color.NRGBA{R: 34, G: 163, B: 92, A: 255}),
		abnormal:   canvas.NewRectangle(color.NRGBA{R: 220, G: 68, B: 68, A: 255}),
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *completenessBar) Set(complete, abnormal int) {
	b.good, b.bad = complete, abnormal
	b.Refresh()
}

func (b *completenessBar) CreateRenderer() fyne.WidgetRenderer {
	r := &completenessBarRenderer{bar: b}
	return r
}

type completenessBarRenderer struct{ bar *completenessBar }

func (r *completenessBarRenderer) MinSize() fyne.Size { return fyne.NewSize(80, 8) }
func (r *completenessBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bar.background, r.bar.complete, r.bar.abnormal}
}
func (r *completenessBarRenderer) Destroy() {}
func (r *completenessBarRenderer) Refresh() { r.Layout(r.bar.Size()) }
func (r *completenessBarRenderer) Layout(size fyne.Size) {
	b := r.bar
	b.background.Resize(size)
	total := b.good + b.bad
	if total == 0 {
		b.complete.Hide()
		b.abnormal.Hide()
		return
	}
	goodWidth := size.Width * float32(b.good) / float32(total)
	b.complete.Show()
	b.complete.Resize(fyne.NewSize(goodWidth, size.Height))
	b.abnormal.Show()
	b.abnormal.Move(fyne.NewPos(goodWidth, 0))
	b.abnormal.Resize(fyne.NewSize(size.Width-goodWidth, size.Height))
}
