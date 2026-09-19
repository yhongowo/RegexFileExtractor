//go:build windows

package ui

import (
	"fyne.io/fyne/v2"

	"regexfileextractor/assets"
)

var windowsUIFont = fyne.NewStaticResource("RFESans-Regular.otf", assets.RFESansRegular)

func uiFont(bool) fyne.Resource {
	return windowsUIFont
}
