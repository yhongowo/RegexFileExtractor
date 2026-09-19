//go:build !windows || !cgo

package ui

import "fyne.io/fyne/v2"

func showNativeFolderPicker(fyne.Window, string, func(string, error)) bool { return false }
