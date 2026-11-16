package core

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	DefaultDatalogMinKB = 61.0
	DefaultDatalogMaxKB = 65.0
)

var datalogName = regexp.MustCompile(`(?i)^X([0-9]+)Y([0-9]+)\.csv$`)

// DatalogCoordinate returns a stable die key and its display name. Numeric key
// normalization treats X1Y1.csv and X01Y01.csv as the same die.
func DatalogCoordinate(name string) (key, folder string, err error) {
	matches := datalogName.FindStringSubmatch(name)
	if matches == nil {
		return "", "", fmt.Errorf("invalid Datalog filename: %q", name)
	}
	x := strings.TrimLeft(matches[1], "0")
	y := strings.TrimLeft(matches[2], "0")
	if x == "" {
		x = "0"
	}
	if y == "" {
		y = "0"
	}
	ext := filepath.Ext(name)
	folder = strings.TrimSuffix(name, ext)
	return "x" + x + "y" + y, folder, nil
}

// IsDatalogComplete applies an inclusive KiB range. Files outside the range,
// including oversized files, are considered abnormal.
func IsDatalogComplete(size int64, minKB, maxKB float64) bool {
	kb := float64(size) / 1024
	return kb >= minKB && kb <= maxKB
}

// PlanDatalogContext writes every selected test to grr/<die>/ and only the
// first source-path-ordered test for each die to corr/<die>/. It never writes.
func PlanDatalogContext(ctx context.Context, files []File) ([]Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ordered := files
	if !sort.SliceIsSorted(files, func(i, j int) bool { return files[i].Path < files[j].Path }) {
		ordered = append([]File(nil), files...)
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	}
	type dieGroup struct {
		folder string
		next   int
	}
	groups := make(map[string]*dieGroup)
	plan := make([]Entry, 0, len(ordered)*2)
	for i, file := range ordered {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := validName(file.Name); err != nil {
			return nil, err
		}
		if i > 0 && ordered[i-1].Path == file.Path {
			return nil, fmt.Errorf("duplicate source: %s", file.Path)
		}
		key, folder, err := DatalogCoordinate(file.Name)
		if err != nil {
			return nil, err
		}
		group := groups[key]
		if group == nil {
			group = &dieGroup{folder: folder}
			groups[key] = group
			plan = append(plan, Entry{File: file, Relative: filepath.Join("corr", folder, file.Name)})
		}
		group.next++
		ext := filepath.Ext(file.Name)
		stem := strings.TrimSuffix(file.Name, ext)
		grrName := fmt.Sprintf("%s_%03d%s", stem, group.next, ext)
		plan = append(plan, Entry{File: file, Relative: filepath.Join("grr", group.folder, grrName)})
	}
	return plan, nil
}

func PlanDatalog(files []File) ([]Entry, error) {
	return PlanDatalogContext(context.Background(), files)
}
