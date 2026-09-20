//go:build perf

// perfprobe opens the real desktop renderer, measures idle memory and drives
// repeated resize events. It uses an isolated configuration and writes only to -out.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"regexfileextractor/internal/config"
	"regexfileextractor/internal/ui"
)

func main() {
	dir := flag.String("out", "dist/perf", "output directory for profiles and measurements")
	iterations := flag.Int("iterations", 120, "number of resize requests")
	interval := flag.Duration("interval", time.Second/60, "delay between resize requests; 0 for a burst")
	settle := flag.Duration("settle", 20*time.Second, "idle time after resizing")
	capture := flag.Bool("capture", false, "save a native screenshot after all memory samples")
	rows := flag.Int("rows", 0, "synthetic result count (no user files are read)")
	flag.Parse()
	if *iterations < 1 || *rows < 0 || *interval < 0 || *settle < 0 {
		panic("iterations must be positive; rows and durations must be nonnegative")
	}
	if err := os.MkdirAll(*dir, 0755); err != nil {
		panic(err)
	}
	a := app.NewWithID("io.regexfileextractor.perfprobe")
	a.Settings().SetTheme(ui.Theme())
	c := ui.New(a, config.Default(), filepath.Join(*dir, "config.json"), nil)
	c.SetPerformanceResults(*rows)
	c.Window.Show()
	go func() {
		time.Sleep(2 * time.Second)
		snapshot(*dir, "startup")
		cpu, err := os.Create(filepath.Join(*dir, "resize-cpu.pprof"))
		if err != nil {
			panic(err)
		}
		if err := pprof.StartCPUProfile(cpu); err != nil {
			panic(err)
		}
		start := time.Now()
		var samples []map[string]any
		for i := 0; i < *iterations; i++ {
			fyne.DoAndWait(func() { c.Window.Resize(fyne.NewSize(float32(1000+(i%60)*5), float32(720+i%40*3))) })
			time.Sleep(*interval)
			if i%10 == 0 {
				sample := memoryReport()
				sample["elapsed_ms"] = time.Since(start).Milliseconds()
				sample["resize_requests"] = i + 1
				samples = append(samples, sample)
			}
		}
		pprof.StopCPUProfile()
		cpu.Close()
		fmt.Printf("Resize sequence: %s\n", time.Since(start))
		snapshot(*dir, "after-resize")
		time.Sleep(*settle)
		snapshot(*dir, "after-idle")
		writeJSON(*dir, "resize-samples", map[string]any{
			"iterations": *iterations, "rows": *rows, "interval": interval.String(),
			"settle": settle.String(), "samples": samples,
		})
		if *capture {
			var img image.Image
			fyne.DoAndWait(func() { img = c.Window.Canvas().Capture() })
			f, err := os.Create(filepath.Join(*dir, "window.png"))
			if err != nil {
				panic(err)
			}
			if err := png.Encode(f, img); err != nil {
				panic(err)
			}
			if err := f.Close(); err != nil {
				panic(err)
			}
		}
		fyne.Do(func() { c.Window.SetCloseIntercept(nil); a.Quit() })
	}()
	a.Run()
}

func memoryReport() map[string]any {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	report := map[string]any{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "heap_alloc_bytes": mem.HeapAlloc, "heap_sys_bytes": mem.HeapSys, "total_alloc_bytes": mem.TotalAlloc, "gc_count": mem.NumGC, "goroutines": runtime.NumGoroutine()}
	for key, value := range processMemory() {
		report[key] = value
	}
	return report
}

func writeJSON(dir, name string, report map[string]any) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), data, 0644); err != nil {
		panic(err)
	}
}

func snapshot(dir, name string) {
	report := memoryReport()
	writeJSON(dir, name, report)
	f, err := os.Create(filepath.Join(dir, name+"-heap.pprof"))
	if err != nil {
		panic(err)
	}
	if err := pprof.WriteHeapProfile(f); err != nil {
		panic(err)
	}
	f.Close()
	// No forced GC: retain the real resize peak and natural idle recovery.
	fmt.Printf("%s: heap %.1f MiB; Go heap reserved %.1f MiB\n", name, float64(report["heap_alloc_bytes"].(uint64))/(1<<20), float64(report["heap_sys_bytes"].(uint64))/(1<<20))
}
