package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type appTheme struct {
	base fyne.Theme
}

func Theme() fyne.Theme {
	return &appTheme{base: theme.DefaultTheme()}
}
func (t *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}
func (t *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}
func (t *appTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInlineIcon:
		return 14
	case theme.SizeNameText:
		return 11
	case theme.SizeNameHeadingText:
		return 18
	case theme.SizeNameInputRadius:
		return 5
	case theme.SizeNameSelectionRadius:
		return 4
	}
	return t.base.Size(name)
}
func (t *appTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 30, G: 41, B: 59, A: 255}
	case theme.ColorNamePrimary, theme.ColorNameFocus:
		return color.NRGBA{R: 120, G: 120, B: 120, A: 255}
	case theme.ColorNameButton, theme.ColorNameInputBackground:
		return color.White
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 203, G: 203, B: 203, A: 255}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 220, G: 220, B: 220, A: 255}
	}
	return t.base.Color(name, theme.VariantLight)
}
