package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func put(t *testing.T, root, relative, content string) File {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return File{Path: path, Name: info.Name(), Size: info.Size(), ModTime: info.ModTime(), Rule: "test"}
}
func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRuleFullMatch(t *testing.T) {
	tests := []struct {
		pattern, name string
		want          bool
	}{
		{`X\d+Y\d+\.csv`, "X01Y01.csv", true},
		{`X\d+Y\d+\.csv`, "prefix_X01Y01.csv", false},
		{`X\d+Y\d+\.csv`, "X01Y01.csv.bak", false},
		{`X\d+Y\d+\.csv`, "X01Y01.CSV", false},
		{`(?i)X\d+Y\d+\.csv`, "x01y01.CSV", true},
		{`a|ab`, "ab", true}, {`a|b`, "abc", false},
		{`(?m)^a$`, "a\nb", false}, {`(?m)^a$`, "a\n", false},
		{`数据\.csv`, "数据.csv", true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.name, func(t *testing.T) {
			re, err := (Rule{Name: "test", Pattern: tt.pattern}).Compile()
			if err != nil {
				t.Fatal(err)
			}
			if got := re.MatchString(tt.name); got != tt.want {
				t.Fatalf("matched %q = %v", tt.name, got)
			}
		})
	}
	for _, rule := range []Rule{{Name: "test", Pattern: "["}, {Name: "", Pattern: ".*"}, {Name: "x", Pattern: ""}} {
		if _, err := rule.Compile(); err == nil {
			t.Fatalf("accepted invalid rule: %+v", rule)
		}
	}
}

func TestScanRecursiveStrictAndExclude(t *testing.T) {
	root := t.TempDir()
	a := put(t, root, "a/X01Y01.csv", "a")
	b := put(t, root, "b/deep/X02Y02.csv", "bb")
	put(t, root, "prefix_X01Y01.csv", "no")
	put(t, root, "X03Y03.csv.bak", "no")
	put(t, root, "output/X04Y04.csv", "exclude")
	put(t, root, "Log.txt", "no")
	if err := os.Mkdir(filepath.Join(root, "X99Y99.csv"), 0755); err != nil {
		t.Fatal(err)
	}
	var last ScanProgress
	result, err := Scan(context.Background(), ScanOptions{Source: root, Exclude: filepath.Join(root, "output"), Rule: Rule{Name: "XY", Pattern: `X\d+Y\d+\.csv`}}, func(p ScanProgress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	canonicalA, _ := CanonicalPath(a.Path)
	canonicalB, _ := CanonicalPath(b.Path)
	if len(result.Files) != 2 || result.Files[0].Path != canonicalA || result.Files[1].Path != canonicalB {
		t.Fatalf("wrong matches: %+v", result.Files)
	}
	if result.Visited != 5 || last.Matched != 2 || last.Visited != 5 {
		t.Fatalf("wrong progress: %+v %+v", result, last)
	}
	if result.Files[0].Rule != "XY" {
		t.Fatal("matched rule omitted")
	}
	_, err = Scan(context.Background(), ScanOptions{Source: root, Exclude: root, Rule: Rule{Name: "x", Pattern: ".*"}}, nil)
	if err == nil {
		t.Fatal("allowed source=destination")
	}
}
func TestScanCancellationAndLinks(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	put(t, external, "external.csv", "outside")
	put(t, root, "inside.csv", "inside")
	if runtime.GOOS != "windows" {
		if err := os.Symlink(external, filepath.Join(root, "link")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(external, "external.csv"), filepath.Join(root, "linked.csv")); err != nil {
			t.Fatal(err)
		}
	}
	opts := ScanOptions{Source: root, Rule: Rule{Name: "csv", Pattern: `.*\.csv`}}
	var progresses []ScanProgress
	result, err := Scan(context.Background(), opts, func(progress ScanProgress) {
		progresses = append(progresses, progress)
	})
	if err != nil || len(result.Files) != 1 {
		t.Fatalf("followed a link: %+v, %v", result, err)
	}
	if len(progresses) != 1 || progresses[0].Visited != 1 || progresses[0].Matched != 1 {
		t.Fatalf("unexpected progress updates: %+v", progresses)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Scan(ctx, opts, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestLayouts(t *testing.T) {
	files := []File{{Path: "c/X02Y02.csv", Name: "X02Y02.csv"}, {Path: "b/X01Y01.csv", Name: "X01Y01.csv"}, {Path: "a/X01Y01.csv", Name: "X01Y01.csv"}}
	for _, tt := range []struct {
		layout Layout
		want   []string
	}{
		{Flat, []string{"X01Y01_001.csv", "X01Y01_002.csv", "X02Y02.csv"}},
		{Group, []string{"X01Y01/X01Y01_001.csv", "X01Y01/X01Y01_002.csv", "X02Y02/X02Y02_001.csv"}},
		{Auto, []string{"X01Y01/X01Y01_001.csv", "X01Y01/X01Y01_002.csv", "X02Y02.csv"}},
	} {
		t.Run(string(tt.layout), func(t *testing.T) {
			plan, err := Plan(files, tt.layout)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, entry := range plan {
				got = append(got, filepath.ToSlash(entry.Relative))
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if plan[0].File.Path != "a/X01Y01.csv" {
				t.Fatal("not ordered by source path")
			}
		})
	}
	if files[0].Path != "c/X02Y02.csv" {
		t.Fatal("mutated input")
	}
}
func TestLayoutNamespacesAndCase(t *testing.T) {
	files := []File{{Path: "1", Name: "X.csv"}, {Path: "2", Name: "x.CSV"}, {Path: "3", Name: "X.txt"}, {Path: "4", Name: "X.txt"}, {Path: "5", Name: "X"}}
	plan, err := Plan(files, Auto)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"X_002/X_001.csv", "X_002/X_002.csv", "X_003/X_001.txt", "X_003/X_002.txt", "X"}
	for i, e := range plan {
		if filepath.ToSlash(e.Relative) != want[i] {
			t.Fatalf("%d: got %s want %s", i, e.Relative, want[i])
		}
	}
	plan, err = Plan([]File{{Path: "1", Name: ".csv"}}, Group)
	if err != nil || filepath.ToSlash(plan[0].Relative) != "files/file_001.csv" {
		t.Fatalf("dotfile: %v %v", plan, err)
	}
	for _, name := range []string{"../bad.csv", "NUL.txt", "nul.txt", "com1.log", "COM¹.txt", "lpt².log", "LPT³", "a\\b.csv", "a.", "CON", "bad:name.csv"} {
		if _, err := Plan([]File{{Path: "a", Name: name}}, Flat); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestCopySkipAndOverwrite(t *testing.T) {
	for _, policy := range []Conflict{Skip, Overwrite} {
		t.Run(string(policy), func(t *testing.T) {
			src, dst := t.TempDir(), t.TempDir()
			a := put(t, src, "a/X.csv", "first")
			b := put(t, src, "b/X.csv", "second")
			plan, _ := Plan([]File{b, a}, Flat)
			if err := os.WriteFile(filepath.Join(dst, "X_001.csv"), []byte("old"), 0644); err != nil {
				t.Fatal(err)
			}
			result, err := Copy(context.Background(), dst, plan, policy, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := "old"
			copied, skipped := 1, 1
			if policy == Overwrite {
				want = "first"
				copied = 2
				skipped = 0
			}
			if result.Copied != copied || result.Skipped != skipped || result.Failed != 0 {
				t.Fatalf("%+v", result)
			}
			if got := read(t, filepath.Join(dst, "X_001.csv")); got != want {
				t.Fatalf("got %s want %s", got, want)
			}
			if got := read(t, filepath.Join(dst, "X_002.csv")); got != "second" {
				t.Fatalf("got %s want second", got)
			}
			if read(t, a.Path) != "first" || read(t, b.Path) != "second" {
				t.Fatal("source modified")
			}
			result, err = Copy(context.Background(), dst, plan, Skip, nil)
			if err != nil || result.Skipped != 2 {
				t.Fatalf("repeat skip: %+v %v", result, err)
			}
		})
	}
}
func TestWorkflowAuto(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	put(t, src, "a/X01Y01.csv", "one")
	put(t, src, "b/X01Y01.csv", "two")
	put(t, src, "b/X02Y02.csv", "unique")
	put(t, src, "b/ignore.txt", "ignore")
	scan, err := Scan(context.Background(), ScanOptions{Source: src, Rule: Rule{Name: "XY", Pattern: `X\d+Y\d+\.csv`}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(scan.Files, Auto)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Copy(context.Background(), dst, plan, Skip, nil)
	if err != nil || result.Copied != 3 {
		t.Fatalf("%+v %v", result, err)
	}
	for path, want := range map[string]string{"X01Y01/X01Y01_001.csv": "one", "X01Y01/X01Y01_002.csv": "two", "X02Y02.csv": "unique"} {
		if got := read(t, filepath.Join(dst, path)); got != want {
			t.Fatalf("%s: %s", path, got)
		}
	}
}
func TestCopyPreservesOutputOnChangedSource(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	file := put(t, src, "X.csv", "before")
	put(t, dst, "X.csv", "original target")
	put(t, src, "X.csv", "changed after preview")
	plan, _ := Plan([]File{file}, Flat)
	result, err := Copy(context.Background(), dst, plan, Overwrite, nil)
	if err != nil || result.Failed != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if read(t, filepath.Join(dst, "X.csv")) != "original target" {
		t.Fatal("failed copy damaged target")
	}
}
func TestCopyCancellation(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	a := put(t, src, "a.csv", "a")
	b := put(t, src, "b.csv", "b")
	put(t, dst, "b.csv", "original")
	plan, _ := Plan([]File{a, b}, Flat)
	ctx, cancel := context.WithCancel(context.Background())
	result, err := Copy(ctx, dst, plan, Overwrite, func(p CopyProgress) {
		if p.Done == 1 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) || result.Copied != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if read(t, filepath.Join(dst, "b.csv")) != "original" {
		t.Fatal("cancel overwrote next file")
	}
	entries, _ := os.ReadDir(dst)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".regex-extract-") {
			t.Fatal("temporary file leaked")
		}
	}
	notCreated := filepath.Join(dst, "not-created")
	_, err = Copy(ctx, notCreated, plan, Skip, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err = os.Stat(notCreated); !os.IsNotExist(err) {
		t.Fatal("cancelled copy created output")
	}
}
func TestCopyRejectsSourceAndEscapes(t *testing.T) {
	src := t.TempDir()
	a := put(t, src, "X.csv", "keep")
	plan, _ := Plan([]File{a}, Flat)
	if _, err := Copy(context.Background(), src, plan, Overwrite, nil); err == nil {
		t.Fatal("allowed overwriting source")
	}
	if read(t, a.Path) != "keep" {
		t.Fatal("source damaged")
	}
	plan[0].Relative = "../escaped.csv"
	if _, err := Copy(context.Background(), t.TempDir(), plan, Skip, nil); err == nil {
		t.Fatal("accepted traversal")
	}
}
func TestCopyRejectsOutputLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating links requires Windows Developer Mode")
	}
	src, dst, outside := t.TempDir(), t.TempDir(), t.TempDir()
	a := put(t, src, "X.csv", "new")
	target := put(t, outside, "X.csv", "keep")
	if err := os.Symlink(target.Path, filepath.Join(dst, "X.csv")); err != nil {
		t.Fatal(err)
	}
	plan, _ := Plan([]File{a}, Flat)
	result, err := Copy(context.Background(), dst, plan, Overwrite, nil)
	if err != nil || result.Failed != 1 || read(t, target.Path) != "keep" {
		t.Fatalf("unsafe leaf link: %+v %v", result, err)
	}
	groupDst := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(groupDst, "X")); err != nil {
		t.Fatal(err)
	}
	plan, _ = Plan([]File{a}, Group)
	result, err = Copy(context.Background(), groupDst, plan, Overwrite, nil)
	if err != nil || result.Failed != 1 {
		t.Fatalf("unsafe directory link: %+v %v", result, err)
	}
}

func TestCopyPreflightCancellationDoesNotWrite(t *testing.T) {
	src := t.TempDir()
	file := put(t, src, "a.csv", "a")
	dst := filepath.Join(t.TempDir(), "not-created")
	plan, _ := Plan([]File{file}, Flat)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result, err := Copy(ctx, dst, plan, Skip, func(p CopyProgress) {
		if p.Phase != CopyChecking || p.Done != 0 {
			t.Fatal("missing preflight phase")
		}
		cancel()
	})
	if !errors.Is(err, context.Canceled) || result.Copied != 0 {
		t.Fatalf("%+v %v", result, err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("preflight cancellation created destination")
	}
}

func TestCopyByteProgressAndSkippedFiles(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	a := put(t, src, "a.csv", strings.Repeat("a", 600000))
	b := put(t, src, "b.csv", "skip")
	put(t, dst, "b.csv", "existing")
	plan, _ := Plan([]File{a, b}, Flat)
	var updates []CopyProgress
	result, err := Copy(context.Background(), dst, plan, Skip, func(p CopyProgress) { updates = append(updates, p) })
	if err != nil || result.Copied != 1 || result.Skipped != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if updates[0].Phase != CopyChecking {
		t.Fatal("missing checking phase")
	}
	last := updates[len(updates)-1]
	if last.Phase != CopyTransferring || last.Done != 2 || last.BytesDone != a.Size+b.Size || last.BytesTotal != last.BytesDone || last.BytesCopied != a.Size {
		t.Fatalf("bad final byte accounting: %+v", last)
	}
	var previous int64
	for _, p := range updates {
		if p.BytesDone < previous || p.BytesDone > p.BytesTotal {
			t.Fatalf("non-monotonic progress: %+v", p)
		}
		previous = p.BytesDone
	}
}

func TestCopyOneMidFileCancellationPreservesTarget(t *testing.T) {
	for _, policy := range []Conflict{Skip, Overwrite} {
		t.Run(string(policy), func(t *testing.T) {
			src, dst := t.TempDir(), t.TempDir()
			file := put(t, src, "a.csv", strings.Repeat("a", 1024*1024))
			target := filepath.Join(dst, "a.csv")
			if policy == Overwrite {
				put(t, dst, "a.csv", "original")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			written := 0
			_, err := copyOne(ctx, file, target, policy, make([]byte, 256*1024), func(n int) { written += n; cancel() })
			if !errors.Is(err, context.Canceled) || written != 256*1024 {
				t.Fatalf("written=%d err=%v", written, err)
			}
			if policy == Overwrite {
				if read(t, target) != "original" {
					t.Fatal("cancellation replaced target")
				}
			} else if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatal("partial output retained")
			}
			entries, _ := os.ReadDir(dst)
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".regex-extract-") {
					t.Fatal("temporary file retained")
				}
			}
		})
	}
}

func TestPlanContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PlanContext(ctx, []File{{Path: "a", Name: "a.csv"}}, Auto); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
