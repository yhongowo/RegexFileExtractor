package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// singleLineLabel keeps the same text texture at every window width. RichText
// ellipsis truncation reshapes the string on each resize and caches a different
// GPU texture for every visible prefix. These labels only change a clip region.
// Full values stay available to accessibility and the file details dialog.
type singleLineLabel struct {
	widget.BaseWidget
	Text      string
	TextStyle fyne.TextStyle
}

func newSingleLineLabel(text string) *singleLineLabel {
	l := &singleLineLabel{Text: text}
	l.ExtendBaseWidget(l)
	return l
}

func (l *singleLineLabel) SetText(text string) {
	if l.Text == text {
		return
	}
	l.Text = text
	l.Refresh()
}

func (l *singleLineLabel) AccessibilityLabel() string           { return l.Text }
func (*singleLineLabel) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleText }

func (l *singleLineLabel) CreateRenderer() fyne.WidgetRenderer {
	text := canvas.NewText("", theme.ForegroundColor())
	ellipsis := canvas.NewText("…", theme.ForegroundColor())
	// Keep text at its measured size: changing the clip must not resize or
	// refresh text, including while the viewport is wider than its contents.
	clip := container.NewClip(container.NewWithoutLayout(text))
	r := &singleLineRenderer{label: l, text: text, ellipsis: ellipsis, clip: clip,
		objects: []fyne.CanvasObject{clip, ellipsis}}
	r.Refresh()
	return r
}

type singleLineRenderer struct {
	label                  *singleLineLabel
	text, ellipsis         *canvas.Text
	clip                   *container.Clip
	objects                []fyne.CanvasObject
	textSize, ellipsisSize fyne.Size
	padding, lineHeight    float32
}

func (r *singleLineRenderer) Destroy()                     {}
func (r *singleLineRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *singleLineRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.ellipsisSize.Width+2*r.padding, r.lineHeight+2*r.padding)
}
func (r *singleLineRenderer) Refresh() {
	th := r.label.Theme()
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	r.padding = th.Size(theme.SizeNameInnerPadding)
	for _, text := range []*canvas.Text{r.text, r.ellipsis} {
		text.TextSize = th.Size(theme.SizeNameText)
		text.TextStyle = r.label.TextStyle
		text.FontSource = th.Font(r.label.TextStyle)
		text.Color = th.Color(theme.ColorNameForeground, variant)
	}
	// Filenames and status strings remain one line even if they contain controls.
	r.text.Text = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(r.label.Text)
	r.textSize, r.ellipsisSize = r.text.MinSize(), r.ellipsis.MinSize()
	r.lineHeight = max(r.textSize.Height, r.ellipsisSize.Height)
	r.text.Resize(r.textSize)
	r.ellipsis.Resize(r.ellipsisSize)
	r.text.Refresh()
	r.ellipsis.Refresh()
	r.Layout(r.label.Size())
	canvas.Refresh(r.label)
}
func (r *singleLineRenderer) Layout(size fyne.Size) {
	width := max(float32(0), size.Width-2*r.padding)
	y := max(float32(0), (size.Height-r.lineHeight)/2)
	if r.textSize.Width > width && width >= r.ellipsisSize.Width {
		width -= r.ellipsisSize.Width
		r.ellipsis.Move(fyne.NewPos(r.padding+width, y))
		r.ellipsis.Show()
	} else {
		r.ellipsis.Hide()
	}
	r.clip.Move(fyne.NewPos(r.padding, y))
	r.clip.Resize(fyne.NewSize(width, min(r.lineHeight, max(float32(0), size.Height-y))))
}
