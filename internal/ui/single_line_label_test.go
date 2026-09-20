package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestSingleLineLabelReusesTextAcrossResize(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(Theme())
	label := newSingleLineLabel("D:/测量数据/" + strings.Repeat("long-directory/", 30) + "X01.csv")
	r := test.WidgetRenderer(label).(*singleLineRenderer)
	originalText, originalSize := r.text.Text, r.text.Size()
	for width := float32(20); width < 1200; width++ {
		label.Resize(fyne.NewSize(width, 32))
		if r.text.Text != originalText || r.text.Size() != originalSize {
			t.Fatal("resize changed text or texture geometry")
		}
		if r.clip.Position().X+r.clip.Size().Width > width {
			t.Fatal("text clip exceeds label")
		}
		if r.ellipsis.Visible() && r.ellipsis.Position().X+r.ellipsis.Size().Width > width {
			t.Fatal("ellipsis exceeds label")
		}
	}
	label.Resize(fyne.NewSize(originalSize.Width+2*r.padding, 32))
	if r.ellipsis.Visible() {
		t.Fatal("ellipsis visible when full text fits")
	}
	if label.AccessibilityLabel() != originalText {
		t.Fatal("accessibility lost the full text")
	}
	label.SetText("new.csv")
	if r.text.Text != "new.csv" || r.ellipsis.Visible() {
		t.Fatal("recycled label retained old content")
	}
	label.SetText("")
	if r.text.Text != "" || r.ellipsis.Visible() {
		t.Fatal("empty label retained old content")
	}
}

func TestSingleLineLabelRefreshStyleAndControls(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	label := newSingleLineLabel("a\nb\rc\td")
	r := test.WidgetRenderer(label).(*singleLineRenderer)
	if r.text.Text != "a b c d" || label.AccessibilityLabel() != label.Text {
		t.Fatal("display must stay one line while preserving the accessible value")
	}
	label.TextStyle.Monospace = true
	label.Refresh()
	if !r.text.TextStyle.Monospace || !r.ellipsis.TextStyle.Monospace {
		t.Fatal("explicit refresh lost text style")
	}
}
