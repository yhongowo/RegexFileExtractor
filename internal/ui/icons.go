package ui

import (
	"fyne.io/fyne/v2"
)

// Fyne's themed resources turn stroke-only SVGs into filled silhouettes, so
// these icons keep their colors in the SVG. All are constructed once.
func outlineIcon(name, paths, stroke string) fyne.Resource {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g fill="none" stroke="` + stroke + `" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">` + paths + `</g></svg>`
	return fyne.NewStaticResource(name+".svg", []byte(svg))
}

var (
	languageIcon    = outlineIcon("language", `<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.8 2.7 4.2 5.7 4.2 9S14.8 18.3 12 21M12 3C9.2 5.7 7.8 8.7 7.8 12s1.4 6.3 4.2 9"/>`, "#252B35")
	infoIcon        = outlineIcon("info", `<circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7h.01"/>`, "#252B35")
	settingsIcon    = outlineIcon("settings", `<circle cx="12" cy="12" r="3"/><path d="M10 2h4l.5 2.2 1.5.7 1.9-1.2 2.8 2.8-1.2 1.9.7 1.5L22 10v4l-2.2.5-.7 1.5 1.2 1.9-2.8 2.8-1.9-1.2-1.5.7L14 22h-4l-.5-2.2-1.5-.7-1.9 1.2-2.8-2.8 1.2-1.9-.7-1.5L2 14v-4l2.2-.5.7-1.5-1.2-1.9 2.8-2.8 1.9 1.2 1.5-.7L10 2z"/>`, "#252B35")
	extractionIcon  = outlineIcon("extraction", `<path d="M3 6h8m4 0h6M3 12h3m4 0h11M3 18h12m4 0h2"/><circle cx="13" cy="6" r="2"/><circle cx="8" cy="12" r="2"/><circle cx="17" cy="18" r="2"/>`, "#252B35")
	folderIcon      = outlineIcon("folder", `<path d="M3 6.5A1.5 1.5 0 0 1 4.5 5H10l2 2h7.5A1.5 1.5 0 0 1 21 8.5v9A1.5 1.5 0 0 1 19.5 19h-15A1.5 1.5 0 0 1 3 17.5z"/>`, "#252B35")
	copyIcon        = outlineIcon("copy", `<rect x="8" y="8" width="12" height="13" rx="1.5"/><path d="M16 8V4.5A1.5 1.5 0 0 0 14.5 3h-10A1.5 1.5 0 0 0 3 4.5v12A1.5 1.5 0 0 0 4.5 18H8"/>`, "#252B35")
	addIcon         = outlineIcon("add", `<path d="M12 4v16M4 12h16"/>`, "#252B35")
	editIcon        = outlineIcon("edit", `<path d="m4 17-.5 3.5L7 20l11.8-11.8-3-3L4 17zM14.5 6.5l3 3"/>`, "#252B35")
	deleteIcon      = outlineIcon("delete", `<path d="M4 6h16M9 6V4h6v2m-9 0 1 15h10l1-15M10 10v7m4-7v7"/>`, "#252B35")
	listIcon        = outlineIcon("list", `<path d="M9 5h12M9 12h12M9 19h12M4 5h.01M4 12h.01M4 19h.01"/>`, "#252B35")
	fileIcon        = outlineIcon("file", `<path d="M5 2.5h9l5 5V21H5zM14 2.5V8h5M8 12h8M8 16h6"/>`, "#252B35")
	searchIcon      = outlineIcon("search", `<circle cx="10.5" cy="10.5" r="6.5"/><path d="m15.5 15.5 5 5"/>`, "#FFFFFF")
	mutedCopyIcon   = outlineIcon("copy-muted", `<rect x="8" y="8" width="12" height="13" rx="1.5"/><path d="M16 8V4.5A1.5 1.5 0 0 0 14.5 3h-10A1.5 1.5 0 0 0 3 4.5v12A1.5 1.5 0 0 0 4.5 18H8"/>`, "#89919E")
	mutedFileIcon   = outlineIcon("file-muted", `<path d="M5 2.5h9l5 5V21H5zM14 2.5V8h5M8 12h8M8 16h6"/>`, "#89919E")
	mutedSearchIcon = outlineIcon("search-muted", `<circle cx="10.5" cy="10.5" r="6.5"/><path d="m15.5 15.5 5 5"/>`, "#89919E")
)
