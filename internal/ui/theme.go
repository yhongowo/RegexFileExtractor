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
	if !style.Monospace && !style.Symbol {
		if font := uiFont(style.Bold); font != nil {
			return font
		}
	}
	return t.base.Font(style)
}
func (t *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}
func (t *appTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNameInlineIcon:
		return 16
	case theme.SizeNameText:
		return 12
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameHeadingText:
		return 15
	case theme.SizeNameSubHeadingText:
		return 14
	case theme.SizeNameInputRadius:
		return 4
	case theme.SizeNameButtonRadius:
		return 4
	case theme.SizeNameSelectionRadius:
		return 4
	}
	return t.base.Size(name)
}
func (t *appTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 246, G: 247, B: 249, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 32, G: 32, B: 32, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0, G: 95, B: 184, A: 255}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 0, G: 103, B: 192, A: 80}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 13}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 24}
	case theme.ColorNameForegroundOnPrimary:
		return color.White
	case theme.ColorNameButton, theme.ColorNameInputBackground:
		return color.White
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 137, G: 145, B: 158, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 243, G: 245, B: 248, A: 255}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 209, G: 213, B: 219, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 156, G: 163, B: 175, A: 255}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 219, G: 234, B: 254, A: 255}
	case theme.ColorNameHeaderBackground:
		return color.NRGBA{R: 249, G: 250, B: 251, A: 255}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 229, G: 231, B: 235, A: 255}
	}
	return t.base.Color(name, theme.VariantLight)
}
