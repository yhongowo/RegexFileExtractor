package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"regexfileextractor/internal/config"
	"regexfileextractor/internal/ui"
)

func main() {
	a := app.NewWithID("io.regexfileextractor.desktop")
	a.Settings().SetTheme(ui.Theme())
	a.SetIcon(theme.FileIcon())
	path, err := config.Path()
	cfg := config.Default()
	if err == nil {
		cfg, err = config.Load(path)
	}
	controller := ui.New(a, cfg, path, err)
	controller.Window.Show()
	controller.ShowLoadError()
	a.Run()
}
