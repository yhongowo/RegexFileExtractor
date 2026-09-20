//go:build perf

package ui

import (
	"fmt"

	"regexfileextractor/internal/core"
)

// SetPerformanceResults seeds synthetic data for the standalone native probe.
// It is excluded from release builds and never reads or copies user files.
func (c *Controller) SetPerformanceResults(count int) {
	c.files = make([]core.File, count)
	c.selected = make([]bool, count)
	for i := range c.files {
		c.files[i] = core.File{
			Path: fmt.Sprintf("D:/measurements/production-line/batch-%05d/channel-01/X%04d.csv", i, i%100),
			Name: fmt.Sprintf("X%04d.csv", i%100), Size: 1024, Rule: "XY CSV",
		}
		c.selected[i] = true
	}
	c.refreshPlan()
}
