//go:build perf

// perfprobe opens the real desktop renderer, measures idle memory and drives
// repeated resize events. It uses an isolated configuration and writes only to -out.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	flag.Parse()
	if err := os.MkdirAll(*dir, 0755); err != nil {
		panic(err)
	}
	a := app.NewWithID("io.regexfileextractor.perfprobe")
	a.Settings().SetTheme(ui.Theme())
	c := ui.New(a, config.Default(), filepath.Join(*dir, "config.json"), nil)
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
		for i := 0; i < 120; i++ {
			fyne.DoAndWait(func() { c.Window.Resize(fyne.NewSize(float32(1000+(i%60)*5), float32(720+i%40*3))) })
			time.Sleep(time.Second / 60)
		}
		pprof.StopCPUProfile()
		cpu.Close()
		fmt.Printf("Resize sequence: %s\n", time.Since(start))
		time.Sleep(time.Second)
		snapshot(*dir, "after-resize")
		fyne.Do(func() { c.Window.SetCloseIntercept(nil); a.Quit() })
	}()
	a.Run()
}

func snapshot(dir, name string) {
	runtime.GC()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	report := map[string]any{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "heap_live_bytes": mem.HeapAlloc, "heap_sys_bytes": mem.HeapSys, "total_alloc_bytes": mem.TotalAlloc, "gc_count": mem.NumGC, "goroutines": runtime.NumGoroutine()}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), data, 0644); err != nil {
		panic(err)
	}
	f, err := os.Create(filepath.Join(dir, name+"-heap.pprof"))
	if err != nil {
		panic(err)
	}
	if err := pprof.WriteHeapProfile(f); err != nil {
		panic(err)
	}
	f.Close()
	fmt.Printf("%s: heap %.1f MiB; Go heap reserved %.1f MiB\n", name, float64(mem.HeapAlloc)/(1<<20), float64(mem.HeapSys)/(1<<20))
}
