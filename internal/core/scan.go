package core

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type File struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
	Rule    string
}

type ScanOptions struct {
	Source  string
	Exclude string // Destination subtree, if it lives inside Source.
	Rule    Rule
}

type ScanProgress struct {
	Visited int
	Matched int
}

type ScanResult struct {
	Files        []File
	Visited      int
	Warnings     []string
	WarningCount int
}

// Scan recursively examines regular filenames. Links are never followed.
// Progress runs on the calling goroutine and is throttled for large directories.
func Scan(ctx context.Context, opts ScanOptions, progress func(ScanProgress)) (ScanResult, error) {
	var result ScanResult
	re, err := opts.Rule.Compile()
	if err != nil {
		return result, err
	}
	root, err := CanonicalPath(opts.Source)
	if err != nil {
		return result, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return result, err
	}
	if !info.IsDir() {
		return result, fmt.Errorf("source is not a directory: %s", root)
	}
	exclude := ""
	if strings.TrimSpace(opts.Exclude) != "" {
		exclude, err = CanonicalPath(opts.Exclude)
		if err != nil {
			return result, err
		}
		if SamePath(root, exclude) {
			return result, fmt.Errorf("source and destination must differ")
		}
		// A destination which contains the source must not exclude the source.
		if !Within(root, exclude) {
			exclude = ""
		}
	}
	last := time.Time{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			if path == root {
				return walkErr
			}
			result.WarningCount++
			if len(result.Warnings) < 100 {
				result.Warnings = append(result.Warnings, walkErr.Error())
			}
			return nil
		}
		if entry.IsDir() {
			if exclude != "" && SamePath(path, exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		result.Visited++
		if re.MatchString(entry.Name()) {
			info, statErr := entry.Info()
			if statErr != nil {
				result.WarningCount++
				if len(result.Warnings) < 100 {
					result.Warnings = append(result.Warnings, statErr.Error())
				}
			} else if info.Mode().IsRegular() {
				result.Files = append(result.Files, File{Path: path, Name: entry.Name(), Size: info.Size(), ModTime: info.ModTime(), Rule: opts.Rule.Name})
			}
		}
		if progress != nil && time.Since(last) >= 100*time.Millisecond {
			progress(ScanProgress{result.Visited, len(result.Files)})
			last = time.Now()
		}
		return nil
	})
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].Path < result.Files[j].Path })
	if progress != nil {
		progress(ScanProgress{result.Visited, len(result.Files)})
	}
	return result, err
}

// CanonicalPath resolves links in existing ancestors, including a not-yet-created destination.
func CanonicalPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("directory is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(abs)
	if parent == abs {
		return "", err
	}
	resolved, err = CanonicalPath(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(abs)), nil
}

func SamePath(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }

func Within(parent, child string) bool {
	parent, child = strings.ToLower(filepath.Clean(parent)), strings.ToLower(filepath.Clean(child))
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
