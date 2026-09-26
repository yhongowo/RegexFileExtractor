package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CopyPhase string

const (
	CopyChecking     CopyPhase = "checking"
	CopyTransferring CopyPhase = "copying"
)

type CopyProgress struct {
	Phase   CopyPhase
	Checked int
	// BytesDone includes completed, skipped and failed entries; BytesCopied counts actual writes.
	BytesDone, BytesCopied, BytesTotal   int64
	Done, Total, Copied, Skipped, Failed int
	Path                                 string
}
type CopyResult struct {
	Copied, Skipped, Failed int
	Errors                  []string
}

// Copy only copies. It never deletes, renames or changes a source file.
// Existing outputs are replaced only after a complete, synced temporary copy.
func Copy(ctx context.Context, destination string, entries []Entry, conflict Conflict, progress func(CopyProgress)) (CopyResult, error) {
	var result CopyResult
	if conflict != Skip && conflict != Overwrite {
		return result, fmt.Errorf("unknown conflict policy: %s", conflict)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	p := CopyProgress{Phase: CopyChecking, Total: len(entries)}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		p.BytesTotal += entry.File.Size
	}
	last := time.Time{}
	report := func(force bool) {
		if progress != nil && (force || time.Since(last) >= 100*time.Millisecond) {
			progress(p)
			last = time.Now()
		}
	}
	report(true)
	if err := ctx.Err(); err != nil {
		return result, err
	}
	root, err := CanonicalPath(destination)
	if err != nil {
		return result, err
	}
	protected := map[string]bool{}
	for i, entry := range entries {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		path, err := CanonicalPath(entry.File.Path)
		if err != nil {
			return result, err
		}
		protected[strings.ToLower(filepath.Clean(path))] = true
		p.Checked = i + 1
		report(false)
		if !filepath.IsLocal(entry.Relative) {
			return result, fmt.Errorf("output path escapes destination: %s", entry.Relative)
		}
		for _, part := range strings.Split(entry.Relative, string(filepath.Separator)) {
			if err := validName(part); err != nil {
				return result, err
			}
		}
	}
	// Validate all outputs before writing anything, including targets that are other sources.
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		report(false)
		target := filepath.Join(root, entry.Relative)
		if protected[strings.ToLower(target)] {
			return result, fmt.Errorf("destination would replace a source file: %s", target)
		}
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return result, err
	}
	p.Phase = CopyTransferring
	report(true)
	var completedBytes int64
	buffer := make([]byte, 256*1024)
	for i, entry := range entries {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		p.Path = entry.File.Name
		var fileBytes int64
		target := filepath.Join(root, entry.Relative)
		err := ensureParents(root, filepath.Dir(target))
		skipped := false
		if err == nil {
			skipped, err = copyOne(ctx, entry.File, target, conflict, buffer, func(n int) {
				fileBytes += int64(n)
				p.BytesCopied += int64(n)
				p.BytesDone = completedBytes + min(fileBytes, entry.File.Size)
				report(false)
			})
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return result, err
		}
		switch {
		case err != nil:
			result.Failed++
			if len(result.Errors) < 100 {
				result.Errors = append(result.Errors, fmt.Sprintf("%s → %s: %v", entry.File.Path, target, err))
			}
		case skipped:
			result.Skipped++
		default:
			result.Copied++
		}
		completedBytes += entry.File.Size
		p.Done, p.Copied, p.Skipped, p.Failed = i+1, result.Copied, result.Skipped, result.Failed
		p.BytesDone = completedBytes
		report(i == len(entries)-1 || i == 0)
	}
	return result, nil
}

// Do not let an existing output subdirectory redirect copies through a symlink.
func ensureParents(root, dir string) error {
	rel, err := filepath.Rel(root, dir)
	if err != nil || !Within(root, dir) {
		return fmt.Errorf("invalid output directory: %s", dir)
	}
	current := root
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		if err := os.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output parent is not a regular directory: %s", current)
		}
	}
	return nil
}

func copyOne(ctx context.Context, file File, target string, conflict Conflict, buffer []byte, progress func(int)) (bool, error) {
	if info, err := os.Lstat(target); err == nil {
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("output is not a regular file: %s", target)
		}
		if conflict == Skip {
			return true, nil
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}
	info, err := os.Lstat(file.Path)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("source is no longer a regular file")
	}
	in, err := os.Open(file.Path)
	if err != nil {
		return false, err
	}
	defer in.Close()
	opened, err := in.Stat()
	if err != nil {
		return false, err
	}
	if !os.SameFile(info, opened) || opened.Size() != file.Size || !opened.ModTime().Equal(file.ModTime) {
		return false, fmt.Errorf("source changed since scanning; scan again")
	}
	if existing, err := os.Stat(target); err == nil && os.SameFile(opened, existing) {
		return false, fmt.Errorf("source and output refer to the same file")
	}
	var out *os.File
	if conflict == Skip {
		out, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			return true, nil
		}
	} else {
		out, err = os.CreateTemp(filepath.Dir(target), ".regex-extract-*")
	}
	if err != nil {
		return false, err
	}
	outputPath := out.Name()
	complete := false
	defer func() {
		out.Close()
		if !complete {
			_ = os.Remove(outputPath)
		}
	}()
	reader := contextReader{ctx, in}
	// Hide os.File.ReadFrom so io.CopyBuffer actually uses our batch buffer,
	// instead of allocating an extra transfer buffer for each small file.
	written, err := io.CopyBuffer(progressWriter{out, progress}, reader, buffer)
	if err != nil {
		return false, err
	}
	after, err := in.Stat()
	if err != nil {
		return false, err
	}
	if written != file.Size || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return false, fmt.Errorf("source changed while copying")
	}
	if err = out.Chmod(0644); err != nil {
		return false, err
	}
	if err = out.Sync(); err != nil {
		return false, err
	}
	if err = out.Close(); err != nil {
		return false, err
	}
	if err = os.Chtimes(outputPath, file.ModTime, file.ModTime); err != nil {
		return false, err
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	if conflict == Overwrite {
		if err = os.Rename(outputPath, target); err != nil {
			return false, err
		}
	}
	complete = true
	return false, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// Count successful writes, not reads, and preserve io.CopyBuffer's shared buffer.
type progressWriter struct {
	writer   io.Writer
	progress func(int)
}

func (w progressWriter) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	if n > 0 && w.progress != nil {
		w.progress(n)
	}
	return n, err
}
