package core

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type Layout string

const (
	Flat  Layout = "flat"
	Group Layout = "group"
	Auto  Layout = "auto"
)

type Conflict string

const (
	Skip      Conflict = "skip"
	Overwrite Conflict = "overwrite"
)

type Entry struct {
	File     File
	Relative string
}

// Plan is deterministic and never writes files. Group identity includes the extension.
// Folder and file names share one Windows-style case-insensitive namespace.
func Plan(files []File, layout Layout) ([]Entry, error) {
	if layout != Flat && layout != Group && layout != Auto {
		return nil, fmt.Errorf("unknown layout: %s", layout)
	}
	// Scans and UI selections are already sorted. Avoid copying the entire file
	// metadata slice on every checkbox change; preserve the caller's slice otherwise.
	ordered := files
	if !sort.SliceIsSorted(files, func(i, j int) bool { return files[i].Path < files[j].Path }) {
		ordered = append([]File(nil), files...)
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	}
	type groupInfo struct {
		name, folder, stem, ext string
		count, next             int
	}
	groups := make(map[string]*groupInfo)
	for i, f := range ordered {
		if err := validName(f.Name); err != nil {
			return nil, err
		}
		if i > 0 && ordered[i-1].Path == f.Path {
			return nil, fmt.Errorf("duplicate source: %s", f.Path)
		}
		key := strings.ToLower(f.Name)
		g := groups[key]
		if g == nil {
			g = &groupInfo{name: f.Name}
			groups[key] = g
		}
		g.count++
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	used := map[string]bool{}
	for key, g := range groups {
		if layout == Flat || (layout == Auto && g.count == 1) {
			used[key] = true
		}
	}
	for _, key := range keys {
		g := groups[key]
		if layout == Group || (layout == Auto && g.count > 1) {
			name := g.name
			base := strings.TrimRight(strings.TrimSuffix(name, filepath.Ext(name)), ". ")
			if base == "" {
				base = "files"
			}
			if validName(base) != nil {
				base = "_" + base
			}
			folder := base
			for n := 2; used[strings.ToLower(folder)]; n++ {
				folder = fmt.Sprintf("%s_%03d", base, n)
			}
			used[strings.ToLower(folder)] = true
			g.folder = folder
			g.ext = filepath.Ext(name)
			g.stem = strings.TrimSuffix(name, g.ext)
			if g.stem == "" {
				g.stem = "file"
			}
		} else if layout == Flat && g.count > 1 {
			name := g.name
			g.ext = filepath.Ext(name)
			g.stem = strings.TrimSuffix(name, g.ext)
			if g.stem == "" {
				g.stem = "file"
			}
		}
	}
	plan := make([]Entry, 0, len(ordered))
	for _, f := range ordered {
		key := strings.ToLower(f.Name)
		g := groups[key]
		rel := g.name // Same spelling for case-only duplicate names.
		if g.folder != "" {
			g.next++
			rel = filepath.Join(g.folder, fmt.Sprintf("%s_%03d%s", g.stem, g.next, g.ext))
		} else if layout == Flat && g.count > 1 {
			g.next++
			candidate := fmt.Sprintf("%s_%03d%s", g.stem, g.next, g.ext)
			for used[strings.ToLower(candidate)] {
				g.next++
				candidate = fmt.Sprintf("%s_%03d%s", g.stem, g.next, g.ext)
			}
			used[strings.ToLower(candidate)] = true
			rel = candidate
		}
		plan = append(plan, Entry{File: f, Relative: rel})
	}
	return plan, nil
}

func validName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `<>:"/\|?*`) || strings.TrimRight(name, ". ") != name {
		return fmt.Errorf("invalid Windows filename: %q", name)
	}
	for _, r := range name {
		if r < 32 {
			return fmt.Errorf("invalid Windows filename: %q", name)
		}
	}
	stem, _, _ := strings.Cut(name, ".")
	stem = strings.ToUpper(stem)
	reserved := stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL"
	if len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '1' && stem[3] <= '9' {
		reserved = true
	}
	if reserved {
		return fmt.Errorf("reserved Windows filename: %q", name)
	}
	return nil
}
